package usecase

import "magomedcoder.ru/repo/internal/domain"

type Authz struct {
	TeamRepos domain.TeamRepoGrantStore
	Collabs   domain.RepoCollaboratorStore
}

func NewAuthz(teamRepos domain.TeamRepoGrantStore, collabs domain.RepoCollaboratorStore) *Authz {
	return &Authz{
		TeamRepos: teamRepos,
		Collabs:   collabs,
	}
}

func (a *Authz) CanReadRepo(repo *domain.Repository, viewerID uint) error {
	if repo == nil {
		return ErrRepoNotFound
	}

	if !repo.IsPrivate {
		return nil
	}

	if a.CanReadPrivateRepo(repo, viewerID) {
		return nil
	}

	return ErrRepoNotFound
}

func (a *Authz) CanReadPrivateRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 || repo == nil {
		return false
	}

	if isPersonalOwner(repo, viewerID) {
		return true
	}

	if a == nil {
		return false
	}

	if a.hasCollab(repo.ID, viewerID) {
		return true
	}

	return a.bestTeamRepoRole(repo.ID, viewerID) != ""
}

func (a *Authz) CanWriteRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 || repo == nil {
		return false
	}

	if isPersonalOwner(repo, viewerID) {
		return true
	}

	if a == nil {
		return false
	}

	if role, ok := a.collabRole(repo.ID, viewerID); ok && role.CanWrite() {
		return true
	}

	role := a.bestTeamRepoRole(repo.ID, viewerID)
	return role.CanWrite()
}

func (a *Authz) CanAdminRepo(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 || repo == nil {
		return false
	}

	if isPersonalOwner(repo, viewerID) {
		return true
	}

	if a == nil {
		return false
	}

	if role, ok := a.collabRole(repo.ID, viewerID); ok && role.CanAdmin() {
		return true
	}

	return a.bestTeamRepoRole(repo.ID, viewerID).CanAdmin()
}

func (a *Authz) CanWriteGit(repo *domain.Repository, viewerID uint) bool {
	return a.CanWriteRepo(repo, viewerID)
}

func (a *Authz) CanContribute(repo *domain.Repository, viewerID uint) bool {
	if viewerID == 0 || repo == nil {
		return false
	}

	if !repo.IsPrivate {
		return true
	}

	return a.CanWriteRepo(repo, viewerID)
}

func isPersonalOwner(repo *domain.Repository, viewerID uint) bool {
	return viewerID != 0 && viewerID == repo.OwnerID
}

func (a *Authz) hasCollab(repoID, userID uint) bool {
	_, ok := a.collabRole(repoID, userID)
	return ok
}

func (a *Authz) collabRole(repoID, userID uint) (domain.RepoCollabRole, bool) {
	if a == nil || a.Collabs == nil {
		return "", false
	}

	c, err := a.Collabs.Find(repoID, userID)
	if err != nil || c == nil {
		return "", false
	}

	return c.Role, true
}

func (a *Authz) bestTeamRepoRole(repoID, userID uint) domain.RepoCollabRole {
	if a == nil || a.TeamRepos == nil {
		return ""
	}

	grants, err := a.TeamRepos.ListByRepoUser(repoID, userID)
	if err != nil {
		return ""
	}

	var best domain.RepoCollabRole
	for _, g := range grants {
		best = maxRepoRole(best, g.Role)
	}

	return best
}

func maxRepoRole(a, b domain.RepoCollabRole) domain.RepoCollabRole {
	rank := func(r domain.RepoCollabRole) int {
		switch r {
		case domain.RepoRoleAdmin:
			return 3
		case domain.RepoRoleWrite:
			return 2
		case domain.RepoRoleRead:
			return 1
		default:
			return 0
		}
	}
	if rank(b) > rank(a) {
		return b
	}
	return a
}
