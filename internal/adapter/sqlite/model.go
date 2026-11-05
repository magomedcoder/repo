package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type repositoryModel struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt
	Name                 string       `gorm:"size:100;not null"`
	OwnerKind            string       `gorm:"size:8;not null;default:user"`
	OwnerID              uint         `gorm:"not null"`
	FolderID             *uint        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Folder               *folderModel `gorm:"foreignKey:FolderID;references:ID"`
	Description          string
	IsPrivate            bool   `gorm:"not null;default:false"`
	ShowOnHome           bool   `gorm:"not null;default:false"`
	ProtectDefaultBranch bool   `gorm:"not null;default:true"`
	IssuesEnabled        bool   `gorm:"not null;default:true"`
	DefaultBranch        string `gorm:"size:255;not null;default:main"`
	Path                 string `gorm:"not null"`
	RepoPath             string `gorm:"size:512;not null;default:''"`
	IssueSeq             int    `gorm:"not null;default:0"`
	PullSeq              int    `gorm:"not null;default:0"`
	MirrorDirection      string `gorm:"size:8;not null;default:''"`
	MirrorURL            string `gorm:"size:2048;not null;default:''"`
	MirrorUsername       string `gorm:"size:255;not null;default:''"`
	MirrorPassword       string `gorm:"size:2048;not null;default:''"`
	MirrorLastSyncAt     *time.Time
	MirrorLastError      string `gorm:"size:1024;not null;default:''"`
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
		ShowOnHome:           m.ShowOnHome && !m.IsPrivate,
		ProtectDefaultBranch: m.ProtectDefaultBranch,
		IssuesEnabled:        m.IssuesEnabled,
		DefaultBranch:        m.DefaultBranch,
		Path:                 m.Path,
		RepoPath:             m.RepoPath,
		MirrorDirection:      m.MirrorDirection,
		MirrorURL:            m.MirrorURL,
		MirrorUsername:       m.MirrorUsername,
		MirrorPassword:       m.MirrorPassword,
		MirrorLastSyncAt:     m.MirrorLastSyncAt,
		MirrorLastError:      m.MirrorLastError,
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
		ShowOnHome:           repo.ShowOnHome && !repo.IsPrivate,
		ProtectDefaultBranch: repo.ProtectDefaultBranch,
		IssuesEnabled:        repo.IssuesEnabled,
		DefaultBranch:        repo.DefaultBranch,
		Path:                 repo.Path,
		RepoPath:             repo.RepoPath,
		MirrorDirection:      repo.MirrorDirection,
		MirrorURL:            repo.MirrorURL,
		MirrorUsername:       repo.MirrorUsername,
		MirrorPassword:       repo.MirrorPassword,
		MirrorLastSyncAt:     repo.MirrorLastSyncAt,
		MirrorLastError:      repo.MirrorLastError,
		LastActivityAt:       repo.LastActivityAt,
		CreatedAt:            repo.CreatedAt,
		UpdatedAt:            repo.UpdatedAt,
	}

	if m.DefaultBranch == "" {
		m.DefaultBranch = "main"
	}

	return m
}
