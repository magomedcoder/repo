package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type Folder struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	OwnerKind string `gorm:"size:8;not null;default:user"`
	OwnerID   uint   `gorm:"not null"`
	Name      string `gorm:"size:64;not null"`
	Slug      string `gorm:"size:64;not null"`
	Path      string `gorm:"size:512;not null"`
}

func (Folder) TableName() string {
	return "folders"
}

func (m Folder) ToDomain() domain.Folder {
	kind := m.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return domain.Folder{
		ID:        m.ID,
		OwnerKind: kind,
		OwnerID:   m.OwnerID,
		Name:      m.Name,
		Slug:      m.Slug,
		Path:      m.Path,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FolderFromDomain(f *domain.Folder) *Folder {
	kind := f.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	return &Folder{
		ID:        f.ID,
		OwnerKind: kind,
		OwnerID:   f.OwnerID,
		Name:      f.Name,
		Slug:      f.Slug,
		Path:      f.Path,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}
