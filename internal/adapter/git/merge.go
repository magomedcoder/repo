package git

import (
	"fmt"
	"maps"
	"sort"
	"strings"
	"time"

	"git.magomedcoder.ru/repo/internal/domain"
	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func (r *Repository) Compare(repoPath, base, head string) (*domain.CompareResult, error) {
	return withGitTimeout(func() (*domain.CompareResult, error) {
		return r.compare(repoPath, base, head)
	})
}

func (r *Repository) compare(repoPath, base, head string) (*domain.CompareResult, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return nil, fmt.Errorf("base ref not found: %w", err)
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return nil, fmt.Errorf("head ref not found: %w", err)
	}

	out := &domain.CompareResult{
		BaseSHA: baseCommit.Hash.String(),
		HeadSHA: headCommit.Hash.String(),
		Commits: []domain.CommitInfo{},
		Files:   []domain.FileDiff{},
	}

	ff, err := r.CanFastForward(repoPath, base, head)
	if err != nil {
		return nil, err
	}

	out.CanFastForward = ff

	commits, err := r.logRange(repoPath, base, head)
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

	for _, change := range changes {
		if fd, ok := fileDiffFromChange(change); ok {
			out.Files = append(out.Files, fd)
		}
	}

	return out, nil
}

func (r *Repository) logRange(repoPath, base, head string) ([]domain.CommitInfo, error) {
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

	excluded, err := ancestorSet(baseCommit)
	if err != nil {
		return nil, err
	}

	var out []domain.CommitInfo
	err = walkCommits(headCommit, func(c *object.Commit) bool {
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

func (r *Repository) CanFastForward(repoPath, base, head string) (bool, error) {
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

	seen, err := ancestorSet(headCommit)
	if err != nil {
		return false, err
	}

	return seen[baseCommit.Hash], nil
}

func (r *Repository) Merge(repoPath, base, head, strategy, authorName, authorEmail, message string) (string, error) {
	return withGitTimeout(func() (string, error) {
		return r.merge(repoPath, base, head, strategy, authorName, authorEmail, message)
	})
}

func (r *Repository) merge(repoPath, base, head, strategy, authorName, authorEmail, message string) (string, error) {
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

	ff, err := r.CanFastForward(repoPath, base, head)
	if err != nil {
		return "", err
	}

	if strategy == "ff-only" {
		if !ff {
			return "", fmt.Errorf("not fast-forward")
		}

		return r.fastForward(repoPath, base, head)
	}

	if ff {
		return r.fastForward(repoPath, base, head)
	}

	return r.mergeTrees(repoPath, base, head, authorName, authorEmail, message)
}

func (r *Repository) fastForward(repoPath, base, head string) (string, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return "", err
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return "", err
	}

	refName := plumbing.ReferenceName("refs/heads/" + base)
	if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, headCommit.Hash)); err != nil {
		return "", fmt.Errorf("fast-forward: %w", err)
	}

	return headCommit.Hash.String(), nil
}

func (r *Repository) mergeTrees(repoPath, base, head, authorName, authorEmail, message string) (string, error) {
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

	mergeBase, err := mergeBaseCommit(baseCommit, headCommit)
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
			return "", fmt.Errorf("merge conflict")
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
	if err := repo.Storer.SetReference(plumbing.NewHashReference(refName, hash)); err != nil {
		return "", err
	}

	return hash.String(), nil
}

func ancestorSet(c *object.Commit) (map[plumbing.Hash]bool, error) {
	seen := map[plumbing.Hash]bool{}
	err := walkCommits(c, func(commit *object.Commit) bool {
		seen[commit.Hash] = true
		return true
	})
	return seen, err
}

func walkCommits(c *object.Commit, visit func(*object.Commit) bool) error {
	stack := []*object.Commit{c}
	seen := map[plumbing.Hash]bool{}
	for len(stack) > 0 {
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

func mergeBaseCommit(a, b *object.Commit) (*object.Commit, error) {
	seen, err := ancestorSet(a)
	if err != nil {
		return nil, err
	}

	var found *object.Commit
	err = walkCommits(b, func(c *object.Commit) bool {
		if seen[c.Hash] && found == nil {
			found = c
			return false
		}
		return found == nil
	})
	if err != nil {
		return nil, err
	}

	if found == nil {
		return nil, fmt.Errorf("merge base not found")
	}

	return found, nil
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
		if err != nil {
			break
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
		parts := strings.Split(path, "/")
		cur := root
		for _, part := range parts[:len(parts)-1] {
			next, ok := cur.dirs[part]
			if !ok {
				next = &dir{files: map[string]object.TreeEntry{}, dirs: map[string]*dir{}}
				cur.dirs[part] = next
			}
			cur = next
		}

		entry.Name = parts[len(parts)-1]
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

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
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

func fileDiffFromChange(change *object.Change) (domain.FileDiff, bool) {
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

	if patch, err := change.Patch(); err == nil {
		fd.Patch = patch.String()
	}

	return fd, true
}
