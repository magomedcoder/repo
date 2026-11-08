package sqlite

import (
	"context"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type PullRequest struct {
	gormutil.Repo[model.PullRequest]
}

func NewPullRequestRepo(db *gorm.DB) *PullRequest {
	return &PullRequest{Repo: gormutil.NewRepo[model.PullRequest](db)}
}

func (s *PullRequest) Create(pr *domain.PullRequest) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		var seq int
		if err := tx.Raw(
			`UPDATE repositories SET pull_seq = pull_seq + 1 WHERE id = ? RETURNING pull_seq`,
			pr.RepositoryID,
		).Scan(&seq).Error; err != nil {
			return err
		}
		if seq == 0 {
			return domain.ErrNotFound
		}

		pr.Number = seq
		if pr.State == "" {
			pr.State = "open"
		}

		m := model.PullRequestFromDomain(pr)
		if err := tx.Create(m).Error; err != nil {
			return err
		}

		*pr = m.ToDomain()
		return nil
	})
}

func (s *PullRequest) Update(pr *domain.PullRequest) error {
	m := model.PullRequestFromDomain(pr)
	if err := s.Save(context.Background(), m); err != nil {
		return err
	}

	*pr = m.ToDomain()
	return nil
}

func (s *PullRequest) MarkMerged(id, mergedBy uint, at time.Time) error {
	res := s.DB(context.Background()).Model(&model.PullRequest{}).
		Where("id = ? AND state = ?", id, "open").
		Updates(map[string]any{
			"state":      "merged",
			"merged_at":  at,
			"merged_by":  mergedBy,
			"updated_at": at,
		})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return nil
	}

	return nil
}

func (s *PullRequest) Delete(id uint) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Where("pull_id = ?", id).Delete(&model.PullComment{}).Error; err != nil {
			return err
		}

		return tx.Delete(&model.PullRequest{}, id).Error
	})
}

func (s *PullRequest) FindByRepoNumber(repoID uint, number int) (*domain.PullRequest, error) {
	row, err := s.FindByWhere(context.Background(), "repository_id = ? AND number = ?", repoID, number)
	if err != nil {
		return nil, mapDBErr(err)
	}

	pr := row.ToDomain()
	return &pr, nil
}

func (s *PullRequest) ListByRepo(repoID uint, state string, offset, limit int) ([]domain.PullRequest, error) {
	if limit <= 0 {
		limit = 30
	}

	opts := []func(*gorm.DB) *gorm.DB{
		gormutil.WithWhere("repository_id = ?", repoID),
		gormutil.WithOrder("number desc"),
		gormutil.LimitOffset(limit, offset),
	}
	switch state {
	case "open", "closed", "merged":
		opts = append([]func(*gorm.DB) *gorm.DB{gormutil.WithWhere("state = ?", state)}, opts...)
	}

	models, err := s.FindAll(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	out := make([]domain.PullRequest, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
