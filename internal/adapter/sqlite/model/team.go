package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type Team struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt
	OwnerUserID uint   `gorm:"not null"`
	Name        string `gorm:"size:64;not null"`
	Slug        string `gorm:"size:64;not null"`
}

func (Team) TableName() string {
	return "teams"
}

func (m Team) ToDomain() domain.Team {
	return domain.Team{
		ID:          m.ID,
		OwnerUserID: m.OwnerUserID,
		Name:        m.Name,
		Slug:        m.Slug,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
