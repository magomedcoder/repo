package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type MirrorGroup struct {
	gormutil.Repo[model.MirrorGroup]
}

func NewMirrorGroupRepo(db *gorm.DB) *MirrorGroup {
	return &MirrorGroup{Repo: gormutil.NewRepo[model.MirrorGroup](db)}
}

func (s *MirrorGroup) Create(ctx context.Context, g *domain.MirrorGroup) error {
	m := &model.MirrorGroup{
		Name:        g.Name,
		UpstreamURL: g.UpstreamURL,
		Direction:   g.Direction,
		Interval:    g.Interval,
	}
	if err := s.Repo.Create(ctx, m); err != nil {
		return err
	}

	g.ID = m.ID
	g.CreatedAt = m.CreatedAt

	return nil
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
