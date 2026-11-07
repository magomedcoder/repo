package git

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"magomedcoder.ru/repo/internal/domain"
)

func (r *Repository) Compare(ctx context.Context, repoPath, base, head string) (*domain.CompareResult, error) {
	return r.withTimeout(ctx, func(ctx context.Context) (*domain.CompareResult, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		return r.compare(ctx, repoPath, base, head)
	})
}

func (r *Repository) compare(ctx context.Context, repoPath, base, head string) (*domain.CompareResult, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRefNotFound, err)
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRefNotFound, err)
	}

	out := &domain.CompareResult{
		BaseSHA: baseCommit.Hash.String(),
		HeadSHA: headCommit.Hash.String(),
		Commits: []domain.CommitInfo{},
		Files:   []domain.FileDiff{},
	}

	ff, err := r.canFastForward(ctx, repoPath, base, head)
	if err != nil {
		return nil, err
	}

	out.CanFastForward = ff

	commits, err := r.logRange(ctx, repoPath, base, head)
	if err != nil {
		return nil, err
	}

	out.Commits = commits

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, err
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, err
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, err
	}

	patchBytes := 0
	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if len(out.Files) >= maxDiffFiles {
			break
		}

		fd, ok := fileDiffFromChange(ctx, change)
		if !ok {
			continue
		}

		if patchBytes+len(fd.Patch) > maxComparePatchBytes {
			fd.Patch = ""
		}
		patchBytes += len(fd.Patch)
		out.Files = append(out.Files, fd)
	}

	return out, nil
}

func (r *Repository) logRange(ctx context.Context, repoPath, base, head string) ([]domain.CommitInfo, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return nil, err
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return nil, err
	}

	excluded, err := ancestorSet(ctx, baseCommit)
	if err != nil {
		return nil, err
	}

	var out []domain.CommitInfo
	err = walkCommits(ctx, headCommit, func(c *object.Commit) bool {
		if len(out) >= maxCompareCommits {
			return false
		}

		if excluded[c.Hash] {
			return false
		}

		out = append(out, commitToInfo(c))
		return true
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (r *Repository) canFastForward(ctx context.Context, repoPath, base, head string) (bool, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return false, err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return false, err
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return false, err
	}

	seen, err := ancestorSet(ctx, headCommit)
	if err != nil {
		return false, err
	}

	return seen[baseCommit.Hash], nil
}

func (r *Repository) Merge(ctx context.Context, repoPath, base, head, strategy, authorName, authorEmail, message string, protect bool) (string, error) {
	return r.withTimeout(ctx, func(ctx context.Context) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		unlock := r.LockRepo(repoPath)
		defer unlock()

		return r.merge(ctx, repoPath, base, head, strategy, authorName, authorEmail, message, protect)
	})
}

func (r *Repository) merge(ctx context.Context, repoPath, base, head, strategy, authorName, authorEmail, message string, protect bool) (string, error) {
	base = strings.TrimSpace(base)
	head = strings.TrimSpace(head)
	if base == "" || head == "" {
		return "", fmt.Errorf("base and head required")
	}

	if base == head {
		return "", fmt.Errorf("base and head are the same")
	}

	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		strategy = "merge"
	}

	if strategy != "merge" && strategy != "ff-only" {
		return "", fmt.Errorf("unsupported merge strategy")
	}

	if authorName == "" {
		authorName = "repo"
	}

	if authorEmail == "" {
		authorEmail = "repo@localhost"
	}

	if message == "" {
		message = fmt.Sprintf("Merge branch '%s' into %s", head, base)
	}

	ff, err := r.canFastForward(ctx, repoPath, base, head)
	if err != nil {
		return "", err
	}

	if strategy == "ff-only" {
		if !ff {
			return "", ErrNotFastForward
		}

		return r.fastForward(repoPath, base, head, protect)
	}

	if ff {
		return r.fastForward(repoPath, base, head, protect)
	}

	return r.mergeTrees(ctx, repoPath, base, head, authorName, authorEmail, message, protect)
}

func (r *Repository) fastForward(repoPath, base, head string, protect bool) (string, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return "", err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return "", err
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return "", err
	}

	refName := plumbing.ReferenceName("refs/heads/" + base)
	oldRef, err := repo.Storer.Reference(refName)
	if err != nil {
		return "", fmt.Errorf("fast-forward base: %w", err)
	}

	if oldRef.Hash() != baseCommit.Hash {
		return "", fmt.Errorf("fast-forward: base moved")
	}

	if err := domain.CheckRefUpdate(protect, refName.String(), oldRef.Hash().String(), headCommit.Hash.String(), true); err != nil {
		return "", err
	}

	newRef := plumbing.NewHashReference(refName, headCommit.Hash)
	if err := repo.Storer.CheckAndSetReference(newRef, oldRef); err != nil {
		return "", fmt.Errorf("fast-forward: %w", err)
	}

	return headCommit.Hash.String(), nil
}

func (r *Repository) mergeTrees(ctx context.Context, repoPath, base, head, authorName, authorEmail, message string, protect bool) (string, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return "", err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return "", err
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return "", err
	}

	mergeBase, err := mergeBaseCommit(ctx, baseCommit, headCommit)
	if err != nil {
		return "", err
	}

	baseFiles, err := flattenTree(baseCommit)
	if err != nil {
		return "", err
	}

	headFiles, err := flattenTree(headCommit)
	if err != nil {
		return "", err
	}

	ancestorFiles, err := flattenTree(mergeBase)
	if err != nil {
		return "", err
	}

	merged := map[string]object.TreeEntry{}
	maps.Copy(merged, baseFiles)

	paths := map[string]struct{}{}
	for path := range baseFiles {
		paths[path] = struct{}{}
	}
	for path := range headFiles {
		paths[path] = struct{}{}
	}
	for path := range ancestorFiles {
		paths[path] = struct{}{}
	}

	for path := range paths {
		a, aok := ancestorFiles[path]
		b, bok := baseFiles[path]
		h, hok := headFiles[path]
		switch {
		case sameEntry(b, bok, h, hok):
			continue
		case sameEntry(a, aok, b, bok):
			if hok {
				merged[path] = h
			} else {
				delete(merged, path)
			}
		case sameEntry(a, aok, h, hok):
			continue
		default:
			return "", ErrMergeConflict
		}
	}

	treeHash, err := writeTree(repo, merged)
	if err != nil {
		return "", err
	}

	when := time.Now()
	sig := object.Signature{Name: authorName, Email: authorEmail, When: when}
	commit := &object.Commit{
		Author:       sig,
		Committer:    sig,
		Message:      message + "\n",
		TreeHash:     treeHash,
		ParentHashes: []plumbing.Hash{baseCommit.Hash, headCommit.Hash},
	}
	obj := repo.Storer.NewEncodedObject()
	if err := commit.Encode(obj); err != nil {
		return "", err
	}

	hash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", err
	}

	refName := plumbing.ReferenceName("refs/heads/" + base)
	oldRef, err := repo.Storer.Reference(refName)
	if err != nil {
		return "", fmt.Errorf("merge base ref: %w", err)
	}

	if oldRef.Hash() != baseCommit.Hash {
		return "", fmt.Errorf("merge: base moved")
	}

	if err := domain.CheckRefUpdate(protect, refName.String(), oldRef.Hash().String(), hash.String(), true); err != nil {
		return "", err
	}

	newRef := plumbing.NewHashReference(refName, hash)
	if err := repo.Storer.CheckAndSetReference(newRef, oldRef); err != nil {
		return "", fmt.Errorf("merge update ref: %w", err)
	}

	return hash.String(), nil
}

func ancestorSet(ctx context.Context, c *object.Commit) (map[plumbing.Hash]bool, error) {
	seen := map[plumbing.Hash]bool{}
	err := walkCommits(ctx, c, func(commit *object.Commit) bool {
		seen[commit.Hash] = true
		return true
	})

	return seen, err
}

func walkCommits(ctx context.Context, c *object.Commit, visit func(*object.Commit) bool) error {
	stack := []*object.Commit{c}
	seen := map[plumbing.Hash]bool{}
	for len(stack) > 0 {
		if err := checkCtx(ctx); err != nil {
			return err
		}

		if len(seen) >= maxWalkCommits {
			return ErrHistoryTooLarge
		}

		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[cur.Hash] {
			continue
		}

		seen[cur.Hash] = true
		if !visit(cur) {
			continue
		}

		parents := cur.Parents()
		err := parents.ForEach(func(parent *object.Commit) error {
			stack = append(stack, parent)
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func mergeBaseCommit(ctx context.Context, a, b *object.Commit) (*object.Commit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bases, err := a.MergeBase(b)
	if err != nil {
		return nil, err
	}

	if len(bases) == 0 || bases[0] == nil {
		return nil, fmt.Errorf("merge base not found")
	}

	return bases[0], nil
}

func flattenTree(c *object.Commit) (map[string]object.TreeEntry, error) {
	tree, err := c.Tree()
	if err != nil {
		return nil, err
	}

	out := map[string]object.TreeEntry{}
	walker := object.NewTreeWalker(tree, true, nil)
	defer walker.Close()
	for {
		name, entry, err := walker.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if entry.Mode == filemode.Dir {
			continue
		}

		out[name] = entry
	}

	return out, nil
}

func sameEntry(a object.TreeEntry, aok bool, b object.TreeEntry, bok bool) bool {
	if aok != bok {
		return false
	}

	if !aok {
		return true
	}

	return a.Hash == b.Hash && a.Mode == b.Mode
}

func writeTree(repo *gogit.Repository, files map[string]object.TreeEntry) (plumbing.Hash, error) {
	type dir struct {
		files map[string]object.TreeEntry
		dirs  map[string]*dir
	}

	root := &dir{files: map[string]object.TreeEntry{}, dirs: map[string]*dir{}}
	for path, entry := range files {
		dirPath, name, ok := strings.CutLast(path, "/")
		if !ok {
			name = path
			dirPath = ""
		}
		cur := root
		if dirPath != "" {
			for part := range strings.SplitSeq(dirPath, "/") {
				next, found := cur.dirs[part]
				if !found {
					next = &dir{files: map[string]object.TreeEntry{}, dirs: map[string]*dir{}}
					cur.dirs[part] = next
				}
				cur = next
			}
		}

		entry.Name = name
		cur.files[entry.Name] = entry
	}

	var store func(*dir) (plumbing.Hash, error)
	store = func(d *dir) (plumbing.Hash, error) {
		var entries []object.TreeEntry
		for name, file := range d.files {
			file.Name = name
			entries = append(entries, file)
		}

		for name, child := range d.dirs {
			hash, err := store(child)
			if err != nil {
				return plumbing.ZeroHash, err
			}
			entries = append(entries, object.TreeEntry{
				Name: name,
				Mode: filemode.Dir,
				Hash: hash,
			})
		}

		slices.SortFunc(entries, func(a, b object.TreeEntry) int {
			return cmp.Compare(a.Name, b.Name)
		})
		tree := &object.Tree{Entries: entries}
		obj := repo.Storer.NewEncodedObject()
		if err := tree.Encode(obj); err != nil {
			return plumbing.ZeroHash, err
		}

		return repo.Storer.SetEncodedObject(obj)
	}

	return store(root)
}

func fileDiffFromChange(ctx context.Context, change *object.Change) (domain.FileDiff, bool) {
	action, err := change.Action()
	if err != nil {
		return domain.FileDiff{}, false
	}

	fd := domain.FileDiff{}
	switch action.String() {
	case "Insert":
		fd.Status = "added"
		fd.Path = change.To.Name
	case "Delete":
		fd.Status = "deleted"
		fd.Path = change.From.Name
	case "Modify":
		fd.Status = "modified"
		fd.Path = change.To.Name
		if change.From.Name != "" && change.From.Name != change.To.Name {
			fd.Status = "renamed"
			fd.OldPath = change.From.Name
		}
	default:
		fd.Status = strings.ToLower(action.String())
		if change.To.Name != "" {
			fd.Path = change.To.Name
		} else {
			fd.Path = change.From.Name
		}
	}

	if patch, err := change.PatchContext(ctx); err == nil {
		text := patch.String()
		if len(text) > maxDiffPatchBytes {
			text = text[:maxDiffPatchBytes]
		}

		fd.Patch = text
	}

	return fd, true
}
