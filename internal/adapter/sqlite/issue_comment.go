package sqlite

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type issueCommentModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	IssueID   uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null"`
	Body      string `gorm:"not null"`
}

func (issueCommentModel) TableName() string {
	return "issue_comments"
}

func (s *IssueStore) CountComments(issueID uint) (int, error) {
	var n int64
	if err := s.db.Model(&issueCommentModel{}).Where("issue_id = ?", issueID).Count(&n).Error; err != nil {
		return 0, err
	}

	return int(n), nil
}

func (s *IssueStore) CreateComment(comment *domain.IssueComment) error {
	model := issueCommentFromDomain(comment)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()
	return nil
}

func (s *IssueStore) UpdateComment(comment *domain.IssueComment) error {
	model := issueCommentFromDomain(comment)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()
	return nil
}

func (s *IssueStore) DeleteComment(id uint) error {
	return s.db.Delete(&issueCommentModel{}, id).Error
}

func (s *IssueStore) FindComment(id uint) (*domain.IssueComment, error) {
	var model issueCommentModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, mapDBErr(err)
	}

	c := model.toDomain()
	return &c, nil
}

func (s *IssueStore) ListComments(issueID uint) ([]domain.IssueComment, error) {
	var models []issueCommentModel
	if err := s.db.Where("issue_id = ?", issueID).Order("created_at asc").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.IssueComment, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func issueCommentFromDomain(c *domain.IssueComment) *issueCommentModel {
	return &issueCommentModel{
		ID:        c.ID,
		IssueID:   c.IssueID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (m issueCommentModel) toDomain() domain.IssueComment {
	return domain.IssueComment{
		ID:        m.ID,
		IssueID:   m.IssueID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
