package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type Team struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	FolderID  uint   `gorm:"not null"`
	Name      string `gorm:"size:64;not null"`
	Slug      string `gorm:"size:64;not null"`
	Caps      uint32 `gorm:"not null;default:0"`
}

func (Team) TableName() string {
	return "teams"
}

func (m Team) ToDomain() domain.Team {
	return domain.Team{
		ID:        m.ID,
		FolderID:  m.FolderID,
		Name:      m.Name,
		Slug:      m.Slug,
		Caps:      domain.FolderCap(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
