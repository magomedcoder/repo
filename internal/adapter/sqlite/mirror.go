package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
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

type MirrorTarget struct {
	gormutil.Repo[model.MirrorTarget]
}

func NewMirrorTargetRepo(db *gorm.DB) *MirrorTarget {
	return &MirrorTarget{Repo: gormutil.NewRepo[model.MirrorTarget](db)}
}

func (s *MirrorTarget) Create(ctx context.Context, t *domain.MirrorTarget) error {
	m := &model.MirrorTarget{
		RepoID:    t.RepoID,
		URL:       t.URL,
		Username:  t.Username,
		Password:  t.Password,
		Direction: t.Direction,
		Enabled:   t.Enabled,
	}

	if err := s.Repo.Create(ctx, m); err != nil {
		return err
	}

	if !t.Enabled {
		if _, err := s.UpdateColumnByID(ctx, m.ID, "enabled", false); err != nil {
			return err
		}
	}

	t.ID = m.ID
	t.CreatedAt = m.CreatedAt

	return nil
}

func (s *MirrorTarget) Update(ctx context.Context, t *domain.MirrorTarget) error {
	_, err := s.UpdateWhere(ctx, map[string]any{
		"url":          t.URL,
		"username":     t.Username,
		"password":     t.Password,
		"direction":    t.Direction,
		"enabled":      t.Enabled,
		"last_sync_at": t.LastSyncAt,
		"last_error":   t.LastError,
	}, "id = ? AND repo_id = ?", t.ID, t.RepoID)
	return err
}

func (s *MirrorTarget) Delete(ctx context.Context, repoID, id uint) error {
	return s.DeleteWhere(ctx, "id = ? AND repo_id = ?", id, repoID)
}

func (s *MirrorTarget) FindByID(ctx context.Context, repoID, id uint) (*domain.MirrorTarget, error) {
	m, err := s.FindByWhere(ctx, "id = ? AND repo_id = ?", id, repoID)
	if err != nil {
		return nil, mapDBErr(err)
	}

	t := m.ToDomain()

	return &t, nil
}

func (s *MirrorTarget) ListByRepo(ctx context.Context, repoID uint) ([]domain.MirrorTarget, error) {
	models, err := s.FindByWhereAndOrder(ctx, "repo_id = ?", "id ASC", repoID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.MirrorTarget, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

type MirrorGroup struct {
	gormutil.Repo[model.MirrorGroup]
}

func NewMirrorGroupRepo(db *gorm.DB) *MirrorGroup {
	return &MirrorGroup{Repo: gormutil.NewRepo[model.MirrorGroup](db)}
}

func (s *MirrorGroup) Create(ctx context.Context, g *domain.MirrorGroup) error {
	m := &model.MirrorGroup{Name: g.Name, UpstreamURL: g.UpstreamURL, Direction: g.Direction, Interval: g.Interval}
	if err := s.Repo.Create(ctx, m); err != nil {
		return err
	}

	g.ID = m.ID
	g.CreatedAt = m.CreatedAt

	return nil
}

func (s *MirrorGroup) Update(ctx context.Context, g *domain.MirrorGroup) error {
	_, err := s.UpdateByID(ctx, g.ID, map[string]any{
		"name":          g.Name,
		"upstream_url":  g.UpstreamURL,
		"direction":     g.Direction,
		"sync_interval": g.Interval,
	})
	return err
}

func (s *MirrorGroup) Delete(ctx context.Context, id uint) error {
	return s.DeleteByID(ctx, id)
}

func (s *MirrorGroup) FindByID(ctx context.Context, id uint) (*domain.MirrorGroup, error) {
	m, err := s.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	g := m.ToDomain()

	return &g, nil
}

func (s *MirrorGroup) List(ctx context.Context) ([]domain.MirrorGroup, error) {
	models, err := s.FindAll(ctx, gormutil.WithOrder("name ASC, id ASC"))
	if err != nil {
		return nil, err
	}

	out := make([]domain.MirrorGroup, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
