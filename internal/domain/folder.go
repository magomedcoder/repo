package domain

import "time"

type Folder struct {
	ID        uint
	OwnerKind string
	OwnerID   uint
	Name      string
	Slug      string
	Path      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FolderStore interface {
	Create(folder *Folder) error

	Update(folder *Folder) error

	Delete(id uint) error

	FindByID(id uint) (*Folder, error)

	FindByPath(path string) (*Folder, error)

	FindByOwnerAndID(ownerKind string, ownerID, id uint) (*Folder, error)

	FindByOwnerAndPath(ownerKind string, ownerID uint, path string) (*Folder, error)

	ListByOwner(ownerKind string, ownerID uint) ([]Folder, error)

	ListForMember(userID uint) ([]Folder, error)

	ExistsBySlug(slug string) (bool, error)

	ExistsByOwnerSlug(ownerKind string, ownerID uint, slug string) (bool, error)
}
