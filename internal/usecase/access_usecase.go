package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrMemberNotFound       = NewError(KindNotFound, "member_not_found")
	ErrMemberExists         = NewError(KindConflict, "member_already_exists")
	ErrInvalidFolderRole    = NewError(KindValidation, "invalid_folder_role")
	ErrInvalidCollabRole    = NewError(KindValidation, "invalid_collaborator_role")
	ErrCannotRemoveOwner    = NewError(KindForbidden, "cannot_remove_owner")
	ErrLastOwner            = NewError(KindForbidden, "last_folder_owner")
	ErrTeamNotFound         = NewError(KindNotFound, "team_not_found")
	ErrTeamExists           = NewError(KindConflict, "team_already_exists")
	ErrTeamNameRequired     = NewError(KindValidation, "team_name_required")
	ErrInvalidTeamName      = NewError(KindValidation, "invalid_team_name")
	ErrTeamMemberExists     = NewError(KindConflict, "team_member_already_exists")
	ErrTeamMemberNotFound   = NewError(KindNotFound, "team_member_not_found")
	ErrCollaboratorNotFound = NewError(KindNotFound, "collaborator_not_found")
	ErrCannotCollabSelf     = NewError(KindValidation, "cannot_collaborator_self")
)

type MemberItem struct {
	UserID    uint            `json:"user_id"`
	Username  string          `json:"username"`
	Role      string          `json:"role"`
	Caps      map[string]bool `json:"caps"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type TeamItem struct {
	Name      string          `json:"name"`
	Slug      string          `json:"slug"`
	Caps      map[string]bool `json:"caps"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type TeamMemberItem struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type TeamRepoGrantItem struct {
	RepoID   uint   `json:"repo_id"`
	RepoName string `json:"repo_name"`
	RepoPath string `json:"repo_path"`
	Role     string `json:"role"`
}

type CollaboratorItem struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccessUseCase struct {
	folders     domain.FolderStore
	repos       domain.RepositoryStore
	users       domain.UserStore
	members     domain.FolderMemberStore
	teams       domain.TeamStore
	teamMembers domain.TeamMemberStore
	teamRepos   domain.TeamRepoGrantStore
	collabs     domain.RepoCollaboratorStore
	authz       *Authz
}

func NewAccessUseCase(
	folders domain.FolderStore,
	repos domain.RepositoryStore,
	users domain.UserStore,
	members domain.FolderMemberStore,
	teams domain.TeamStore,
	teamMembers domain.TeamMemberStore,
	teamRepos domain.TeamRepoGrantStore,
	collabs domain.RepoCollaboratorStore,
	authz *Authz,
) *AccessUseCase {
	return &AccessUseCase{
		folders:     folders,
		repos:       repos,
		users:       users,
		members:     members,
		teams:       teams,
		teamMembers: teamMembers,
		teamRepos:   teamRepos,
		collabs:     collabs,
		authz:       authz,
	}
}

func (uc *AccessUseCase) resolveFolder(path string) (*domain.Folder, error) {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil, ErrFolderNotFound
	}

	folder, err := uc.folders.FindByPath(path)
	if err != nil {
		return nil, ErrFolderNotFound
	}

	return folder, nil
}

func (uc *AccessUseCase) ListMembers(ctx context.Context, viewerID uint, folderPath string) ([]MemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapView); err != nil {
		return nil, ErrFolderNotFound
	}

	rows, err := uc.members.ListByFolder(folder.ID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}

	return uc.mapMembers(rows)
}

type UpsertMemberInput struct {
	ViewerID   uint
	FolderPath string
	Username   string
	Role       string
	Caps       map[string]bool
}

func (uc *AccessUseCase) AddMember(ctx context.Context, in UpsertMemberInput) (*MemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(in.FolderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, in.ViewerID, domain.CapManageMembers); err != nil {
		return nil, err
	}

	role, caps, err := uc.normalizeGrantedMemberCaps(folder.ID, in.ViewerID, in.Role, in.Caps)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(in.Username)))
	if err != nil {
		return nil, ErrUserNotFound
	}

	if _, err := uc.members.Find(folder.ID, user.ID); err == nil {
		return nil, ErrMemberExists
	}

	m := &domain.FolderMember{
		FolderID: folder.ID,
		UserID:   user.ID,
		Role:     role,
		Caps:     caps,
	}
	if err := uc.members.Create(m); err != nil {
		return nil, fmt.Errorf("add member: %w", err)
	}

	return &MemberItem{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      string(m.Role),
		Caps:      CapsJSON(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (uc *AccessUseCase) UpdateMember(ctx context.Context, in UpsertMemberInput) (*MemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(in.FolderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, in.ViewerID, domain.CapManageMembers); err != nil {
		return nil, err
	}

	role, caps, err := uc.normalizeGrantedMemberCaps(folder.ID, in.ViewerID, in.Role, in.Caps)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(in.Username)))
	if err != nil {
		return nil, ErrUserNotFound
	}

	m, err := uc.members.Find(folder.ID, user.ID)
	if err != nil {
		return nil, ErrMemberNotFound
	}

	if m.Role == domain.FolderRoleOwner && role != domain.FolderRoleOwner {
		if err := uc.ensureOtherOwner(folder.ID, user.ID); err != nil {
			return nil, err
		}
	}

	m.Role = role
	m.Caps = caps
	if err := uc.members.Update(m); err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}

	return &MemberItem{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      string(m.Role),
		Caps:      CapsJSON(m.Caps),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}, nil
}

func (uc *AccessUseCase) RemoveMember(ctx context.Context, viewerID uint, folderPath, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapManageMembers); err != nil {
		return err
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return ErrUserNotFound
	}

	m, err := uc.members.Find(folder.ID, user.ID)
	if err != nil {
		return ErrMemberNotFound
	}

	if m.Role == domain.FolderRoleOwner {
		if err := uc.ensureOtherOwner(folder.ID, user.ID); err != nil {
			return err
		}
	}

	if err := uc.members.Delete(folder.ID, user.ID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}

	return nil
}

func (uc *AccessUseCase) ensureOtherOwner(folderID, exceptUserID uint) error {
	rows, err := uc.members.ListByFolder(folderID)
	if err != nil {
		return err
	}

	for _, m := range rows {
		if m.UserID != exceptUserID && m.Role == domain.FolderRoleOwner {
			return nil
		}
	}

	return ErrLastOwner
}

func (uc *AccessUseCase) normalizeGrantedMemberCaps(folderID, viewerID uint, roleRaw string, capsMap map[string]bool) (domain.FolderRole, domain.FolderCap, error) {
	role, caps, err := normalizeMemberRoleCaps(roleRaw, capsMap)
	if err != nil {
		return "", 0, err
	}

	actorCaps, _, err := uc.authz.FolderCaps(folderID, viewerID)
	if err != nil {
		return "", 0, err
	}

	if role == domain.FolderRoleOwner || caps&domain.CapDeleteFolder != 0 {
		if actorCaps&domain.CapDeleteFolder == 0 {
			return "", 0, ErrFolderForbidden
		}
	}

	caps &= actorCaps
	if caps&domain.CapView == 0 {
		caps |= domain.CapView
	}

	if role == domain.FolderRoleOwner {
		caps = domain.CapsForRole(domain.FolderRoleOwner)
	}

	return role, caps, nil
}

func normalizeMemberRoleCaps(roleRaw string, capsMap map[string]bool) (domain.FolderRole, domain.FolderCap, error) {
	role, ok := domain.ParseFolderRole(roleRaw)
	if !ok {
		if strings.TrimSpace(roleRaw) == "" {
			role = domain.FolderRoleMember
		} else {
			return "", 0, ErrInvalidFolderRole
		}
	}

	base := domain.CapsForRole(role)
	caps := base
	if capsMap != nil {
		caps = CapsFromJSON(capsMap, base)
		if role == domain.FolderRoleOwner {
			caps = domain.CapsForRole(domain.FolderRoleOwner)
		}

		if role == domain.FolderRoleAdmin {
			caps &^= domain.CapDeleteFolder
			caps |= domain.CapView
		}

		if role == domain.FolderRoleMember {
			caps &^= domain.CapDeleteFolder
			if caps == 0 {
				caps = domain.CapView
			}
		}

		if caps&domain.CapView == 0 {
			caps |= domain.CapView
		}
	}

	return role, caps, nil
}

func (uc *AccessUseCase) mapMembers(rows []domain.FolderMember) ([]MemberItem, error) {
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}

	users, err := uc.users.FindByIDs(context.Background(), ids)
	if err != nil {
		return nil, err
	}

	out := make([]MemberItem, 0, len(rows))
	for _, r := range rows {
		username := ""
		if u := users[r.UserID]; u != nil {
			username = u.Username
		}

		out = append(out, MemberItem{
			UserID:    r.UserID,
			Username:  username,
			Role:      string(r.Role),
			Caps:      CapsJSON(r.Caps),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}

	return out, nil
}

func (uc *AccessUseCase) ListTeams(ctx context.Context, viewerID uint, folderPath string) ([]TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapView); err != nil {
		return nil, ErrFolderNotFound
	}

	rows, err := uc.teams.ListByFolder(folder.ID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	out := make([]TeamItem, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTeamItem(t))
	}

	return out, nil
}

type UpsertTeamInput struct {
	ViewerID   uint
	FolderPath string
	Name       string
	Caps       map[string]bool
}

func (uc *AccessUseCase) CreateTeam(ctx context.Context, in UpsertTeamInput) (*TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(in.FolderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, in.ViewerID, domain.CapManageMembers); err != nil {
		return nil, err
	}

	name, slug, err := normalizeFolderName(in.Name)
	if err != nil {
		if err == ErrFolderNameRequired {
			return nil, ErrTeamNameRequired
		}

		return nil, ErrInvalidTeamName
	}

	exists, err := uc.teams.ExistsByFolderSlug(folder.ID, slug)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrTeamExists
	}

	caps := domain.CapView
	if in.Caps != nil {
		caps = CapsFromJSON(in.Caps, domain.CapView)
	}

	if caps&domain.CapView == 0 {
		caps |= domain.CapView
	}

	caps &^= domain.CapDeleteFolder

	t := &domain.Team{
		FolderID: folder.ID,
		Name:     name,
		Slug:     slug,
		Caps:     caps,
	}
	if err := uc.teams.Create(t); err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	item := toTeamItem(*t)

	return &item, nil
}

func (uc *AccessUseCase) UpdateTeam(ctx context.Context, folderPath, slug string, in UpsertTeamInput) (*TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, in.ViewerID, domain.CapManageMembers); err != nil {
		return nil, err
	}

	t, err := uc.teams.FindByFolderSlug(folder.ID, slug)
	if err != nil {
		return nil, ErrTeamNotFound
	}

	if strings.TrimSpace(in.Name) != "" {
		name, newSlug, nerr := normalizeFolderName(in.Name)
		if nerr != nil {
			return nil, ErrInvalidTeamName
		}

		if newSlug != t.Slug {
			exists, eerr := uc.teams.ExistsByFolderSlug(folder.ID, newSlug)
			if eerr != nil {
				return nil, eerr
			}

			if exists {
				return nil, ErrTeamExists
			}
			t.Slug = newSlug
		}

		t.Name = name
	}

	if in.Caps != nil {
		caps := CapsFromJSON(in.Caps, t.Caps)
		if caps&domain.CapView == 0 {
			caps |= domain.CapView
		}

		caps &^= domain.CapDeleteFolder
		t.Caps = caps
	}
	if err := uc.teams.Update(t); err != nil {
		return nil, fmt.Errorf("update team: %w", err)
	}
	item := toTeamItem(*t)

	return &item, nil
}

func (uc *AccessUseCase) DeleteTeam(ctx context.Context, viewerID uint, folderPath, slug string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapManageMembers); err != nil {
		return err
	}

	t, err := uc.teams.FindByFolderSlug(folder.ID, slug)
	if err != nil {
		return ErrTeamNotFound
	}

	return uc.teams.Delete(t.ID)
}

func (uc *AccessUseCase) ListTeamMembers(ctx context.Context, viewerID uint, folderPath, slug string) ([]TeamMemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, t, err := uc.resolveTeam(viewerID, folderPath, slug)
	if err != nil {
		return nil, err
	}

	_ = folder
	rows, err := uc.teamMembers.ListByTeam(t.ID)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}

	users, err := uc.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]TeamMemberItem, 0, len(rows))
	for _, r := range rows {
		username := ""
		if u := users[r.UserID]; u != nil {
			username = u.Username
		}

		out = append(out, TeamMemberItem{
			UserID:    r.UserID,
			Username:  username,
			CreatedAt: r.CreatedAt,
		})
	}

	return out, nil
}

func (uc *AccessUseCase) AddTeamMember(ctx context.Context, viewerID uint, folderPath, slug, username string) (*TeamMemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, t, err := uc.resolveTeamManage(viewerID, folderPath, slug)
	if err != nil {
		return nil, err
	}

	_ = folder

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, ErrUserNotFound
	}

	if _, err := uc.teamMembers.Find(t.ID, user.ID); err == nil {
		return nil, ErrTeamMemberExists
	}

	m := &domain.TeamMember{TeamID: t.ID, UserID: user.ID}
	if err := uc.teamMembers.Create(m); err != nil {
		return nil, fmt.Errorf("add team member: %w", err)
	}

	return &TeamMemberItem{
		UserID:    user.ID,
		Username:  user.Username,
		CreatedAt: m.CreatedAt,
	}, nil
}

func (uc *AccessUseCase) RemoveTeamMember(ctx context.Context, viewerID uint, folderPath, slug, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	_, t, err := uc.resolveTeamManage(viewerID, folderPath, slug)
	if err != nil {
		return err
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return ErrUserNotFound
	}

	if _, err := uc.teamMembers.Find(t.ID, user.ID); err != nil {
		return ErrTeamMemberNotFound
	}

	return uc.teamMembers.Delete(t.ID, user.ID)
}

func (uc *AccessUseCase) ListTeamRepos(ctx context.Context, viewerID uint, folderPath, slug string) ([]TeamRepoGrantItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, t, err := uc.resolveTeam(viewerID, folderPath, slug)
	if err != nil {
		return nil, err
	}

	rows, err := uc.teamRepos.ListByTeam(t.ID)
	if err != nil {
		return nil, err
	}

	repos, err := uc.repos.ListByFolderID(ctx, folder.ID)
	if err != nil {
		return nil, err
	}

	byID := map[uint]domain.Repository{}
	for _, r := range repos {
		byID[r.ID] = r
	}

	out := make([]TeamRepoGrantItem, 0, len(rows))
	for _, g := range rows {
		repoPath := ""
		name := ""
		if r, ok := byID[g.RepoID]; ok {
			name = r.Name
			repoPath = strings.TrimSuffix(r.RepoPath, ".git")
		}

		out = append(out, TeamRepoGrantItem{
			RepoID:   g.RepoID,
			RepoName: name,
			RepoPath: repoPath,
			Role:     string(g.Role),
		})
	}

	return out, nil
}

type SetTeamRepoInput struct {
	ViewerID   uint
	FolderPath string
	TeamSlug   string
	RepoName   string
	Role       string
}

func (uc *AccessUseCase) SetTeamRepo(ctx context.Context, in SetTeamRepoInput) (*TeamRepoGrantItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	folder, t, err := uc.resolveTeamManage(in.ViewerID, in.FolderPath, in.TeamSlug)
	if err != nil {
		return nil, err
	}

	role, ok := domain.ParseRepoCollabRole(in.Role)
	if !ok {
		return nil, ErrInvalidCollabRole
	}

	name, err := normalizeRepoName(in.RepoName)
	if err != nil {
		return nil, err
	}

	logical := folder.Path + "/" + name + ".git"
	repo, err := uc.repos.FindByRepoPath(ctx, logical)
	if err != nil || repo.FolderID == nil || *repo.FolderID != folder.ID {
		return nil, ErrRepoNotFound
	}

	g := &domain.TeamRepoGrant{TeamID: t.ID, RepoID: repo.ID, Role: role}
	if err := uc.teamRepos.Upsert(g); err != nil {
		return nil, fmt.Errorf("set team repo grant: %w", err)
	}

	return &TeamRepoGrantItem{
		RepoID:   repo.ID,
		RepoName: repo.Name,
		RepoPath: strings.TrimSuffix(repo.RepoPath, ".git"),
		Role:     string(role),
	}, nil
}

func (uc *AccessUseCase) RemoveTeamRepo(ctx context.Context, viewerID uint, folderPath, slug, repoName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	folder, t, err := uc.resolveTeamManage(viewerID, folderPath, slug)
	if err != nil {
		return err
	}

	name, err := normalizeRepoName(repoName)
	if err != nil {
		return err
	}

	logical := folder.Path + "/" + name + ".git"
	repo, err := uc.repos.FindByRepoPath(ctx, logical)
	if err != nil {
		return ErrRepoNotFound
	}

	return uc.teamRepos.Delete(t.ID, repo.ID)
}

func (uc *AccessUseCase) resolveTeam(viewerID uint, folderPath, slug string) (*domain.Folder, *domain.Team, error) {
	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return nil, nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapView); err != nil {
		return nil, nil, ErrFolderNotFound
	}

	t, err := uc.teams.FindByFolderSlug(folder.ID, slug)
	if err != nil {
		return nil, nil, ErrTeamNotFound
	}

	return folder, t, nil
}

func (uc *AccessUseCase) resolveTeamManage(viewerID uint, folderPath, slug string) (*domain.Folder, *domain.Team, error) {
	folder, err := uc.resolveFolder(folderPath)
	if err != nil {
		return nil, nil, err
	}

	if err := uc.authz.RequireFolderCap(folder.ID, viewerID, domain.CapManageMembers); err != nil {
		return nil, nil, err
	}

	t, err := uc.teams.FindByFolderSlug(folder.ID, slug)
	if err != nil {
		return nil, nil, ErrTeamNotFound
	}

	return folder, t, nil
}

func toTeamItem(t domain.Team) TeamItem {
	return TeamItem{
		Name:      t.Name,
		Slug:      t.Slug,
		Caps:      CapsJSON(t.Caps),
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func (uc *AccessUseCase) ListCollaborators(ctx context.Context, in ResolveRepositoryInput) ([]CollaboratorItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := ResolveReadable(uc.repos, uc.authz, in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	rows, err := uc.collabs.ListByRepo(repo.ID)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}

	users, err := uc.users.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]CollaboratorItem, 0, len(rows))
	for _, r := range rows {
		username := ""
		if u := users[r.UserID]; u != nil {
			username = u.Username
		}

		out = append(out, CollaboratorItem{
			UserID:    r.UserID,
			Username:  username,
			Role:      string(r.Role),
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}

	return out, nil
}

type UpsertCollaboratorInput struct {
	ResolveRepositoryInput
	Username string
	Role     string
}

func (uc *AccessUseCase) AddCollaborator(ctx context.Context, in UpsertCollaboratorInput) (*CollaboratorItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := ResolveReadable(uc.repos, uc.authz, in.ResolveRepositoryInput)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	role, ok := domain.ParseRepoCollabRole(in.Role)
	if !ok {
		return nil, ErrInvalidCollabRole
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(in.Username)))
	if err != nil {
		return nil, ErrUserNotFound
	}

	if isPersonalOwner(repo, user.ID) {
		return nil, ErrCannotCollabSelf
	}

	c := &domain.RepoCollaborator{
		RepoID: repo.ID,
		UserID: user.ID,
		Role:   role,
	}
	if err := uc.collabs.Upsert(c); err != nil {
		return nil, fmt.Errorf("add collaborator: %w", err)
	}

	return &CollaboratorItem{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      string(c.Role),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func (uc *AccessUseCase) RemoveCollaborator(ctx context.Context, in ResolveRepositoryInput, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, err := ResolveReadable(uc.repos, uc.authz, in)
	if err != nil {
		return err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return ErrRepoNotFound
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return ErrUserNotFound
	}

	if _, err := uc.collabs.Find(repo.ID, user.ID); err != nil {
		return ErrCollaboratorNotFound
	}

	return uc.collabs.Delete(repo.ID, user.ID)
}
