package domain

import (
	"context"
	"time"
)

const (
	ActivityPush       = "push"
	ActivityIssueOpen  = "issue_opened"
	ActivityIssueClose = "issue_closed"
	ActivityPROpen     = "pull_opened"
	ActivityPRMerge    = "pull_merged"
	ActivityPRClose    = "pull_closed"
	ActivityRepoCreate = "repo_created"
)

type ActivityEvent struct {
	ID        uint
	ActorID   uint
	OwnerID   uint
	RepoID    *uint
	Type      string
	Summary   string
	Ref       string
	CreatedAt time.Time
}

type ActivityStore interface {
	Create(ev *ActivityEvent) error

	ListByOwner(ownerID uint, limit int) ([]ActivityEvent, error)

	ListPublicByOwner(ownerID uint, limit int) ([]ActivityEvent, error)

	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)

	DeleteByRepo(ctx context.Context, repoID uint) error
}
