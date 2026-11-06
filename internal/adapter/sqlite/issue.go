package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type Issue struct {
	gormutil.Repo[model.Issue]
}

func NewIssueRepo(db *gorm.DB) *Issue {
	return &Issue{Repo: gormutil.NewRepo[model.Issue](db)}
}

func (s *Issue) Create(issue *domain.Issue) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		var seq int
		if err := tx.Raw(`UPDATE repositories SET issue_seq = issue_seq + 1 WHERE id = ? RETURNING issue_seq`, issue.RepositoryID).Scan(&seq).Error; err != nil {
			return err
		}
		if seq == 0 {
			return domain.ErrNotFound
		}

		issue.Number = seq
		if issue.State == "" {
			issue.State = "open"
		}

		m := model.IssueFromDomain(issue)
		if err := tx.Create(m).Error; err != nil {
			return err
		}

		*issue = m.ToDomain()
		return nil
	})
}

func (s *Issue) Update(issue *domain.Issue) error {
	m := model.IssueFromDomain(issue)
	if err := s.Save(context.Background(), m); err != nil {
		return err
	}

	*issue = m.ToDomain()
	return nil
}

func (s *Issue) Delete(id uint) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", id).Delete(&model.IssueLabel{}).Error; err != nil {
			return err
		}

		if err := tx.Where("issue_id = ?", id).Delete(&model.IssueComment{}).Error; err != nil {
			return err
		}

		return tx.Delete(&model.Issue{}, id).Error
	})
}

func (s *Issue) FindByRepoNumber(repoID uint, number int) (*domain.Issue, error) {
	row, err := s.FindByWhere(context.Background(), "repository_id = ? AND number = ?", repoID, number)
	if err != nil {
		return nil, mapDBErr(err)
	}

	issue := row.ToDomain()
	return &issue, nil
}

func (s *Issue) ListByRepo(repoID uint, state string, offset, limit int) ([]domain.Issue, error) {
	if limit <= 0 {
		limit = 30
	}

	opts := []func(*gorm.DB) *gorm.DB{
		gormutil.WithWhere("repository_id = ?", repoID),
		gormutil.WithOrder("number desc"),
		gormutil.LimitOffset(limit, offset),
	}
	if state == "open" || state == "closed" {
		opts = append([]func(*gorm.DB) *gorm.DB{gormutil.WithWhere("state = ?", state)}, opts...)
	}

	models, err := s.FindAll(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Issue, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
