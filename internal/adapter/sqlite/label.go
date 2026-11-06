package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

func (s *Issue) labels() *gormutil.Repo[model.Label] {
	r := gormutil.NewRepo[model.Label](s.Db)
	return &r
}

func (s *Issue) CreateLabel(label *domain.Label) error {
	m := model.LabelFromDomain(label)
	if err := s.labels().Create(context.Background(), m); err != nil {
		return err
	}

	*label = m.ToDomain()

	return nil
}

func (s *Issue) UpdateLabel(label *domain.Label) error {
	m := model.LabelFromDomain(label)
	if err := s.labels().Save(context.Background(), m); err != nil {
		return err
	}

	*label = m.ToDomain()

	return nil
}

func (s *Issue) DeleteLabel(id uint) error {
	return s.Txx(context.Background(), func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", id).Delete(&model.IssueLabel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&model.Label{}, id).Error
	})
}

func (s *Issue) FindLabel(id uint) (*domain.Label, error) {
	row, err := s.labels().FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	l := row.ToDomain()

	return &l, nil
}

func (s *Issue) ListLabels(repoID uint) ([]domain.Label, error) {
	models, err := s.labels().FindByWhereAndOrder(context.Background(), "repository_id = ?", "name asc", repoID)
	if err != nil {
		return nil, err
	}

	return labelsToDomain(models), nil
}

func (s *Issue) FindLabelByRepoName(repoID uint, name string) (*domain.Label, error) {
	row, err := s.labels().FindByWhere(context.Background(), "repository_id = ? AND name = ?", repoID, name)
	if err != nil {
		return nil, mapDBErr(err)
	}

	l := row.ToDomain()

	return &l, nil
}

func labelsToDomain(models []model.Label) []domain.Label {
	out := make([]domain.Label, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}
	return out
}
