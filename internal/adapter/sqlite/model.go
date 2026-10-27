package sqlite

import (
	"time"

	"git.magomedcoder.ru/repo/internal/domain"
	"gorm.io/gorm"
)

type repositoryModel struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt
	Name                 string `gorm:"size:100;not null"`
	OwnerKind            string `gorm:"size:8;not null;default:user"`
	OwnerID              uint   `gorm:"not null"`
	FolderID             *uint  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Folder               *folderModel `gorm:"foreignKey:FolderID;references:ID"`
	Description          string
	IsPrivate            bool   `gorm:"not null;default:false"`
	ProtectDefaultBranch bool   `gorm:"not null;default:true"`
	DefaultBranch        string `gorm:"size:255;not null;default:main"`
	Path                 string `gorm:"not null"`
	RepoPath             string `gorm:"size:512;not null;default:''"`
	LastActivityAt       *time.Time
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
		ID:                   m.ID,
		Name:                 m.Name,
		OwnerKind:            kind,
		OwnerID:              m.OwnerID,
		FolderID:             m.FolderID,
		Description:          m.Description,
		IsPrivate:            m.IsPrivate,
		ProtectDefaultBranch: m.ProtectDefaultBranch,
		DefaultBranch:        m.DefaultBranch,
		Path:                 m.Path,
		RepoPath:             m.RepoPath,
		LastActivityAt:       m.LastActivityAt,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
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
		ID:                   repo.ID,
		Name:                 repo.Name,
		OwnerKind:            kind,
		OwnerID:              repo.OwnerID,
		FolderID:             repo.FolderID,
		Description:          repo.Description,
		IsPrivate:            repo.IsPrivate,
		ProtectDefaultBranch: repo.ProtectDefaultBranch,
		DefaultBranch:        repo.DefaultBranch,
		Path:                 repo.Path,
		RepoPath:             repo.RepoPath,
		LastActivityAt:       repo.LastActivityAt,
		CreatedAt:            repo.CreatedAt,
		UpdatedAt:            repo.UpdatedAt,
	}

	if m.DefaultBranch == "" {
		m.DefaultBranch = "main"
	}

	return m
}
