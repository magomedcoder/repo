package git

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/fsutil"
)

func (r *Repository) open(repoPath string) (*gogit.Repository, error) {
	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return nil, err
	}

	if r.opens != nil {
		return r.opens.getOrOpen(safe, func() (*gogit.Repository, error) {
			repo, err := gogit.PlainOpen(safe)
			if err != nil {
				return nil, fmt.Errorf("open repository: %w", err)
			}
			return repo, nil
		})
	}

	repo, err := gogit.PlainOpen(safe)
	if err != nil {
		return nil, fmt.Errorf("open repository: %w", err)
	}
	return repo, nil
}

func (r *Repository) ListBranches(repoPath string) ([]domain.RefInfo, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	iter, err := repo.Branches()
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var out []domain.RefInfo
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		out = append(out, domain.RefInfo{
			Name:      strings.TrimPrefix(ref.Name().String(), "refs/heads/"),
			CommitSHA: ref.Hash().String(),
		})

		return nil
	})

	return out, err
}

func (r *Repository) ListTags(repoPath string) ([]domain.RefInfo, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	var out []domain.RefInfo
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		sha := ref.Hash().String()
		if tag, err := repo.TagObject(ref.Hash()); err == nil {
			sha = tag.Target.String()
		}

		out = append(out, domain.RefInfo{
			Name:      strings.TrimPrefix(ref.Name().String(), "refs/tags/"),
			CommitSHA: sha,
		})
		return nil
	})

	return out, err
}

func (r *Repository) ListCommits(repoPath, ref string, offset, limit int) ([]domain.CommitInfo, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	hash, err := r.resolveHash(repo, ref)
	if err != nil {
		return nil, err
	}

	iter, err := repo.Log(&gogit.LogOptions{From: hash})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	if limit <= 0 {
		limit = 30
	}

	if offset < 0 {
		offset = 0
	}

	if offset > maxCommitOffset {
		offset = maxCommitOffset
	}

	var out []domain.CommitInfo
	i := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if i < offset {
			i++
			return nil
		}

		if len(out) >= limit {
			return stoplist
		}

		out = append(out, commitToInfo(c))
		i++
		return nil
	})
	if err == stoplist {
		err = nil
	}

	return out, err
}

func (r *Repository) GetCommit(repoPath, sha string) (*domain.CommitInfo, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	c, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCommitNotFound, err)
	}

	info := commitToInfo(c)
	return &info, nil
}

func (r *Repository) ListTree(repoPath, ref, dirPath string) ([]domain.TreeEntry, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := r.commitFromRef(repo, ref)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	dirPath = strings.Trim(dirPath, "/")
	if dirPath != "" {
		entry, err := tree.FindEntry(dirPath)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrPathNotFound, err)
		}

		if entry.Mode != filemode.Dir {
			return nil, ErrNotDirectory
		}

		tree, err = repo.TreeObject(entry.Hash)
		if err != nil {
			return nil, err
		}
	}

	out := make([]domain.TreeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		entryPath := e.Name
		if dirPath != "" {
			entryPath = path.Join(dirPath, e.Name)
		}

		item := domain.TreeEntry{
			Name: e.Name,
			Path: entryPath,
			SHA:  e.Hash.String(),
			Mode: e.Mode.String(),
		}

		switch e.Mode {
		case filemode.Dir:
			item.Type = "tree"
		default:
			item.Type = "blob"
			if blob, err := repo.BlobObject(e.Hash); err == nil {
				item.Size = blob.Size
			}
		}
		out = append(out, item)
	}

	_ = fillTreeModifiedAt(commit, out)
	return out, nil
}

const (
	maxTreeHistoryWalk = 100
	maxStatsCommits    = 10000
	maxStatsLangFiles  = 20000
	maxBlameLines      = 20000
	maxDiffFiles       = 500
	maxDiffPatchBytes  = 1 << 20
	maxCommitOffset    = 10000
)

func fillTreeModifiedAt(start *object.Commit, entries []domain.TreeEntry) error {
	if len(entries) == 0 || start == nil {
		return nil
	}

	tipWhen := start.Author.When
	pending := make(map[string]int, len(entries))
	for i := range entries {
		entries[i].ModifiedAt = tipWhen
		pending[entries[i].Path] = i
	}

	current := start
	for walked := 0; len(pending) > 0 && walked < maxTreeHistoryWalk; walked++ {
		currTree, err := current.Tree()
		if err != nil {
			break
		}

		when := current.Author.When
		if current.NumParents() == 0 {
			for _, idx := range pending {
				entries[idx].ModifiedAt = when
			}
			return nil
		}

		parent, err := current.Parent(0)
		if err != nil {
			for _, idx := range pending {
				entries[idx].ModifiedAt = when
			}
			return nil
		}

		parentTree, err := parent.Tree()
		if err != nil {
			for _, idx := range pending {
				entries[idx].ModifiedAt = when
			}
			return nil
		}

		for p, idx := range pending {
			if pathChangedSinceParent(currTree, parentTree, p) {
				entries[idx].ModifiedAt = when
				delete(pending, p)
			}
		}

		current = parent
	}

	if len(pending) > 0 && current != nil {
		when := current.Author.When
		for _, idx := range pending {
			entries[idx].ModifiedAt = when
		}
	}

	return nil
}

func pathChangedSinceParent(curr, parent *object.Tree, entryPath string) bool {
	currHash, currOK := treeEntryHash(curr, entryPath)
	if !currOK {
		return false
	}

	parentHash, parentOK := treeEntryHash(parent, entryPath)
	if !parentOK {
		return true
	}

	return currHash != parentHash
}

func treeEntryHash(tree *object.Tree, entryPath string) (plumbing.Hash, bool) {
	if tree == nil {
		return plumbing.ZeroHash, false
	}

	entry, err := tree.FindEntry(entryPath)
	if err != nil {
		return plumbing.ZeroHash, false
	}

	return entry.Hash, true
}

func (r *Repository) GetBlob(repoPath, ref, filePath string) (*domain.BlobContent, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := r.commitFromRef(repo, ref)
	if err != nil {
		return nil, err
	}

	filePath = strings.Trim(filePath, "/")
	if filePath == "" {
		return nil, fmt.Errorf("path required")
	}

	file, err := commit.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFileNotFound, err)
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, domain.MaxBlobReadBytes))
	if err != nil {
		return nil, err
	}

	out := &domain.BlobContent{
		Path: filePath,
		Size: file.Size,
	}
	if isBinary(data) {
		out.IsBinary = true
		out.Encoding = "base64"
		out.Content = base64.StdEncoding.EncodeToString(data)
	} else {
		out.Encoding = "utf-8"
		out.Content = string(data)
	}

	return out, nil
}

func (r *Repository) GetBlame(ctx context.Context, repoPath, ref, filePath string) (*domain.BlameResult, error) {
	return r.withTimeout(ctx, func(ctx context.Context) (*domain.BlameResult, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return r.getBlame(ctx, repoPath, ref, filePath)
	})
}

func (r *Repository) getBlame(ctx context.Context, repoPath, ref, filePath string) (*domain.BlameResult, error) {
	filePath = strings.Trim(filePath, "/")
	if filePath == "" {
		return nil, fmt.Errorf("path required")
	}

	if strings.TrimSpace(ref) == "" {
		ref = "HEAD"
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := r.commitFromRef(repo, ref)
	if err != nil {
		return nil, err
	}

	file, err := commit.File(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFileNotFound, err)
	}

	if file.Size > maxBlameFileBytes {
		return nil, ErrBlameTooLarge
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	type blameRes struct {
		b   *gogit.BlameResult
		err error
	}

	done := make(chan blameRes, 1)
	go func() {
		release, err := acquireSem(ctx, blameSem)
		if err != nil {
			done <- blameRes{nil, err}
			return
		}
		defer release()

		b, err := gogit.Blame(commit, filePath)
		done <- blameRes{b, err}
	}()

	var blamed *gogit.BlameResult
	select {
	case res := <-done:
		if res.err != nil {
			if errors.Is(res.err, domain.ErrBusy) || errors.Is(res.err, context.Canceled) || errors.Is(res.err, context.DeadlineExceeded) {
				return nil, res.err
			}
			return nil, fmt.Errorf("blame: %w", res.err)
		}
		blamed = res.b
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	result := &domain.BlameResult{Path: filePath, Ref: ref, Lines: []domain.BlameLine{}}
	for i, line := range blamed.Lines {
		if i >= maxBlameLines {
			break
		}
		if line == nil {
			continue
		}

		result.Lines = append(result.Lines, domain.BlameLine{
			Line:       i + 1,
			Content:    line.Text,
			CommitSHA:  line.Hash.String(),
			AuthorName: line.AuthorName,
			AuthoredAt: line.Date.UTC().Format(time.RFC3339),
		})
	}

	return result, nil
}

func (r *Repository) GetCommitDiff(ctx context.Context, repoPath, sha string) (*domain.CommitDiff, error) {
	return r.withTimeout(ctx, func(ctx context.Context) (*domain.CommitDiff, error) {
		release, err := acquireSem(ctx, diffSem)
		if err != nil {
			return nil, err
		}
		defer release()

		return r.getCommitDiff(ctx, repoPath, sha)
	})
}

func (r *Repository) getCommitDiff(ctx context.Context, repoPath, sha string) (*domain.CommitDiff, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCommitNotFound, err)
	}

	diff := &domain.CommitDiff{
		SHA:     commit.Hash.String(),
		Message: strings.TrimSpace(commit.Message),
		Files:   []domain.FileDiff{},
	}

	var parentTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err == nil {
			parentTree, _ = parent.Tree()
		}
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var changes object.Changes
	if parentTree == nil {
		changes, err = object.DiffTree(nil, tree)
	} else {
		changes, err = parentTree.Diff(tree)
	}

	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if len(diff.Files) >= maxDiffFiles {
			break
		}

		if fd, ok := fileDiffFromChange(ctx, change); ok {
			diff.Files = append(diff.Files, fd)
		}
	}

	return diff, nil
}

func (r *Repository) GetStats(repoPath, ref string) (*domain.RepoStats, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return nil, err
	}

	hash, herr := r.resolveHash(repo, ref)
	cacheKey := safe + "@" + hash.String()
	if herr == nil && r.stats != nil {
		if cached, ok := r.stats.get(cacheKey); ok {
			return cached, nil
		}
	}

	stats := &domain.RepoStats{
		Languages: map[string]int64{},
	}
	if herr != nil {
		return stats, nil
	}

	iter, err := repo.Log(&gogit.LogOptions{From: hash})
	if err != nil {
		return stats, nil
	}
	defer iter.Close()

	_ = iter.ForEach(func(c *object.Commit) error {
		stats.CommitCount++
		if stats.CommitCount >= maxStatsCommits {
			return stoplist
		}

		return nil
	})

	commit, err := repo.CommitObject(hash)
	if err != nil {
		return stats, nil
	}

	tree, err := commit.Tree()
	if err != nil {
		return stats, nil
	}

	langFiles := 0
	_ = tree.Files().ForEach(func(f *object.File) error {
		langFiles++
		if langFiles > maxStatsLangFiles {
			return stoplist
		}

		lang := languageFromPath(f.Name)
		if lang == "" {
			return nil
		}

		stats.Languages[lang] += f.Size
		return nil
	})

	if r.stats != nil {
		r.stats.put(cacheKey, stats)
	}

	return stats, nil
}

func (r *Repository) FindReadme(repoPath, ref string) (*domain.BlobContent, error) {
	candidates := []string{"README.md", "Readme.md", "readme.md", "README", "README.txt", "readme.txt"}
	var sawMissing bool
	for _, name := range candidates {
		blob, err := r.GetBlob(repoPath, ref, name)
		if err == nil {
			return blob, nil
		}
		switch {
		case errors.Is(err, ErrEmptyRepository), errors.Is(err, ErrRefNotFound):
			return nil, err
		case errors.Is(err, ErrFileNotFound), errors.Is(err, ErrPathNotFound), errors.Is(err, ErrNotDirectory):
			sawMissing = true
			continue
		default:
			return nil, err
		}
	}
	if sawMissing {
		return nil, nil
	}
	return nil, ErrReadmeNotFound
}

func (r *Repository) resolveHash(repo *gogit.Repository, ref string) (plumbing.Hash, error) {
	ref = strings.TrimSpace(ref)
	if err := validateRefName(ref); err != nil {
		return plumbing.ZeroHash, err
	}

	if ref == "" {
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, ErrEmptyRepository
		}

		return head.Hash(), nil
	}

	if h := plumbing.NewHash(ref); !h.IsZero() && len(ref) >= 7 {
		if _, err := repo.CommitObject(h); err == nil {
			return h, nil
		}
	}

	for _, full := range []string{ref, "refs/heads/" + ref, "refs/tags/" + ref} {
		if resolved, err := repo.ResolveRevision(plumbing.Revision(full)); err == nil {
			return *resolved, nil
		}
	}

	return plumbing.ZeroHash, fmt.Errorf("%w: %s", ErrRefNotFound, ref)
}

func (r *Repository) commitFromRef(repo *gogit.Repository, ref string) (*object.Commit, error) {
	hash, err := r.resolveHash(repo, ref)
	if err != nil {
		return nil, err
	}

	return repo.CommitObject(hash)
}

func commitToInfo(c *object.Commit) domain.CommitInfo {
	parents := make([]string, 0, c.NumParents())
	for i := range c.NumParents() {
		parents = append(parents, c.ParentHashes[i].String())
	}

	return domain.CommitInfo{
		SHA:         c.Hash.String(),
		Message:     strings.TrimSpace(c.Message),
		AuthorName:  c.Author.Name,
		AuthorEmail: c.Author.Email,
		AuthoredAt:  c.Author.When,
		Parents:     parents,
	}
}

var stoplist = fmt.Errorf("stop")

func isBinary(data []byte) bool {
	if bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	return !utf8.Valid(data)
}

func languageFromPath(p string) string {
	ext := strings.ToLower(filepath.Ext(p))
	switch ext {
	case ".go":
		return "Go"
	case ".js", ".mjs", ".cjs":
		return "JavaScript"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt":
		return "Kotlin"
	case ".c":
		return "C"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "C++"
	case ".cs":
		return "C#"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".swift":
		return "Swift"
	case ".vue":
		return "Vue"
	case ".css", ".scss", ".sass":
		return "CSS"
	case ".html", ".htm":
		return "HTML"
	case ".md", ".markdown":
		return "Markdown"
	case ".json":
		return "JSON"
	case ".yml", ".yaml":
		return "YAML"
	case ".sh", ".bash":
		return "Shell"
	case ".sql":
		return "SQL"
	default:
		return ""
	}
}
