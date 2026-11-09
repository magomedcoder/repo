package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

type RepoCollaborator struct {
	gormutil.Repo[model.RepoCollaborator]
}

func NewRepoCollaboratorRepo(db *gorm.DB) *RepoCollaborator {
	return &RepoCollaborator{Repo: gormutil.NewRepo[model.RepoCollaborator](db)}
}

func (s *RepoCollaborator) Upsert(c *domain.RepoCollaborator) error {
	ctx := context.Background()
	existing, err := s.FindByWhereWithQuery(ctx, "repo_id = ? AND user_id = ?", []any{c.RepoID, c.UserID}, gormutil.WithUnscoped())
	if err == nil {
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Role = string(c.Role)
		if err := s.Save(ctx, existing, gormutil.WithUnscoped()); err != nil {
			return err
		}
		*c = existing.ToDomain()
		return nil
	}

	m := model.RepoCollaborator{
		RepoID: c.RepoID,
		UserID: c.UserID,
		Role:   string(c.Role),
	}
	if err := s.Repo.Create(ctx, &m); err != nil {
		return err
	}

	*c = m.ToDomain()

	return nil
}

func (s *RepoCollaborator) Delete(repoID, userID uint) error {
	return s.DeleteWhere(context.Background(), "repo_id = ? AND user_id = ?", repoID, userID)
}

func (s *RepoCollaborator) DeleteByUserID(userID uint) error {
	return s.DeleteWhere(context.Background(), "user_id = ?", userID)
}

func (s *RepoCollaborator) Find(repoID, userID uint) (*domain.RepoCollaborator, error) {
	row, err := s.FindByWhere(context.Background(), "repo_id = ? AND user_id = ?", repoID, userID)
	if err != nil {
		return nil, mapDBErr(err)
	}

	out := row.ToDomain()

	return &out, nil
}

func (s *RepoCollaborator) ListByRepo(repoID uint) ([]domain.RepoCollaborator, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "repo_id = ?", "id ASC", repoID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.RepoCollaborator, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
