package domain

import "time"

type PullRequest struct {
	ID           uint
	RepositoryID uint
	Number       int
	Title        string
	Body         string
	State        string // open | closed | merged
	AuthorID     uint
	BaseBranch   string
	HeadBranch   string
	MergedAt     *time.Time
	MergedBy     *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type PullRequestComment struct {
	ID        uint
	PullID    uint
	AuthorID  uint
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CompareResult struct {
	BaseSHA        string
	HeadSHA        string
	Commits        []CommitInfo
	Files          []FileDiff
	CanFastForward bool
}

type PullRequestStore interface {
	Create(pr *PullRequest) error

	Update(pr *PullRequest) error

	Delete(id uint) error

	FindByRepoNumber(repoID uint, number int) (*PullRequest, error)

	ListByRepo(repoID uint, state string, offset, limit int) ([]PullRequest, error)

	CountComments(pullID uint) (int, error)

	CreateComment(comment *PullRequestComment) error

	UpdateComment(comment *PullRequestComment) error

	DeleteComment(id uint) error

	FindComment(id uint) (*PullRequestComment, error)

	ListComments(pullID uint) ([]PullRequestComment, error)
}
