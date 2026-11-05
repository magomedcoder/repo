package usecase

import "magomedcoder.ru/repo/internal/domain"

type Authz struct {
	Members     domain.FolderMemberStore
	Teams       domain.TeamStore
	TeamMembers domain.TeamMemberStore
	TeamRepos   domain.TeamRepoGrantStore
	Collabs     domain.RepoCollaboratorStore
}

func NewAuthz(members domain.FolderMemberStore, teams domain.TeamStore, teamMembers domain.TeamMemberStore, teamRepos domain.TeamRepoGrantStore, collabs domain.RepoCollaboratorStore) *Authz {
	return &Authz{
		Members:     members,
		Teams:       teams,
		TeamMembers: teamMembers,
		TeamRepos:   teamRepos,
		Collabs:     collabs,
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

func (a *Authz) FolderCaps(folderID, userID uint) (domain.FolderCap, domain.FolderRole, error) {
	if a == nil || a.Members == nil || userID == 0 || folderID == 0 {
		return 0, "", nil
	}

	var caps domain.FolderCap
	var role domain.FolderRole

	if m, err := a.Members.Find(folderID, userID); err == nil && m != nil {
		caps |= m.Caps
		role = m.Role
	}

	if a.TeamMembers != nil && a.Teams != nil {
		teamIDs, err := a.TeamMembers.ListTeamIDsByFolderUser(folderID, userID)
		if err != nil {
			return 0, "", err
		}
		for _, tid := range teamIDs {
			t, err := a.Teams.FindByID(tid)
			if err != nil || t == nil {
				continue
			}
			caps |= t.Caps
		}
	}

	return caps, role, nil
}

func (a *Authz) HasFolderCap(folderID, userID uint, need domain.FolderCap) bool {
	caps, _, err := a.FolderCaps(folderID, userID)
	if err != nil {
		return false
	}
	return caps&need == need
}

func (a *Authz) RequireFolderCap(folderID, userID uint, need domain.FolderCap) error {
	if userID == 0 {
		return ErrUnauthorized
	}

	if !a.HasFolderCap(folderID, userID, need) {
		return ErrFolderForbidden
	}

	return nil
}

func isPersonalOwner(repo *domain.Repository, viewerID uint) bool {
	kind := repo.OwnerKind
	if kind == "" {
		kind = domain.OwnerKindUser
	}
	return kind == domain.OwnerKindUser && viewerID == repo.OwnerID
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

func CapsJSON(c domain.FolderCap) map[string]bool {
	return map[string]bool{
		"view":            c&domain.CapView != 0,
		"create_repos":    c&domain.CapCreateRepos != 0,
		"manage_members":  c&domain.CapManageMembers != 0,
		"manage_settings": c&domain.CapManageSettings != 0,
		"delete_folder":   c&domain.CapDeleteFolder != 0,
		"delete_repos":    c&domain.CapDeleteRepos != 0,
	}
}

func CapsFromJSON(m map[string]bool, base domain.FolderCap) domain.FolderCap {
	if m == nil {
		return base
	}

	var c domain.FolderCap
	set := func(key string, bit domain.FolderCap) {
		if v, ok := m[key]; ok {
			if v {
				c |= bit
			}
			return
		}
		c |= base & bit
	}
	set("view", domain.CapView)
	set("create_repos", domain.CapCreateRepos)
	set("manage_members", domain.CapManageMembers)
	set("manage_settings", domain.CapManageSettings)
	set("delete_folder", domain.CapDeleteFolder)
	set("delete_repos", domain.CapDeleteRepos)

	return c
}
