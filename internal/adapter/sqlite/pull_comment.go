package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
)

func (s *PullRequest) pullComments() *gormutil.Repo[model.PullComment] {
	r := gormutil.NewRepo[model.PullComment](s.Db)
	return &r
}

func (s *PullRequest) CountComments(pullID uint) (int, error) {
	n, err := s.pullComments().QueryCount(context.Background(), "pull_id = ?", pullID)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func (s *PullRequest) CreateComment(comment *domain.PullRequestComment) error {
	m := model.PullCommentFromDomain(comment)
	if err := s.pullComments().Create(context.Background(), m); err != nil {
		return err
	}

	*comment = m.ToDomain()

	return nil
}

func (s *PullRequest) UpdateComment(comment *domain.PullRequestComment) error {
	m := model.PullCommentFromDomain(comment)
	if err := s.pullComments().Save(context.Background(), m); err != nil {
		return err
	}

	*comment = m.ToDomain()

	return nil
}

func (s *PullRequest) DeleteComment(id uint) error {
	return s.pullComments().DeleteByID(context.Background(), id)
}

func (s *PullRequest) FindComment(id uint) (*domain.PullRequestComment, error) {
	row, err := s.pullComments().FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	c := row.ToDomain()

	return &c, nil
}

func (s *PullRequest) ListComments(pullID uint) ([]domain.PullRequestComment, error) {
	models, err := s.pullComments().FindByWhereAndOrder(context.Background(), "pull_id = ?", "id asc", pullID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.PullRequestComment, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
