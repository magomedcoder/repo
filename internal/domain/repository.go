package domain

import (
	"context"
	"io"
	"time"
)

const OwnerKindUser = "user"

type Repository struct {
	ID                   uint
	Name                 string
	OwnerKind            string
	OwnerID              uint
	FolderID             *uint
	Description          string
	IsPrivate            bool
	ShowOnHome           bool
	ProtectDefaultBranch bool
	IssuesEnabled        bool
	DefaultBranch        string
	Path                 string
	RepoPath             string
	LastActivityAt       *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RepoOrder string

const (
	RepoOrderUpdated RepoOrder = ""
	RepoOrderName    RepoOrder = "name"
)

type RepositoryStore interface {
	Create(repo *Repository) error

	Update(repo *Repository) error

	Delete(id uint) error

	FindByRepoPath(repoPath string) (*Repository, error)

	ListByOwnerID(ownerKind string, ownerID uint, limit, offset int, order RepoOrder) ([]Repository, error)

	CountByOwnerID(ownerKind string, ownerID uint) (int64, error)

	ListPublicIDsByOwner(ownerKind string, ownerID uint) ([]uint, error)

	ListByOwnerAndFolderID(ownerKind string, ownerID uint, folderID *uint) ([]Repository, error)

	ListPublic(limit, offset int, order RepoOrder) ([]Repository, error)

	ListHome(limit, offset int, order RepoOrder) ([]Repository, error)

	SearchByName(query string, ownerKind string, ownerID uint, publicOnly bool, limit, offset int, order RepoOrder) ([]Repository, error)

	ExistsByOwnerFolderName(ownerKind string, ownerID uint, folderID *uint, name string) (bool, error)

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
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"` // tree | blob
	Mode       string    `json:"mode"`
	Size       int64     `json:"size"`
	SHA        string    `json:"sha"`
	ModifiedAt time.Time `json:"modified_at,omitzero"`
}

const MaxBlobReadBytes = 2 << 20

type BlobContent struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	IsBinary bool   `json:"is_binary"`
	Content  string `json:"content,omitzero"`
	Encoding string `json:"encoding,omitzero"` // utf-8 | base64
}

type BlameLine struct {
	Line       int    `json:"line"`
	Content    string `json:"content"`
	CommitSHA  string `json:"commit_sha"`
	AuthorName string `json:"author_name"`
	AuthoredAt string `json:"authored_at"`
}

type BlameResult struct {
	Path  string      `json:"path"`
	Ref   string      `json:"ref"`
	Lines []BlameLine `json:"lines"`
}

type CommitDiff struct {
	SHA     string     `json:"sha"`
	Message string     `json:"message"`
	Files   []FileDiff `json:"files"`
}

type FileDiff struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	OldPath string `json:"old_path,omitzero"`
	Patch   string `json:"patch"`
}

type RepoStats struct {
	CommitCount int              `json:"commit_count"`
	SizeBytes   int64            `json:"size_bytes"`
	Languages   map[string]int64 `json:"languages"`
}

type ReceivePackOptions struct {
	ProtectDefaultBranch bool
	DefaultBranch        string
}

type GitRepository interface {
	GetReposRoot() string

	InitBare(path string, opts BareInitOptions) error

	SetReceiveDenyNonFastForwards(path string, deny bool) error

	Remove(path string) error

	Move(oldPath, newPath string) error

	AdvertiseRefs(ctx context.Context, repoPath, service string, w io.Writer) error

	ServePack(ctx context.Context, repoPath, service string, stdin io.Reader, stdout io.Writer, receive *ReceivePackOptions) error

	LockRepo(repoPath string) func()

	EnsureDefaultBranch(repoPath, branch string) error

	ListBranches(repoPath string) ([]RefInfo, error)

	ListTags(repoPath string) ([]RefInfo, error)

	ListCommits(repoPath, ref string, offset, limit int) ([]CommitInfo, error)

	GetCommit(repoPath, sha string) (*CommitInfo, error)

	ListTree(repoPath, ref, path string) ([]TreeEntry, error)

	GetBlob(repoPath, ref, path string) (*BlobContent, error)

	GetBlame(repoPath, ref, path string) (*BlameResult, error)

	GetCommitDiff(repoPath, sha string) (*CommitDiff, error)

	Compare(repoPath, base, head string) (*CompareResult, error)

	CanFastForward(repoPath, base, head string) (bool, error)

	Merge(repoPath, base, head, strategy, authorName, authorEmail, message string) (string, error)

	GetStats(repoPath, ref string) (*RepoStats, error)

	FindReadme(repoPath, ref string) (*BlobContent, error)
}
