package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type TeamMember struct {
	gormutil.Repo[model.TeamMember]
}

func NewTeamMemberRepo(db *gorm.DB) *TeamMember {
	return &TeamMember{Repo: gormutil.NewRepo[model.TeamMember](db)}
}

func (s *TeamMember) Create(mem *domain.TeamMember) error {
	row := model.TeamMember{TeamID: mem.TeamID, UserID: mem.UserID}
	if err := s.Repo.Create(context.Background(), &row); err != nil {
		return err
	}
	*mem = row.ToDomain()

	return nil
}

func (s *TeamMember) Delete(teamID, userID uint) error {
	return s.DeleteWhere(context.Background(), "team_id = ? AND user_id = ?", teamID, userID)
}

func (s *TeamMember) Find(teamID, userID uint) (*domain.TeamMember, error) {
	row, err := s.FindByWhere(context.Background(), "team_id = ? AND user_id = ?", teamID, userID)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *TeamMember) ListByTeam(teamID uint) ([]domain.TeamMember, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "team_id = ?", "id ASC", teamID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.TeamMember, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *TeamMember) DeleteByTeam(teamID uint) error {
	return s.DeleteWhere(context.Background(), "team_id = ?", teamID)
}
