package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type AccessToken struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UserID     uint   `gorm:"index;not null"`
	Name       string `gorm:"size:100;not null"`
	TokenHash  string `gorm:"uniqueIndex;size:67;not null"`
	Prefix     string `gorm:"size:16;not null"`
	Scope      string `gorm:"size:32;not null;default:git:read"`
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
}

func (AccessToken) TableName() string {
	return "access_tokens"
}

func (m AccessToken) ToDomain() domain.AccessToken {
	return domain.AccessToken{
		ID:         m.ID,
		UserID:     m.UserID,
		Name:       m.Name,
		TokenHash:  m.TokenHash,
		Prefix:     m.Prefix,
		Scope:      m.Scope,
		CreatedAt:  m.CreatedAt,
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
	}
}
