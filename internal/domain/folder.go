package domain

import "time"

type Folder struct {
	ID        uint
	OwnerID   uint
	ParentID  *uint // nil = user root
	Name      string
	Slug      string
	Path      string // materialized, e.g. "work/backend"
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FolderStore interface {
	Create(folder *Folder) error

	Update(folder *Folder) error

	Delete(id uint) error

	FindByID(id uint) (*Folder, error)

	FindByOwnerAndID(ownerID, id uint) (*Folder, error)

	FindByOwnerAndPath(ownerID uint, path string) (*Folder, error)

	ListByOwner(ownerID uint) ([]Folder, error)

	ListByOwnerAndParent(ownerID uint, parentID *uint) ([]Folder, error)

	ExistsByOwnerParentSlug(ownerID uint, parentID *uint, slug string) (bool, error)

	CountChildren(folderID uint) (int64, error)

	ListDescendants(ownerID uint, pathPrefix string) ([]Folder, error)
}
