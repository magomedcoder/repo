package domain

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrMirrorDiverged           = errors.New("mirror: remote has diverged (non-fast-forward)")
	ErrMirrorTooLarge           = errors.New("mirror: repository exceeds configured max size")
	ErrMirrorLFSMissing         = errors.New("mirror: git-lfs binary not found")
	ErrMirrorLFSUnavailable     = errors.New("mirror: LFS sync over system git is unavailable")
	ErrMirrorPartialUnavailable = errors.New("mirror: partial clone over system git is unavailable")
	ErrMirrorUnsigned           = errors.New("mirror: upstream tip is not a verified signed commit")
)

func RedactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if !strings.Contains(raw, "://") {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}

	u.User = nil

	return u.String()
}

const (
	MirrorStatusIdle    = "idle"
	MirrorStatusSyncing = "syncing"
	MirrorStatusOK      = "ok"
	MirrorStatusError   = "error"
	MirrorStatusPaused  = "paused"
	MirrorStatusSkipped = "skipped"
)

const (
	MirrorConflictForce  = "force"
	MirrorConflictReject = "reject"
)

const (
	MirrorTriggerManual   = "manual"
	MirrorTriggerSchedule = "schedule"
	MirrorTriggerPush     = "push"
	MirrorTriggerMerge    = "merge"
	MirrorTriggerConfig   = "configure"
	MirrorTriggerBulk     = "bulk"
	MirrorTriggerCreate   = "create"
)

const (
	MirrorEventOK      = "ok"
	MirrorEventFailed  = "failed"
	MirrorEventSkipped = "skipped"
	MirrorEventRunning = "running"
)

const (
	ActivityMirrorConfigure = "mirror_configured"
	ActivityMirrorSync      = "mirror_synced"
)

type MirrorSyncEvent struct {
	ID         uint
	RepoID     uint
	Direction  string
	Trigger    string
	Status     string
	Error      string
	StartedAt  time.Time
	FinishedAt *time.Time
	DurationMs int64
	ActorID    uint
}

type MirrorGroup struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	UpstreamURL string    `json:"upstream_url"`
	Direction   string    `json:"direction"`
	Interval    string    `json:"interval"`
	CreatedAt   time.Time `json:"created_at"`
}

type MirrorListFilter struct {
	Direction string
	Status    string
	OnlyError bool
	Query     string
	GroupID   uint
	Limit     int
	Offset    int
}

type MirrorSyncStore interface {
	Create(ctx context.Context, ev *MirrorSyncEvent) error

	Update(ctx context.Context, ev *MirrorSyncEvent) error

	ListByRepo(ctx context.Context, repoID uint, limit int) ([]MirrorSyncEvent, error)

	Prune(ctx context.Context, repoID uint, keep int) error

	DeleteByRepo(ctx context.Context, repoID uint) error
}

type MirrorGroupStore interface {
	Create(ctx context.Context, g *MirrorGroup) error

	Delete(ctx context.Context, id uint) error

	FindByID(ctx context.Context, id uint) (*MirrorGroup, error)

	List(ctx context.Context) ([]MirrorGroup, error)
}
