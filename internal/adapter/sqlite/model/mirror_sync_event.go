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
