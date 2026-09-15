package domain

import "time"

type Repository struct {
	ID          uint
	Name        string
	OwnerID     uint
	FolderID    *uint // nil = user root
	Description string
	Path        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RepositoryStore interface {
	Create(repo *Repository) error

	ListByOwnerID(ownerID uint) ([]Repository, error)

	ListByOwnerAndFolderID(ownerID uint, folderID *uint) ([]Repository, error)

	ExistsByName(name string) (bool, error)

	CountByFolderID(folderID uint) (int64, error)
}

type GitRepository interface {
	InitBare(path string) error
}
