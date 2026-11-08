package domain

import (
	"context"
	"io"
	"time"
)

const (
	MirrorNone = ""
	MirrorPull = "pull"
	MirrorPush = "push"
)

type Repository struct {
	ID                   uint
	Name                 string
	OwnerID              uint
	Description          string
	IsPrivate            bool
	ShowOnHome           bool
	ProtectDefaultBranch bool
	IssuesEnabled        bool
	DefaultBranch        string
	Path                 string
	RepoPath             string
	MirrorDirection      string
	MirrorURL            string
	MirrorUsername       string
	MirrorPassword       string
	MirrorLastSyncAt     *time.Time
	MirrorLastError      string
	MirrorEnabled        bool
	MirrorInterval       string
	MirrorSyncWindow     string
	MirrorForce          bool
	MirrorPrune          bool
	MirrorRefspecs       string
	MirrorSSHPrivateKey  string
	MirrorSSHPassphrase  string
	MirrorInsecureTLS    bool
	MirrorPinnedHosts    string
	MirrorMaxBytes       int64
	MirrorConflictPolicy string
	MirrorStatus         string
	MirrorFailCount      int
	MirrorNextSyncAt     *time.Time
	MirrorLastProgress   string
	MirrorRemoteHEAD     string
	MirrorOwnerOnly      bool
	MirrorGroupID        uint
	MirrorVerifySigned   bool
	LastActivityAt       *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RepoOrder string

const (
	RepoOrderUpdated RepoOrder = ""
	RepoOrderName    RepoOrder = "name"
)

type AdminRepoFilter struct {
	Query   string
	Private *bool
	Mirror  bool
	Order   RepoOrder
	Limit   int
	Offset  int
}

type RepositoryStore interface {
	Create(repo *Repository) error

	Update(repo *Repository) error

	Delete(id uint) error

	FindByRepoPath(ctx context.Context, repoPath string) (*Repository, error)

	ListByOwnerID(ctx context.Context, ownerID uint, limit, offset int, order RepoOrder) ([]Repository, error)

	CountByOwnerID(ownerID uint) (int64, error)

	ListIDsByOwner(ctx context.Context, ownerID uint) ([]uint, error)

	ListPublic(ctx context.Context, limit, offset int, order RepoOrder) ([]Repository, error)

	ListHome(ctx context.Context, limit, offset int, order RepoOrder) ([]Repository, error)

	ListSharedForUser(ctx context.Context, userID uint, limit, offset int, order RepoOrder) ([]Repository, error)

	ListMirrors(ctx context.Context, filter MirrorListFilter) ([]Repository, error)

	ListMirrorSchedules(ctx context.Context, f MirrorScheduleFilter) ([]MirrorSchedule, error)

	ResetStaleMirrorSyncing(ctx context.Context) (int64, error)

	CountMirrorsInError(ctx context.Context, since time.Time) (int64, error)

	CountAll(ctx context.Context) (int64, error)

	ListAdmin(ctx context.Context, filter AdminRepoFilter) ([]Repository, int, error)

	UpdateMirrorState(ctx context.Context, repo *Repository) error

	FindByID(ctx context.Context, id uint) (*Repository, error)

	SearchByName(ctx context.Context, query string, ownerID uint, publicOnly bool, limit, offset int, order RepoOrder) ([]Repository, error)

	SearchAccessible(ctx context.Context, query string, viewerID uint, limit, offset int, order RepoOrder) ([]Repository, error)

	ExistsByName(name string) (bool, error)

	TouchLastActivity(id uint, at time.Time) error

	PurgeSoftDeleted(ctx context.Context, before time.Time) (paths []string, ids []uint, err error)
}

type RepoNameHoldStore interface {
	Hold(ctx context.Context, name string, expiresAt time.Time) error

	IsHeld(ctx context.Context, name string, now time.Time) (bool, error)

	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}

type BareInitOptions struct {
	DefaultBranch string
}

type MirrorAuth struct {
	Username string
	Password string
}

type MirrorOptions struct {
	Auth          *MirrorAuth
	SSHPrivateKey string
	SSHPassphrase string
	InsecureTLS   bool
	Force         *bool
	Prune         *bool
	RefSpecs      []string
	LFS           bool
	MaxBytes      int64
	Partial       bool
	PinnedHosts   map[string]string
	Progress      io.Writer
}

type MirrorRemoteInfo struct {
	DefaultBranch string    `json:"default_branch"`
	Refs          []RefInfo `json:"refs"`
	Head          string    `json:"head"`
}

type MirrorRefDiff struct {
	Name   string `json:"name"`
	Local  string `json:"local,omitzero"`
	Remote string `json:"remote,omitzero"`
	Reason string `json:"reason"`
}

type MirrorSchedule struct {
	ID         uint
	Direction  string
	URL        string
	Enabled    bool
	Interval   string
	SyncWindow string
	NextSyncAt *time.Time
	LastSyncAt *time.Time
}

type MirrorScheduleFilter struct {
	Offset    int
	Limit     int
	DueBefore *time.Time
}

type MirrorVerifyResult struct {
	OK            bool            `json:"ok"`
	DefaultBranch string          `json:"default_branch,omitzero"`
	Head          string          `json:"head,omitzero"`
	Checked       int             `json:"checked"`
	Diffs         []MirrorRefDiff `json:"diffs,omitzero"`
	Truncated     bool            `json:"truncated,omitzero"`
	TotalDiffs    int             `json:"total_diffs,omitzero"`
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
	Languages   map[string]int64 `json:"languages"`
}

type CodeSearchHit struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type ReceivePackOptions struct {
	ProtectDefaultBranch bool
	DefaultBranch        string
}

type GitRepository interface {
	GetReposRoot() string

	InitBare(path string, opts BareInitOptions) error

	MirrorClone(ctx context.Context, path, url string, opts *MirrorOptions) (defaultBranch string, err error)

	MirrorFetch(ctx context.Context, path, url string, opts *MirrorOptions) error

	MirrorPush(ctx context.Context, path, url string, opts *MirrorOptions) error

	MirrorCheck(ctx context.Context, url string, opts *MirrorOptions) (*MirrorRemoteInfo, error)

	MirrorDryRunFetch(ctx context.Context, path, url string, opts *MirrorOptions) (*MirrorRemoteInfo, []RefInfo, error)

	MirrorVerify(ctx context.Context, path, url, direction string, opts *MirrorOptions) (*MirrorVerifyResult, error)

	Remove(path string) error

	Move(oldPath, newPath string) error

	AdvertiseRefs(ctx context.Context, repoPath, service string, w io.Writer) error

	ServePack(ctx context.Context, repoPath, service string, stdin io.Reader, stdout io.Writer, receive *ReceivePackOptions) error

	LockRepo(repoPath string) func()

	EnsureDefaultBranch(repoPath, branch string) error

	DetectHEADBranch(repoPath string) (string, error)

	VerifyCommitSignature(ctx context.Context, repoPath, sha string) error

	ListBranches(repoPath string) ([]RefInfo, error)

	ListTags(repoPath string) ([]RefInfo, error)

	ListCommits(repoPath, ref string, offset, limit int) ([]CommitInfo, error)

	GetCommit(repoPath, sha string) (*CommitInfo, error)

	ListTree(repoPath, ref, path string) ([]TreeEntry, error)

	GetBlob(repoPath, ref, path string) (*BlobContent, error)

	GetBlame(ctx context.Context, repoPath, ref, path string) (*BlameResult, error)

	GetCommitDiff(ctx context.Context, repoPath, sha string) (*CommitDiff, error)

	Compare(ctx context.Context, repoPath, base, head string) (*CompareResult, error)

	Merge(ctx context.Context, repoPath, base, head, strategy, authorName, authorEmail, message string, protect bool) (string, error)

	GetStats(repoPath, ref string) (*RepoStats, error)

	FindReadme(repoPath, ref string) (*BlobContent, error)

	WalkTextFiles(ctx context.Context, repoPath string) (tipSHA string, files []IndexedTextFile, err error)
}

type IndexedTextFile struct {
	Path string
	Body string
}
