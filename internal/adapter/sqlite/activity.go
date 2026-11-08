package sqlite

import (
	"context"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type Activity struct {
	gormutil.Repo[model.Activity]
}

func NewActivityRepo(db *gorm.DB) *Activity {
	return &Activity{Repo: gormutil.NewRepo[model.Activity](db)}
}

func (s *Activity) Create(ev *domain.ActivityEvent) error {
	m := &model.Activity{
		ActorID: ev.ActorID,
		OwnerID: ev.OwnerID,
		RepoID:  ev.RepoID,
		Type:    ev.Type,
		Summary: ev.Summary,
		Ref:     ev.Ref,
	}
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	*ev = m.ToDomain()

	return nil
}

func (s *Activity) ListByOwner(ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	if limit <= 0 {
		limit = 30
	}

	models, err := s.FindAll(context.Background(),
		gormutil.WithWhere("owner_id = ?", ownerID),
		gormutil.WithOrder("created_at DESC"),
		gormutil.WithLimit(limit),
	)
	if err != nil {
		return nil, err
	}

	return activityRowsToDomain(models), nil
}

func (s *Activity) ListPublicByOwner(ownerID uint, limit int) ([]domain.ActivityEvent, error) {
	if limit <= 0 {
		limit = 30
	}

	models, err := s.FindAll(context.Background(),
		gormutil.WithSelect("activity_events.*"),
		gormutil.WithJoins("JOIN repositories ON repositories.id = activity_events.repo_id AND repositories.deleted_at IS NULL AND repositories.is_private = ?", false),
		gormutil.WithWhere("activity_events.owner_id = ?", ownerID),
		gormutil.WithOrder("activity_events.created_at DESC, activity_events.id DESC"),
		gormutil.WithLimit(limit),
	)
	if err != nil {
		return nil, err
	}

	return activityRowsToDomain(models), nil
}

func (s *Activity) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return s.DeleteWhereRows(ctx, "created_at < ?", before.In(time.Local))
}

func (s *Activity) DeleteByRepo(ctx context.Context, repoID uint) error {
	if repoID == 0 {
		return nil
	}

	_, err := s.DeleteWhereRows(ctx, "repo_id = ?", repoID)
	return err
}

func activityRowsToDomain(models []model.Activity) []domain.ActivityEvent {
	out := make([]domain.ActivityEvent, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}
	return out
}
