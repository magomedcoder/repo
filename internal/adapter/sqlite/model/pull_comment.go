package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type PullComment struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	PullID    uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null"`
	Body      string `gorm:"not null"`
}

func (PullComment) TableName() string {
	return "pull_request_comments"
}

func PullCommentFromDomain(c *domain.PullRequestComment) *PullComment {
	return &PullComment{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		PullID:    c.PullID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
	}
}

func (m PullComment) ToDomain() domain.PullRequestComment {
	return domain.PullRequestComment{
		ID:        m.ID,
		PullID:    m.PullID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
