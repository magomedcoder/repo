package model

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

type Issue struct {
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

func (Issue) TableName() string {
	return "issues"
}

func IssueFromDomain(issue *domain.Issue) *Issue {
	return &Issue{
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

func (m Issue) ToDomain() domain.Issue {
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
