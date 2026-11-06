package model

import (
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type PullRequest struct {
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

func (PullRequest) TableName() string {
	return "pull_requests"
}

func PullRequestFromDomain(pr *domain.PullRequest) *PullRequest {
	return &PullRequest{
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

func (m PullRequest) ToDomain() domain.PullRequest {
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
