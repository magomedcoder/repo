package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type labelModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_label_name"`
	Name         string `gorm:"size:50;not null;uniqueIndex:idx_label_name"`
	Color        string `gorm:"size:16;not null"`
}

func (labelModel) TableName() string {
	return "labels"
}

func (s *IssueStore) CreateLabel(label *domain.Label) error {
	model := labelFromDomain(label)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*label = model.toDomain()

	return nil
}

func (s *IssueStore) UpdateLabel(label *domain.Label) error {
	model := labelFromDomain(label)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*label = model.toDomain()

	return nil
}

func (s *IssueStore) DeleteLabel(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", id).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&labelModel{}, id).Error
	})
}

func (s *IssueStore) FindLabel(id uint) (*domain.Label, error) {
	var model labelModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, mapDBErr(err)
	}

	l := model.toDomain()

	return &l, nil
}

func (s *IssueStore) ListLabels(repoID uint) ([]domain.Label, error) {
	var models []labelModel
	if err := s.db.Where("repository_id = ?", repoID).Order("name asc").Find(&models).Error; err != nil {
		return nil, err
	}

	return labelsToDomain(models), nil
}

func (s *IssueStore) FindLabelByRepoName(repoID uint, name string) (*domain.Label, error) {
	var model labelModel
	if err := s.db.Where("repository_id = ? AND name = ?", repoID, name).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}

	l := model.toDomain()

	return &l, nil
}

func labelFromDomain(l *domain.Label) *labelModel {
	return &labelModel{
		ID:           l.ID,
		RepositoryID: l.RepositoryID,
		Name:         l.Name,
		Color:        l.Color,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func (m labelModel) toDomain() domain.Label {
	return domain.Label{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Name:         m.Name,
		Color:        m.Color,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func labelsToDomain(models []labelModel) []domain.Label {
	out := make([]domain.Label, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out
}
