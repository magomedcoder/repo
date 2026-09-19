package domain

import "time"

type Issue struct {
	ID           uint
	RepositoryID uint
	Number       int
	Title        string
	Body         string
	State        string
	AuthorID     uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type IssueComment struct {
	ID        uint
	IssueID   uint
	AuthorID  uint
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Label struct {
	ID           uint
	RepositoryID uint
	Name         string
	Color        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type IssueStore interface {
	Create(issue *Issue) error

	Update(issue *Issue) error
	
	Delete(id uint) error

	FindByRepoNumber(repoID uint, number int) (*Issue, error)

	ListByRepo(repoID uint, state string, offset, limit int) ([]Issue, error)

	CountComments(issueID uint) (int, error)

	CreateComment(comment *IssueComment) error

	UpdateComment(comment *IssueComment) error

	DeleteComment(id uint) error

	FindComment(id uint) (*IssueComment, error)
	
	ListComments(issueID uint) ([]IssueComment, error)

	CreateLabel(label *Label) error

	UpdateLabel(label *Label) error

	DeleteLabel(id uint) error

	FindLabel(id uint) (*Label, error)

	ListLabels(repoID uint) ([]Label, error)

	FindLabelByRepoName(repoID uint, name string) (*Label, error)

	SetIssueLabels(issueID uint, labelIDs []uint) error

	ListIssueLabels(issueID uint) ([]Label, error)

	ListLabelsByIssues(issueIDs []uint) (map[uint][]Label, error)
}
