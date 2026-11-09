package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type MirrorGroup struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string `gorm:"size:255;not null"`
	UpstreamURL string `gorm:"column:upstream_url;size:2048;not null;default:''"`
	Direction   string `gorm:"size:8;not null;default:'pull'"`
	Interval    string `gorm:"column:sync_interval;size:32;not null;default:''"`
}

func (MirrorGroup) TableName() string {
	return "mirror_groups"
}

func (m MirrorGroup) ToDomain() domain.MirrorGroup {
	return domain.MirrorGroup{
		ID:          m.ID,
		Name:        m.Name,
		UpstreamURL: m.UpstreamURL,
		Direction:   m.Direction,
		Interval:    m.Interval,
		CreatedAt:   m.CreatedAt,
	}
}
