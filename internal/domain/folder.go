package domain

import "time"

type Folder struct {
	ID        uint
	OwnerKind string // user | org
	OwnerID   uint
	ParentID  *uint // nil = namespace root
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

	FindByOwnerAndID(ownerKind string, ownerID, id uint) (*Folder, error)

	FindByOwnerAndPath(ownerKind string, ownerID uint, path string) (*Folder, error)

	ListByOwner(ownerKind string, ownerID uint) ([]Folder, error)

	ListByOwnerAndParent(ownerKind string, ownerID uint, parentID *uint) ([]Folder, error)

	ExistsByOwnerParentSlug(ownerKind string, ownerID uint, parentID *uint, slug string) (bool, error)
	CountChildren(folderID uint) (int64, error)

	ListDescendants(ownerKind string, ownerID uint, pathPrefix string) ([]Folder, error)
}
