package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

type MirrorSync struct {
	gormutil.Repo[model.MirrorSyncEvent]
}

func NewMirrorSyncRepo(db *gorm.DB) *MirrorSync {
	return &MirrorSync{Repo: gormutil.NewRepo[model.MirrorSyncEvent](db)}
}

func (s *MirrorSync) Create(ctx context.Context, ev *domain.MirrorSyncEvent) error {
	m := model.MirrorSyncEventFromDomain(ev)
	if err := s.Repo.Create(ctx, m); err != nil {
		return err
	}

	ev.ID = m.ID

	return nil
}

func (s *MirrorSync) Update(ctx context.Context, ev *domain.MirrorSyncEvent) error {
	_, err := s.UpdateByID(ctx, ev.ID, map[string]any{
		"status":      ev.Status,
		"error":       ev.Error,
		"finished_at": ev.FinishedAt,
		"duration_ms": ev.DurationMs,
	})

	return err
}

func (s *MirrorSync) ListByRepo(ctx context.Context, repoID uint, limit int) ([]domain.MirrorSyncEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	models, err := s.FindAll(ctx,
		gormutil.WithWhere("repo_id = ?", repoID),
		gormutil.WithOrder("started_at DESC, id DESC"),
		gormutil.WithLimit(limit),
	)
	if err != nil {
		return nil, err
	}

	out := make([]domain.MirrorSyncEvent, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *MirrorSync) Prune(ctx context.Context, repoID uint, keep int) error {
	if keep <= 0 {
		return nil
	}

	return s.DB(ctx).Exec(`DELETE FROM mirror_sync_events WHERE repo_id = ? AND id NOT IN (SELECT id FROM mirror_sync_events WHERE repo_id = ? ORDER BY started_at DESC, id DESC LIMIT ?)`, repoID, repoID, keep).Error
}

func (s *MirrorSync) DeleteByRepo(ctx context.Context, repoID uint) error {
	return s.DeleteWhere(ctx, "repo_id = ?", repoID)
}
