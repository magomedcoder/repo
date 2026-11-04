package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type issueModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_issue_number"`
	Number       int    `gorm:"not null;uniqueIndex:idx_issue_number"`
	Title        string `gorm:"size:200;not null"`
	Body         string
	State        string `gorm:"size:16;not null"`
	AuthorID     uint   `gorm:"not null"`
}

func (issueModel) TableName() string {
	return "issues"
}

type IssueStore struct {
	db *gorm.DB
}

func NewIssueStore(db *gorm.DB) *IssueStore {
	return &IssueStore{db: db}
}

func (s *IssueStore) Create(issue *domain.Issue) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
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

		model := issueFromDomain(issue)
		if err := tx.Create(model).Error; err != nil {
			return err
		}

		*issue = model.toDomain()
		return nil
	})
}

func (s *IssueStore) Update(issue *domain.Issue) error {
	model := issueFromDomain(issue)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*issue = model.toDomain()
	return nil
}

func (s *IssueStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", id).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		if err := tx.Where("issue_id = ?", id).Delete(&issueCommentModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&issueModel{}, id).Error
	})
}

func (s *IssueStore) FindByRepoNumber(repoID uint, number int) (*domain.Issue, error) {
	var model issueModel
	if err := s.db.Where("repository_id = ? AND number = ?", repoID, number).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}

	issue := model.toDomain()
	return &issue, nil
}

func (s *IssueStore) ListByRepo(repoID uint, state string, offset, limit int) ([]domain.Issue, error) {
	q := s.db.Where("repository_id = ?", repoID)
	if state == "open" || state == "closed" {
		q = q.Where("state = ?", state)
	}

	if limit <= 0 {
		limit = 30
	}

	var models []issueModel
	if err := q.Order("number desc").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Issue, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func issueFromDomain(issue *domain.Issue) *issueModel {
	return &issueModel{
		ID:           issue.ID,
		RepositoryID: issue.RepositoryID,
		Number:       issue.Number,
		Title:        issue.Title,
		Body:         issue.Body,
		State:        issue.State,
		AuthorID:     issue.AuthorID,
		CreatedAt:    issue.CreatedAt,
		UpdatedAt:    issue.UpdatedAt,
	}
}

func (m issueModel) toDomain() domain.Issue {
	return domain.Issue{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Number:       m.Number,
		Title:        m.Title,
		Body:         m.Body,
		State:        m.State,
		AuthorID:     m.AuthorID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
