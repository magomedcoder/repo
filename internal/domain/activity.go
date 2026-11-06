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
	OwnerKind string
	OwnerID   uint
	RepoID    *uint
	Type      string
	Summary   string
	Ref       string
	CreatedAt time.Time
}

type ActivityStore interface {
	Create(ev *ActivityEvent) error

	ListByOwner(ownerKind string, ownerID uint, limit int) ([]ActivityEvent, error)

	ListPublicByOwner(ownerKind string, ownerID uint, limit int) ([]ActivityEvent, error)

	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
