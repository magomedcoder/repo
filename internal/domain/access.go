package domain

import (
	"strings"
	"time"
)

type FolderCap uint32

const (
	CapView FolderCap = 1 << iota
	CapCreateRepos
	CapCreateFolders
	CapManageMembers
	CapManageSettings
	CapDeleteFolder
	CapDeleteRepos
)

const CapAll = CapView | CapCreateRepos | CapCreateFolders | CapManageMembers | CapManageSettings | CapDeleteFolder | CapDeleteRepos

type FolderRole string

const (
	FolderRoleOwner  FolderRole = "owner"
	FolderRoleAdmin  FolderRole = "admin"
	FolderRoleMember FolderRole = "member"
)

func (r FolderRole) Valid() bool {
	switch r {
	case FolderRoleOwner, FolderRoleAdmin, FolderRoleMember:
		return true
	default:
		return false
	}
}

func ParseFolderRole(s string) (FolderRole, bool) {
	r := FolderRole(strings.ToLower(strings.TrimSpace(s)))
	if !r.Valid() {
		return "", false
	}
	return r, true
}

func CapsForRole(role FolderRole) FolderCap {
	switch role {
	case FolderRoleOwner:
		return CapAll
	case FolderRoleAdmin:
		return CapAll &^ CapDeleteFolder
	default:
		return CapView
	}
}

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

type FolderMember struct {
	ID        uint
	FolderID  uint
	UserID    uint
	Role      FolderRole
	Caps      FolderCap
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Team struct {
	ID        uint
	FolderID  uint
	Name      string
	Slug      string
	Caps      FolderCap
	CreatedAt time.Time
	UpdatedAt time.Time
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

type FolderMemberStore interface {
	Create(m *FolderMember) error

	Update(m *FolderMember) error

	Delete(folderID, userID uint) error

	Find(folderID, userID uint) (*FolderMember, error)

	ListByFolder(folderID uint) ([]FolderMember, error)

	ListFolderIDsByUser(userID uint) ([]uint, error)
}

type TeamStore interface {
	Create(t *Team) error

	Update(t *Team) error

	Delete(id uint) error

	FindByID(id uint) (*Team, error)

	FindByFolderSlug(folderID uint, slug string) (*Team, error)

	ListByFolder(folderID uint) ([]Team, error)

	ExistsByFolderSlug(folderID uint, slug string) (bool, error)
}

type TeamMemberStore interface {
	Create(m *TeamMember) error

	Delete(teamID, userID uint) error

	Find(teamID, userID uint) (*TeamMember, error)

	ListByTeam(teamID uint) ([]TeamMember, error)

	ListTeamIDsByFolderUser(folderID, userID uint) ([]uint, error)
}

type TeamRepoGrantStore interface {
	Upsert(g *TeamRepoGrant) error

	Delete(teamID, repoID uint) error

	Find(teamID, repoID uint) (*TeamRepoGrant, error)

	ListByTeam(teamID uint) ([]TeamRepoGrant, error)

	ListByRepoUser(repoID, userID uint) ([]TeamRepoGrant, error)
}

type RepoCollaboratorStore interface {
	Upsert(c *RepoCollaborator) error

	Delete(repoID, userID uint) error

	Find(repoID, userID uint) (*RepoCollaborator, error)

	ListByRepo(repoID uint) ([]RepoCollaborator, error)
}
