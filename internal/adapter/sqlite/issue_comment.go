package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
)

func (s *Issue) issueComments() *gormutil.Repo[model.IssueComment] {
	r := gormutil.NewRepo[model.IssueComment](s.Db)
	return &r
}

func (s *Issue) CountComments(issueID uint) (int, error) {
	n, err := s.issueComments().QueryCount(context.Background(), "issue_id = ?", issueID)
	if err != nil {
		return 0, err
	}

	return int(n), nil
}

func (s *Issue) CreateComment(comment *domain.IssueComment) error {
	m := model.IssueCommentFromDomain(comment)
	if err := s.issueComments().Create(context.Background(), m); err != nil {
		return err
	}

	*comment = m.ToDomain()
	return nil
}

func (s *Issue) UpdateComment(comment *domain.IssueComment) error {
	m := model.IssueCommentFromDomain(comment)
	if err := s.issueComments().Save(context.Background(), m); err != nil {
		return err
	}

	*comment = m.ToDomain()
	return nil
}

func (s *Issue) DeleteComment(id uint) error {
	return s.issueComments().DeleteByID(context.Background(), id)
}

func (s *Issue) FindComment(id uint) (*domain.IssueComment, error) {
	row, err := s.issueComments().FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	c := row.ToDomain()
	return &c, nil
}

func (s *Issue) ListComments(issueID uint) ([]domain.IssueComment, error) {
	models, err := s.issueComments().FindByWhereAndOrder(context.Background(), "issue_id = ?", "created_at asc", issueID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.IssueComment, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
