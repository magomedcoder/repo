package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type repositoryModel struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         `gorm:"uniqueIndex;not null"`
	OwnerID     uint           `gorm:"index;not null"`
	FolderID    uint           `gorm:"index;default:0"` // 0 = user root
	Description string
	Path        string
}

func (repositoryModel) TableName() string {
	return "repositories"
}

func (m repositoryModel) toDomain() domain.Repository {
	repo := domain.Repository{
		ID:          m.ID,
		Name:        m.Name,
		OwnerID:     m.OwnerID,
		Description: m.Description,
		Path:        m.Path,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.FolderID != 0 {
		fid := m.FolderID
		repo.FolderID = &fid
	}

	return repo
}

func fromDomain(repo *domain.Repository) *repositoryModel {
	m := &repositoryModel{
		ID:          repo.ID,
		Name:        repo.Name,
		OwnerID:     repo.OwnerID,
		Description: repo.Description,
		Path:        repo.Path,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}
	if repo.FolderID != nil {
		m.FolderID = *repo.FolderID
	}

	return m
}
