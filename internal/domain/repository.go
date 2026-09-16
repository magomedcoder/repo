package domain

import (
	"io"
	"time"
)

type Repository struct {
	ID             uint
	Name           string
	OwnerID        uint
	FolderID       *uint
	Description    string
	IsPrivate      bool
	DefaultBranch  string
	Path           string
	LastActivityAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RepositoryStore interface {
	Create(repo *Repository) error

	Update(repo *Repository) error

	Delete(id uint) error

	FindByOwnerFolderName(ownerID uint, folderID *uint, name string) (*Repository, error)

	ListByOwnerID(ownerID uint) ([]Repository, error)

	ListByOwnerAndFolderID(ownerID uint, folderID *uint) ([]Repository, error)

	ListPublic(limit int) ([]Repository, error)

	ExistsByOwnerFolderName(ownerID uint, folderID *uint, name string) (bool, error)

	CountByFolderID(folderID uint) (int64, error)

	TouchLastActivity(id uint, at time.Time) error
}

type BareInitOptions struct {
	DefaultBranch        string
	DenyForcePushDefault bool
}

type RefInfo struct {
	Name      string `json:"name"`
	CommitSHA string `json:"commit_sha"`
}

type CommitInfo struct {
	SHA         string    `json:"sha"`
	Message     string    `json:"message"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	AuthoredAt  time.Time `json:"authored_at"`
	Parents     []string  `json:"parents"`
}

type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // tree | blob
	Mode string `json:"mode"`
	Size int64  `json:"size"`
	SHA  string `json:"sha"`
}

type BlobContent struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsBinary bool   `json:"is_binary"`
	Content  string `json:"content,omitempty"`
	Encoding string `json:"encoding,omitempty"` // utf-8 | base64
}

type CommitDiff struct {
	SHA     string     `json:"sha"`
	Message string     `json:"message"`
	Files   []FileDiff `json:"files"`
}

type FileDiff struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	OldPath string `json:"old_path,omitempty"`
	Patch   string `json:"patch"`
}

type RepoStats struct {
	CommitCount int              `json:"commit_count"`
	SizeBytes   int64            `json:"size_bytes"`
	Languages   map[string]int64 `json:"languages"`
}

type GitRepository interface {
	InitBare(path string, opts BareInitOptions) error

	Remove(path string) error

	Move(oldPath, newPath string) error

	AdvertiseRefs(repoPath, service string, w io.Writer) error

	ServePack(repoPath, service string, stdin io.Reader, stdout io.Writer) error

	ListBranches(repoPath string) ([]RefInfo, error)

	ListTags(repoPath string) ([]RefInfo, error)

	ListCommits(repoPath, ref string, offset, limit int) ([]CommitInfo, error)

	GetCommit(repoPath, sha string) (*CommitInfo, error)

	ListTree(repoPath, ref, path string) ([]TreeEntry, error)

	GetBlob(repoPath, ref, path string) (*BlobContent, error)

	GetCommitDiff(repoPath, sha string) (*CommitDiff, error)

	GetStats(repoPath, ref string) (*RepoStats, error)

	FindReadme(repoPath, ref string) (*BlobContent, error)
}
