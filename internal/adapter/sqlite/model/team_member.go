package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type TeamMember struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
	TeamID    uint `gorm:"not null"`
	UserID    uint `gorm:"not null"`
}

func (TeamMember) TableName() string {
	return "team_members"
}

func (m TeamMember) ToDomain() domain.TeamMember {
	return domain.TeamMember{
		ID:        m.ID,
		TeamID:    m.TeamID,
		UserID:    m.UserID,
		CreatedAt: m.CreatedAt,
	}
}
