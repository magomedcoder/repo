package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type TeamRepoGrant struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	TeamID    uint   `gorm:"not null"`
	RepoID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
}

func (TeamRepoGrant) TableName() string {
	return "team_repo_grants"
}

func (m TeamRepoGrant) ToDomain() domain.TeamRepoGrant {
	return domain.TeamRepoGrant{
		ID:        m.ID,
		TeamID:    m.TeamID,
		RepoID:    m.RepoID,
		Role:      domain.RepoCollabRole(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
