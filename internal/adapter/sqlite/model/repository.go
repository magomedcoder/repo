package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type Repository struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt
	Name                 string `gorm:"size:100;not null"`
	OwnerID              uint   `gorm:"not null"`
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
	MirrorEnabled        bool   `gorm:"not null;default:true"`
	MirrorInterval       string `gorm:"size:32;not null;default:''"`
	MirrorSyncWindow     string `gorm:"size:32;not null;default:''"`
	MirrorForce          bool   `gorm:"not null;default:true"`
	MirrorPrune          bool   `gorm:"not null;default:true"`
	MirrorRefspecs       string `gorm:"size:2048;not null;default:''"`
	MirrorSSHPrivateKey  string `gorm:"type:text;not null;default:''"`
	MirrorSSHPassphrase  string `gorm:"size:2048;not null;default:''"`
	MirrorInsecureTLS    bool   `gorm:"not null;default:false"`
	MirrorPinnedHosts    string `gorm:"size:1024;not null;default:''"`
	MirrorLFS            bool   `gorm:"not null;default:false"`
	MirrorMaxBytes       int64  `gorm:"not null;default:0"`
	MirrorConflictPolicy string `gorm:"size:16;not null;default:''"`
	MirrorStatus         string `gorm:"size:16;not null;default:''"`
	MirrorFailCount      int    `gorm:"not null;default:0"`
	MirrorNextSyncAt     *time.Time
	MirrorLastProgress   string `gorm:"size:512;not null;default:''"`
	MirrorRemoteHEAD     string `gorm:"column:mirror_remote_head;size:64;not null;default:''"`
	MirrorOwnerOnly      bool   `gorm:"not null;default:false"`
	MirrorGroupID        uint   `gorm:"not null;default:0"`
	MirrorPartial        bool   `gorm:"not null;default:false"`
	MirrorVerifySigned   bool   `gorm:"not null;default:false"`
	LastActivityAt       *time.Time
}

func (Repository) TableName() string {
	return "repositories"
}

func (m Repository) ToDomain() domain.Repository {
	repo := domain.Repository{
		ID:                   m.ID,
		Name:                 m.Name,
		OwnerID:              m.OwnerID,
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
		MirrorEnabled:        m.MirrorEnabled,
		MirrorInterval:       m.MirrorInterval,
		MirrorSyncWindow:     m.MirrorSyncWindow,
		MirrorForce:          m.MirrorForce,
		MirrorPrune:          m.MirrorPrune,
		MirrorRefspecs:       m.MirrorRefspecs,
		MirrorSSHPrivateKey:  m.MirrorSSHPrivateKey,
		MirrorSSHPassphrase:  m.MirrorSSHPassphrase,
		MirrorInsecureTLS:    m.MirrorInsecureTLS,
		MirrorPinnedHosts:    m.MirrorPinnedHosts,
		MirrorLFS:            m.MirrorLFS,
		MirrorMaxBytes:       m.MirrorMaxBytes,
		MirrorConflictPolicy: m.MirrorConflictPolicy,
		MirrorStatus:         m.MirrorStatus,
		MirrorFailCount:      m.MirrorFailCount,
		MirrorNextSyncAt:     m.MirrorNextSyncAt,
		MirrorLastProgress:   m.MirrorLastProgress,
		MirrorRemoteHEAD:     m.MirrorRemoteHEAD,
		MirrorOwnerOnly:      m.MirrorOwnerOnly,
		MirrorGroupID:        m.MirrorGroupID,
		MirrorPartial:        m.MirrorPartial,
		MirrorVerifySigned:   m.MirrorVerifySigned,
		LastActivityAt:       m.LastActivityAt,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}

	if repo.DefaultBranch == "" {
		repo.DefaultBranch = "main"
	}

	return repo
}

func RepositoryFromDomain(repo *domain.Repository) *Repository {
	m := &Repository{
		ID:                   repo.ID,
		Name:                 repo.Name,
		OwnerID:              repo.OwnerID,
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
		MirrorEnabled:        repo.MirrorEnabled,
		MirrorInterval:       repo.MirrorInterval,
		MirrorSyncWindow:     repo.MirrorSyncWindow,
		MirrorForce:          repo.MirrorForce,
		MirrorPrune:          repo.MirrorPrune,
		MirrorRefspecs:       repo.MirrorRefspecs,
		MirrorSSHPrivateKey:  repo.MirrorSSHPrivateKey,
		MirrorSSHPassphrase:  repo.MirrorSSHPassphrase,
		MirrorInsecureTLS:    repo.MirrorInsecureTLS,
		MirrorPinnedHosts:    repo.MirrorPinnedHosts,
		MirrorLFS:            repo.MirrorLFS,
		MirrorMaxBytes:       repo.MirrorMaxBytes,
		MirrorConflictPolicy: repo.MirrorConflictPolicy,
		MirrorStatus:         repo.MirrorStatus,
		MirrorFailCount:      repo.MirrorFailCount,
		MirrorNextSyncAt:     repo.MirrorNextSyncAt,
		MirrorLastProgress:   repo.MirrorLastProgress,
		MirrorRemoteHEAD:     repo.MirrorRemoteHEAD,
		MirrorOwnerOnly:      repo.MirrorOwnerOnly,
		MirrorGroupID:        repo.MirrorGroupID,
		MirrorPartial:        repo.MirrorPartial,
		MirrorVerifySigned:   repo.MirrorVerifySigned,
		LastActivityAt:       repo.LastActivityAt,
		CreatedAt:            repo.CreatedAt,
		UpdatedAt:            repo.UpdatedAt,
	}

	if m.DefaultBranch == "" {
		m.DefaultBranch = "main"
	}

	return m
}
