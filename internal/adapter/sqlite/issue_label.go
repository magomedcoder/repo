package sqlite

import (
	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type issueLabelModel struct {
	IssueID uint `gorm:"primaryKey"`
	LabelID uint `gorm:"primaryKey"`
}

func (issueLabelModel) TableName() string {
	return "issue_labels"
}

func (s *IssueStore) SetIssueLabels(issueID uint, labelIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", issueID).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		for _, id := range labelIDs {
			row := issueLabelModel{IssueID: issueID, LabelID: id}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *IssueStore) ListLabelsByIssues(issueIDs []uint) (map[uint][]domain.Label, error) {
	out := map[uint][]domain.Label{}
	if len(issueIDs) == 0 {
		return out, nil
	}

	var links []issueLabelModel
	if err := s.db.Where("issue_id IN ?", issueIDs).Find(&links).Error; err != nil {
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

	var models []labelModel
	if err := s.db.Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}

	byID := map[uint]domain.Label{}
	for _, m := range models {
		byID[m.ID] = m.toDomain()
	}

	for _, link := range links {
		if label, ok := byID[link.LabelID]; ok {
			out[link.IssueID] = append(out[link.IssueID], label)
		}
	}

	return out, nil
}
