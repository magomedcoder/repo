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

	ListForMember(userID uint) ([]Folder, error)

	ExistsBySlug(slug string) (bool, error)
}
