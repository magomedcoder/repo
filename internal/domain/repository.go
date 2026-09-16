package domain

import "time"

type Repository struct {
	ID            uint
	Name          string
	OwnerID       uint
	FolderID      *uint // nil = user root
	Description   string
	IsPrivate     bool
	DefaultBranch string
	Path          string // internal filesystem path
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
}

type GitRepository interface {
	InitBare(path string) error

	Remove(path string) error

	Move(oldPath, newPath string) error
}
