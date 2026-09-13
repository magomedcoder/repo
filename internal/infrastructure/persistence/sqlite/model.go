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
	OwnerID     uint
	Description string
	Path        string
}

func (repositoryModel) TableName() string {
	return "repositories"
}

func (m repositoryModel) toDomain() domain.Repository {
	return domain.Repository{
		ID:          m.ID,
		Name:        m.Name,
		OwnerID:     m.OwnerID,
		Description: m.Description,
		Path:        m.Path,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func fromDomain(repo *domain.Repository) *repositoryModel {
	return &repositoryModel{
		ID:          repo.ID,
		Name:        repo.Name,
		OwnerID:     repo.OwnerID,
		Description: repo.Description,
		Path:        repo.Path,
		CreatedAt:   repo.CreatedAt,
		UpdatedAt:   repo.UpdatedAt,
	}
}
