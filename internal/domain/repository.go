package domain

import (
	"io"
	"time"
)

type Repository struct {
	ID             uint
	Name           string
	OwnerID        uint
	FolderID       *uint // nil = user root
	Description    string
	IsPrivate      bool
	DefaultBranch  string
	Path           string // internal filesystem path
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

type GitRepository interface {
	InitBare(path string, opts BareInitOptions) error

	Remove(path string) error

	Move(oldPath, newPath string) error

	AdvertiseRefs(repoPath, service string, w io.Writer) error

	ServePack(repoPath, service string, stdin io.Reader, stdout io.Writer) error
}
