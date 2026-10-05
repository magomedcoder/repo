package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type repositoryModel struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	Name           string         `gorm:"size:100;not null;uniqueIndex:idx_repo_scope"`
	OwnerKind      string         `gorm:"size:8;not null;default:user;uniqueIndex:idx_repo_scope"`
	OwnerID        uint           `gorm:"not null;uniqueIndex:idx_repo_scope"`
	FolderID       uint           `gorm:"not null;default:0;uniqueIndex:idx_repo_scope"`
	Description    string
	IsPrivate      bool   `gorm:"not null;default:false"`
	DefaultBranch  string `gorm:"size:255;not null;default:main"`
	Path           string `gorm:"not null"`
	LastActivityAt *time.Time
}

func (repositoryModel) TableName() string {
	return "repositories"
}

func (m repositoryModel) toDomain() domain.Repository {
	kind := m.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	repo := domain.Repository{
		ID:             m.ID,
		Name:           m.Name,
		OwnerKind:      kind,
		OwnerID:        m.OwnerID,
		Description:    m.Description,
		IsPrivate:      m.IsPrivate,
		DefaultBranch:  m.DefaultBranch,
		Path:           m.Path,
		LastActivityAt: m.LastActivityAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}

	if m.FolderID != 0 {
		fid := m.FolderID
		repo.FolderID = &fid
	}
	
	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	return repo
}

func fromDomain(repo *domain.Repository) *repositoryModel {
	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}

	m := &repositoryModel{
		ID:             repo.ID,
		Name:           repo.Name,
		OwnerKind:      kind,
		OwnerID:        repo.OwnerID,
		Description:    repo.Description,
		IsPrivate:      repo.IsPrivate,
		DefaultBranch:  repo.DefaultBranch,
		Path:           repo.Path,
		LastActivityAt: repo.LastActivityAt,
		CreatedAt:      repo.CreatedAt,
		UpdatedAt:      repo.UpdatedAt,
	}
	if repo.FolderID != nil {
		m.FolderID = *repo.FolderID
	}
	
	if m.DefaultBranch == "" {
		m.DefaultBranch = "main"
	}

	return m
}
