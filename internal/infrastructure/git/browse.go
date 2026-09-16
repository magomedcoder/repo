package git

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/filemode"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/magomedcoder/repo/internal/domain"
)

func (r *Repository) open(repoPath string) (*gogit.Repository, error) {
	repo, err := gogit.PlainOpen(repoPath)
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
		return nil, fmt.Errorf("commit not found: %w", err)
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
			return nil, fmt.Errorf("path not found: %w", err)
		}

		if entry.Mode != filemode.Dir {
			return nil, fmt.Errorf("path is not a directory")
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

	return out, nil
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
		return nil, fmt.Errorf("file not found: %w", err)
	}

	reader, err := file.Reader()
	if err != nil {
		return nil, err
	}

	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, 2<<20)) // 2MB cap for API
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

func (r *Repository) GetCommitDiff(repoPath, sha string) (*domain.CommitDiff, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(plumbing.NewHash(sha))
	if err != nil {
		return nil, fmt.Errorf("commit not found: %w", err)
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
		action, err := change.Action()
		if err != nil {
			continue
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
		diff.Files = append(diff.Files, fd)
	}

	return diff, nil
}

func (r *Repository) GetStats(repoPath, ref string) (*domain.RepoStats, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	stats := &domain.RepoStats{
		Languages: map[string]int64{},
	}

	// size on disk
	_ = filepath.Walk(repoPath, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		stats.SizeBytes += info.Size()
		return nil
	})

	hash, err := r.resolveHash(repo, ref)
	if err != nil {
		// empty repo
		return stats, nil
	}

	iter, err := repo.Log(&gogit.LogOptions{From: hash})
	if err != nil {
		return stats, nil
	}
	defer iter.Close()

	_ = iter.ForEach(func(c *object.Commit) error {
		stats.CommitCount++
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

	_ = tree.Files().ForEach(func(f *object.File) error {
		lang := languageFromPath(f.Name)
		if lang == "" {
			return nil
		}

		stats.Languages[lang] += f.Size
		return nil
	})

	return stats, nil
}

func (r *Repository) FindReadme(repoPath, ref string) (*domain.BlobContent, error) {
	candidates := []string{"README.md", "Readme.md", "readme.md", "README", "README.txt", "readme.txt"}
	for _, name := range candidates {
		blob, err := r.GetBlob(repoPath, ref, name)
		if err == nil {
			return blob, nil
		}
	}

	return nil, fmt.Errorf("readme not found")
}

func (r *Repository) resolveHash(repo *gogit.Repository, ref string) (plumbing.Hash, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		head, err := repo.Head()
		if err != nil {
			return plumbing.ZeroHash, fmt.Errorf("empty repository")
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

	return plumbing.ZeroHash, fmt.Errorf("ref not found: %s", ref)
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
	for i := 0; i < c.NumParents(); i++ {
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
