package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type Session struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	Token      string `gorm:"uniqueIndex;size:128;not null"`
	UserID     uint   `gorm:"index;not null;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User       User   `gorm:"foreignKey:UserID;references:ID"`
	IP         string `gorm:"size:64;not null;default:''"`
	UserAgent  string `gorm:"size:512;not null;default:''"`
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

func (Session) TableName() string {
	return "sessions"
}

func (m Session) ToDomain() *domain.Session {
	lastSeen := m.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = m.CreatedAt
	}

	return &domain.Session{
		ID:         m.ID,
		Token:      m.Token,
		UserID:     m.UserID,
		IP:         m.IP,
		UserAgent:  m.UserAgent,
		ExpiresAt:  m.ExpiresAt,
		LastSeenAt: lastSeen,
		CreatedAt:  m.CreatedAt,
	}
}
