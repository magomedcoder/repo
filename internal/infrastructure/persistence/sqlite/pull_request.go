package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type pullRequestModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_pull_number"`
	Number       int    `gorm:"not null;uniqueIndex:idx_pull_number"`
	Title        string `gorm:"size:200;not null"`
	Body         string
	State        string `gorm:"size:16;not null;index"`
	AuthorID     uint   `gorm:"not null;index"`
	BaseBranch   string `gorm:"size:255;not null"`
	HeadBranch   string `gorm:"size:255;not null"`
	MergedAt     *time.Time
	MergedBy     *uint
}

func (pullRequestModel) TableName() string {
	return "pull_requests"
}

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

type PullRequestStore struct {
	db *gorm.DB
}

func NewPullRequestStore(db *gorm.DB) *PullRequestStore {
	return &PullRequestStore{db: db}
}

func (s *PullRequestStore) Create(pr *domain.PullRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var maxNum int
		if err := tx.Model(&pullRequestModel{}).
			Where("repository_id = ?", pr.RepositoryID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNum).Error; err != nil {
			return err
		}

		pr.Number = maxNum + 1
		if pr.State == "" {
			pr.State = "open"
		}

		model := pullFromDomain(pr)
		if err := tx.Create(model).Error; err != nil {
			return err
		}

		*pr = model.toDomain()
		return nil
	})
}

func (s *PullRequestStore) Update(pr *domain.PullRequest) error {
	model := pullFromDomain(pr)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*pr = model.toDomain()
	return nil
}

func (s *PullRequestStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pull_id = ?", id).Delete(&pullCommentModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&pullRequestModel{}, id).Error
	})
}

func (s *PullRequestStore) FindByRepoNumber(repoID uint, number int) (*domain.PullRequest, error) {
	var model pullRequestModel
	if err := s.db.Where("repository_id = ? AND number = ?", repoID, number).First(&model).Error; err != nil {
		return nil, err
	}

	pr := model.toDomain()
	return &pr, nil
}

func (s *PullRequestStore) ListByRepo(repoID uint, state string, offset, limit int) ([]domain.PullRequest, error) {
	q := s.db.Where("repository_id = ?", repoID)
	switch state {
	case "open", "closed", "merged":
		q = q.Where("state = ?", state)
	}

	if limit <= 0 {
		limit = 30
	}

	var models []pullRequestModel
	if err := q.Order("number desc").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.PullRequest, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
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
		return nil, err
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

func pullFromDomain(pr *domain.PullRequest) *pullRequestModel {
	return &pullRequestModel{
		ID:           pr.ID,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
		RepositoryID: pr.RepositoryID,
		Number:       pr.Number,
		Title:        pr.Title,
		Body:         pr.Body,
		State:        pr.State,
		AuthorID:     pr.AuthorID,
		BaseBranch:   pr.BaseBranch,
		HeadBranch:   pr.HeadBranch,
		MergedAt:     pr.MergedAt,
		MergedBy:     pr.MergedBy,
	}
}

func (m pullRequestModel) toDomain() domain.PullRequest {
	return domain.PullRequest{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Number:       m.Number,
		Title:        m.Title,
		Body:         m.Body,
		State:        m.State,
		AuthorID:     m.AuthorID,
		BaseBranch:   m.BaseBranch,
		HeadBranch:   m.HeadBranch,
		MergedAt:     m.MergedAt,
		MergedBy:     m.MergedBy,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
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
