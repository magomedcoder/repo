package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type IssueComment struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	IssueID   uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null"`
	Body      string `gorm:"not null"`
}

func (IssueComment) TableName() string {
	return "issue_comments"
}

func IssueCommentFromDomain(c *domain.IssueComment) *IssueComment {
	return &IssueComment{
		ID:        c.ID,
		IssueID:   c.IssueID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (m IssueComment) ToDomain() domain.IssueComment {
	return domain.IssueComment{
		ID:        m.ID,
		IssueID:   m.IssueID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
