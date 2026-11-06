package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type RepoCollaborator struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	RepoID    uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
}

func (RepoCollaborator) TableName() string {
	return "repo_collaborators"
}

func (m RepoCollaborator) ToDomain() domain.RepoCollaborator {
	return domain.RepoCollaborator{
		ID:        m.ID,
		RepoID:    m.RepoID,
		UserID:    m.UserID,
		Role:      domain.RepoCollabRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
