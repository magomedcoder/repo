package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type Activity struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	ActorID   uint `gorm:"not null"`
	OwnerID   uint `gorm:"not null"`
	RepoID    *uint
	Type      string `gorm:"size:32;not null"`
	Summary   string `gorm:"size:512;not null"`
	Ref       string `gorm:"size:255"`
}

func (Activity) TableName() string {
	return "activity_events"
}

func (m Activity) ToDomain() domain.ActivityEvent {
	return domain.ActivityEvent{
		ID:        m.ID,
		ActorID:   m.ActorID,
		OwnerID:   m.OwnerID,
		RepoID:    m.RepoID,
		Type:      m.Type,
		Summary:   m.Summary,
		Ref:       m.Ref,
		CreatedAt: m.CreatedAt,
	}
}
