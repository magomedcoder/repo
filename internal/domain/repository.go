package domain

import "time"

type Repository struct {
	ID          uint
	Name        string
	OwnerID     uint
	Description string
	Path        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RepositoryStore interface {
	Create(repo *Repository) error

	ListByOwnerID(ownerID uint) ([]Repository, error)

	ExistsByName(name string) (bool, error)
}

type GitRepository interface {
	InitBare(path string) error
}
