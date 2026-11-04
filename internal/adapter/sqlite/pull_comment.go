package sqlite

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type pullCommentModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	PullID    uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null"`
	Body      string `gorm:"not null"`
}

func (pullCommentModel) TableName() string {
	return "pull_request_comments"
}

func (s *PullRequestStore) CountComments(pullID uint) (int, error) {
	var n int64
	if err := s.db.Model(&pullCommentModel{}).Where("pull_id = ?", pullID).Count(&n).Error; err != nil {
		return 0, err
	}

	return int(n), nil
}

func (s *PullRequestStore) CreateComment(comment *domain.PullRequestComment) error {
	model := pullCommentFromDomain(comment)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()

	return nil
}

func (s *PullRequestStore) UpdateComment(comment *domain.PullRequestComment) error {
	model := pullCommentFromDomain(comment)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()

	return nil
}

func (s *PullRequestStore) DeleteComment(id uint) error {
	return s.db.Delete(&pullCommentModel{}, id).Error
}

func (s *PullRequestStore) FindComment(id uint) (*domain.PullRequestComment, error) {
	var model pullCommentModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, mapDBErr(err)
	}

	c := model.toDomain()

	return &c, nil
}

func (s *PullRequestStore) ListComments(pullID uint) ([]domain.PullRequestComment, error) {
	var models []pullCommentModel
	if err := s.db.Where("pull_id = ?", pullID).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.PullRequestComment, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func pullCommentFromDomain(c *domain.PullRequestComment) *pullCommentModel {
	return &pullCommentModel{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		PullID:    c.PullID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
	}
}

func (m pullCommentModel) toDomain() domain.PullRequestComment {
	return domain.PullRequestComment{
		ID:        m.ID,
		PullID:    m.PullID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
