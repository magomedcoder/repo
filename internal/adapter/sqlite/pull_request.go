package sqlite

import (
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type pullRequestModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_pull_number"`
	Number       int    `gorm:"not null;uniqueIndex:idx_pull_number"`
	Title        string `gorm:"size:200;not null"`
	Body         string
	State        string `gorm:"size:16;not null"`
	AuthorID     uint   `gorm:"not null"`
	BaseBranch   string `gorm:"size:255;not null"`
	HeadBranch   string `gorm:"size:255;not null"`
	MergedAt     *time.Time
	MergedBy     *uint
}

func (pullRequestModel) TableName() string {
	return "pull_requests"
}

type PullRequestStore struct {
	db *gorm.DB
}

func NewPullRequestStore(db *gorm.DB) *PullRequestStore {
	return &PullRequestStore{db: db}
}

func (s *PullRequestStore) Create(pr *domain.PullRequest) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
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
		return nil, mapDBErr(err)
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
