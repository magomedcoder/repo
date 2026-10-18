package domain

import "time"

const (
	ActivityPush       = "push"
	ActivityIssueOpen  = "issue_opened"
	ActivityIssueClose = "issue_closed"
	ActivityPROpen     = "pull_opened"
	ActivityPRMerge    = "pull_merged"
	ActivityPRClose    = "pull_closed"
	ActivityMemberAdd  = "member_added"
	ActivityOrgCreate  = "org_created"
	ActivityRepoCreate = "repo_created"
	ActivityTransfer   = "repo_transferred"
)

type ActivityEvent struct {
	ID        uint
	ActorID   uint
	OwnerKind string
	OwnerID   uint
	RepoID    *uint
	OrgID     *uint
	Type      string
	Summary   string
	Ref       string
	CreatedAt time.Time
}

type ActivityStore interface {
	Create(ev *ActivityEvent) error

	ListByOwner(ownerKind string, ownerID uint, limit int) ([]ActivityEvent, error)

	ListForUser(userID uint, limit int) ([]ActivityEvent, error)
}
