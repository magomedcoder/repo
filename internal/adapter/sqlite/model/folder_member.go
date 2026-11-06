package model

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type FolderMember struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt
	FolderID  uint   `gorm:"not null"`
	UserID    uint   `gorm:"not null"`
	Role      string `gorm:"size:16;not null"`
	Caps      uint32 `gorm:"not null;default:0"`
}

func (FolderMember) TableName() string {
	return "folder_members"
}

func (m FolderMember) ToDomain() domain.FolderMember {
	return domain.FolderMember{
		ID:        m.ID,
		FolderID:  m.FolderID,
		UserID:    m.UserID,
		Role:      domain.FolderRole(m.Role),
		Caps:      domain.FolderCap(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
