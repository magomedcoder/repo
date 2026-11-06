package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type TeamRepoGrant struct {
	gormutil.Repo[model.TeamRepoGrant]
}

func NewTeamRepoGrantRepo(db *gorm.DB) *TeamRepoGrant {
	return &TeamRepoGrant{Repo: gormutil.NewRepo[model.TeamRepoGrant](db)}
}

func (s *TeamRepoGrant) Upsert(g *domain.TeamRepoGrant) error {
	ctx := context.Background()
	existing, err := s.FindByWhereWithQuery(ctx, "team_id = ? AND repo_id = ?", []any{g.TeamID, g.RepoID}, gormutil.WithUnscoped())
	if err == nil {
		existing.DeletedAt = gorm.DeletedAt{}
		existing.Role = string(g.Role)
		if err := s.Save(ctx, existing, gormutil.WithUnscoped()); err != nil {
			return err
		}
		*g = existing.ToDomain()
		return nil
	}

	m := model.TeamRepoGrant{
		TeamID: g.TeamID,
		RepoID: g.RepoID,
		Role:   string(g.Role),
	}

	if err := s.Repo.Create(ctx, &m); err != nil {
		return err
	}
	*g = m.ToDomain()

	return nil
}

func (s *TeamRepoGrant) Delete(teamID, repoID uint) error {
	return s.DeleteWhere(context.Background(), "team_id = ? AND repo_id = ?", teamID, repoID)
}

func (s *TeamRepoGrant) ListByTeam(teamID uint) ([]domain.TeamRepoGrant, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "team_id = ?", "id ASC", teamID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.TeamRepoGrant, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *TeamRepoGrant) ListByRepoUser(repoID, userID uint) ([]domain.TeamRepoGrant, error) {
	models, err := s.FindAll(context.Background(),
		gormutil.WithJoins("JOIN team_members ON team_members.team_id = team_repo_grants.team_id AND team_members.deleted_at IS NULL"),
		gormutil.WithWhere("team_repo_grants.repo_id = ? AND team_members.user_id = ?", repoID, userID),
	)
	if err != nil {
		return nil, err
	}

	out := make([]domain.TeamRepoGrant, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
