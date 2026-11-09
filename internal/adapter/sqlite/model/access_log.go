package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type AccessLog struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Action    string `gorm:"size:32;not null;index"`
	IP        string `gorm:"size:64;not null;default:''"`
	UserAgent string `gorm:"size:512;not null;default:''"`
	UserID    *uint
	Username  string `gorm:"size:64;not null;default:''"`
	Path      string `gorm:"size:1024;not null;default:''"`
	Repo      string `gorm:"size:512;not null;default:''"`
	Ref       string `gorm:"size:255;not null;default:''"`
}

func (AccessLog) TableName() string {
	return "access_logs"
}

func (m AccessLog) ToDomain() domain.AccessLog {
	return domain.AccessLog{
		ID:        m.ID,
		Action:    m.Action,
		IP:        m.IP,
		UserAgent: m.UserAgent,
		UserID:    m.UserID,
		Username:  m.Username,
		Path:      m.Path,
		Repo:      m.Repo,
		Ref:       m.Ref,
		CreatedAt: m.CreatedAt,
	}
}
