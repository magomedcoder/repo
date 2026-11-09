package model

import (
	"encoding/json"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type AccessToken struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UserID     uint   `gorm:"index;not null"`
	Name       string `gorm:"size:100;not null"`
	TokenHash  string `gorm:"uniqueIndex;size:67;not null"`
	Prefix     string `gorm:"size:16;not null"`
	Scope      string `gorm:"size:32;not null;default:git:read"`
	RepoIDs    string `gorm:"type:text;not null;default:''"`
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
		RepoIDs:    DecodeRepoIDs(m.RepoIDs),
		CreatedAt:  m.CreatedAt,
		LastUsedAt: m.LastUsedAt,
		ExpiresAt:  m.ExpiresAt,
	}
}

func EncodeRepoIDs(ids []uint) string {
	if len(ids) == 0 {
		return ""
	}

	b, err := json.Marshal(ids)
	if err != nil {
		return ""
	}

	return string(b)
}

func DecodeRepoIDs(raw string) []uint {
	if raw == "" {
		return nil
	}

	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []uint{0}
	}

	if len(ids) == 0 {
		return nil
	}

	return ids
}
