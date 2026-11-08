package domain

import (
	"strings"
	"time"
)

type RepoCollabRole string

const (
	RepoRoleAdmin RepoCollabRole = "admin"
	RepoRoleWrite RepoCollabRole = "write"
	RepoRoleRead  RepoCollabRole = "read"
)

func (r RepoCollabRole) Valid() bool {
	switch r {
	case RepoRoleAdmin, RepoRoleWrite, RepoRoleRead:
		return true
	default:
		return false
	}
}

func ParseRepoCollabRole(s string) (RepoCollabRole, bool) {
	r := RepoCollabRole(strings.ToLower(strings.TrimSpace(s)))
	if !r.Valid() {
		return "", false
	}

	return r, true
}

func (r RepoCollabRole) CanWrite() bool {
	return r == RepoRoleWrite || r == RepoRoleAdmin
}

func (r RepoCollabRole) CanAdmin() bool {
	return r == RepoRoleAdmin
}

type Team struct {
	ID          uint
	OwnerUserID uint
	Name        string
	Slug        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TeamMember struct {
	ID        uint
	TeamID    uint
	UserID    uint
	CreatedAt time.Time
}

type TeamRepoGrant struct {
	ID        uint
	TeamID    uint
	RepoID    uint
	Role      RepoCollabRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RepoCollaborator struct {
	ID        uint
	RepoID    uint
	UserID    uint
	Role      RepoCollabRole
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeamStore interface {
	Create(t *Team) error

	Update(t *Team) error

	Delete(id uint) error

	FindByID(id uint) (*Team, error)

	FindByOwnerSlug(ownerUserID uint, slug string) (*Team, error)

	ListByOwner(ownerUserID uint) ([]Team, error)

	ExistsByOwnerSlug(ownerUserID uint, slug string) (bool, error)
}

type TeamMemberStore interface {
	Create(m *TeamMember) error

	Delete(teamID, userID uint) error

	Find(teamID, userID uint) (*TeamMember, error)

	ListByTeam(teamID uint) ([]TeamMember, error)

	DeleteByTeam(teamID uint) error
}

type TeamRepoGrantStore interface {
	Upsert(g *TeamRepoGrant) error

	Delete(teamID, repoID uint) error

	ListByTeam(teamID uint) ([]TeamRepoGrant, error)

	ListByRepo(repoID uint) ([]TeamRepoGrant, error)

	DeleteByTeam(teamID uint) error

	DeleteByRepoAndTeamOwner(repoID, ownerUserID uint) error

	ListByRepoUser(repoID, userID uint) ([]TeamRepoGrant, error)
}

type RepoCollaboratorStore interface {
	Upsert(c *RepoCollaborator) error

	Delete(repoID, userID uint) error

	Find(repoID, userID uint) (*RepoCollaborator, error)

	ListByRepo(repoID uint) ([]RepoCollaborator, error)

	DeleteByUserID(userID uint) error
}
