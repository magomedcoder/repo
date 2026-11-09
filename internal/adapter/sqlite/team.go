package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

type Team struct {
	gormutil.Repo[model.Team]
}

func NewTeamRepo(db *gorm.DB) *Team {
	return &Team{Repo: gormutil.NewRepo[model.Team](db)}
}

func (s *Team) Create(t *domain.Team) error {
	m := model.Team{
		OwnerUserID: t.OwnerUserID,
		Name:        t.Name,
		Slug:        t.Slug,
	}
	if err := s.Repo.Create(context.Background(), &m); err != nil {
		return err
	}
	*t = m.ToDomain()

	return nil
}

func (s *Team) Update(t *domain.Team) error {
	m := model.Team{
		ID:          t.ID,
		OwnerUserID: t.OwnerUserID,
		Name:        t.Name,
		Slug:        t.Slug,
	}
	if err := s.Save(context.Background(), &m); err != nil {
		return err
	}
	*t = m.ToDomain()

	return nil
}

func (s *Team) Delete(id uint) error {
	return s.DeleteByID(context.Background(), id)
}

func (s *Team) FindByID(id uint) (*domain.Team, error) {
	row, err := s.Repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *Team) FindByOwnerSlug(ownerUserID uint, slug string) (*domain.Team, error) {
	row, err := s.FindByWhere(context.Background(), "owner_user_id = ? AND slug = ?", ownerUserID, slug)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *Team) ListByOwner(ownerUserID uint) ([]domain.Team, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "owner_user_id = ?", "name ASC", ownerUserID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Team, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *Team) ExistsByOwnerSlug(ownerUserID uint, slug string) (bool, error) {
	return s.QueryExist(context.Background(), "owner_user_id = ? AND slug = ?", ownerUserID, slug)
}
