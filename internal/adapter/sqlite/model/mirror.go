package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type MirrorSyncEvent struct {
	ID         uint      `gorm:"primaryKey"`
	RepoID     uint      `gorm:"not null;index:idx_mirror_sync_events_repo,priority:1"`
	Direction  string    `gorm:"size:8;not null;default:''"`
	Trigger    string    `gorm:"size:16;not null;default:''"`
	Status     string    `gorm:"size:16;not null;default:''"`
	Error      string    `gorm:"size:1024;not null;default:''"`
	StartedAt  time.Time `gorm:"not null;index:idx_mirror_sync_events_repo,priority:2"`
	FinishedAt *time.Time
	DurationMs int64 `gorm:"not null;default:0"`
	ActorID    uint  `gorm:"not null;default:0"`
}

func (MirrorSyncEvent) TableName() string {
	return "mirror_sync_events"
}

func (m MirrorSyncEvent) ToDomain() domain.MirrorSyncEvent {
	return domain.MirrorSyncEvent{
		ID:         m.ID,
		RepoID:     m.RepoID,
		Direction:  m.Direction,
		Trigger:    m.Trigger,
		Status:     m.Status,
		Error:      m.Error,
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
		DurationMs: m.DurationMs,
		ActorID:    m.ActorID,
	}
}

func MirrorSyncEventFromDomain(ev *domain.MirrorSyncEvent) *MirrorSyncEvent {
	return &MirrorSyncEvent{
		ID:         ev.ID,
		RepoID:     ev.RepoID,
		Direction:  ev.Direction,
		Trigger:    ev.Trigger,
		Status:     ev.Status,
		Error:      ev.Error,
		StartedAt:  ev.StartedAt,
		FinishedAt: ev.FinishedAt,
		DurationMs: ev.DurationMs,
		ActorID:    ev.ActorID,
	}
}

type MirrorTarget struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	RepoID     uint   `gorm:"not null;index:idx_mirror_targets_repo"`
	URL        string `gorm:"column:url;size:2048;not null;default:''"`
	Username   string `gorm:"size:255;not null;default:''"`
	Password   string `gorm:"size:2048;not null;default:''"`
	Direction  string `gorm:"size:8;not null;default:'push'"`
	Enabled    bool   `gorm:"not null;default:true"`
	LastSyncAt *time.Time
	LastError  string `gorm:"size:1024;not null;default:''"`
}

func (MirrorTarget) TableName() string {
	return "mirror_targets"
}

func (m MirrorTarget) ToDomain() domain.MirrorTarget {
	return domain.MirrorTarget{
		ID:         m.ID,
		RepoID:     m.RepoID,
		URL:        m.URL,
		Username:   m.Username,
		Password:   m.Password,
		Direction:  m.Direction,
		Enabled:    m.Enabled,
		LastSyncAt: m.LastSyncAt,
		LastError:  m.LastError,
		CreatedAt:  m.CreatedAt,
	}
}

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
