package sqlite

import (
	"context"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

func (s *Issue) issueLabels() *gormutil.Repo[model.IssueLabel] {
	r := gormutil.NewRepo[model.IssueLabel](s.Db)
	return &r
}

func (s *Issue) SetIssueLabels(issueID uint, labelIDs []uint) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", issueID).Delete(&model.IssueLabel{}).Error; err != nil {
			return err
		}

		for _, id := range labelIDs {
			row := model.IssueLabel{IssueID: issueID, LabelID: id}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Issue) ListLabelsByIssues(issueIDs []uint) (map[uint][]domain.Label, error) {
	out := map[uint][]domain.Label{}
	if len(issueIDs) == 0 {
		return out, nil
	}

	ctx := context.Background()
	il := s.issueLabels()
	links, err := il.FindAll(ctx, gormutil.WithWhere("issue_id IN ?", issueIDs))
	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return out, nil
	}

	ids := make([]uint, 0, len(links))
	seen := map[uint]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.LabelID]; ok {
			continue
		}

		seen[link.LabelID] = struct{}{}
		ids = append(ids, link.LabelID)
	}

	labels := s.labels()
	labelRows, err := labels.FindAll(ctx, gormutil.WithWhere("id IN ?", ids))
	if err != nil {
		return nil, err
	}

	byID := map[uint]domain.Label{}
	for _, m := range labelRows {
		byID[m.ID] = m.ToDomain()
	}

	for _, link := range links {
		if label, ok := byID[link.LabelID]; ok {
			out[link.IssueID] = append(out[link.IssueID], label)
		}
	}

	return out, nil
}
