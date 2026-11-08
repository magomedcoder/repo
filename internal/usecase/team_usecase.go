package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"magomedcoder.ru/repo/internal/domain"
)

const (
	maxTeamNameLen = 64
	minTeamNameLen = 1
)

var (
	ErrTeamNotFound       = NewError(KindNotFound, "team_not_found")
	ErrTeamExists         = NewError(KindConflict, "team_already_exists")
	ErrTeamNameRequired   = NewError(KindValidation, "team_name_required")
	ErrInvalidTeamName    = NewError(KindValidation, "invalid_team_name")
	ErrTeamMemberExists   = NewError(KindConflict, "team_member_already_exists")
	ErrTeamMemberNotFound = NewError(KindNotFound, "team_member_not_found")
	ErrTeamQuota          = NewError(KindForbidden, "team_quota_exceeded")
)

const maxTeamsPerOwner = 100

var (
	nonSlugChars = regexp.MustCompile(`[^a-z0-9_-]+`)
	teamSlugRe   = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
)

type TeamItem struct {
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	Owner    string `json:"owner,omitzero"`
	Role     string `json:"role"`
}

type RepoTeamGrantItem struct {
	TeamID    uint   `json:"team_id"`
	TeamName  string `json:"team_name"`
	TeamSlug  string `json:"team_slug"`
	TeamOwner string `json:"team_owner,omitzero"`
	Role      string `json:"role"`
	Owned     bool   `json:"owned"`
}

type SetTeamRepoInput struct {
	ViewerID uint
	TeamSlug string
	RepoName string
	Role     string
}

type SetRepoTeamInput struct {
	ResolveRepositoryInput
	TeamSlug string
	Role     string
}

type TeamUseCase struct {
	teams       domain.TeamStore
	teamMembers domain.TeamMemberStore
	teamRepos   domain.TeamRepoGrantStore
	repos       domain.RepositoryStore
	users       domain.UserStore
	authz       *Authz
}

func NewTeamUseCase(
	teams domain.TeamStore,
	teamMembers domain.TeamMemberStore,
	teamRepos domain.TeamRepoGrantStore,
	repos domain.RepositoryStore,
	users domain.UserStore,
	authz *Authz,
) *TeamUseCase {
	return &TeamUseCase{
		teams:       teams,
		teamMembers: teamMembers,
		teamRepos:   teamRepos,
		repos:       repos,
		users:       users,
		authz:       authz,
	}
}

func (uc *TeamUseCase) List(ctx context.Context, viewerID uint) ([]TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	rows, err := uc.teams.ListByOwner(viewerID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	out := make([]TeamItem, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTeamItem(t))
	}

	return out, nil
}

func (uc *TeamUseCase) Create(ctx context.Context, viewerID uint, rawName string) (*TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	name, slug, err := normalizeTeamName(rawName)
	if err != nil {
		return nil, err
	}

	existing, err := uc.teams.ListByOwner(viewerID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	if len(existing) >= maxTeamsPerOwner {
		return nil, ErrTeamQuota
	}

	exists, err := uc.teams.ExistsByOwnerSlug(viewerID, slug)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrTeamExists
	}

	t := &domain.Team{
		OwnerUserID: viewerID,
		Name:        name,
		Slug:        slug,
	}
	if err := uc.teams.Create(t); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTeamExists
		}

		return nil, fmt.Errorf("create team: %w", err)
	}

	item := toTeamItem(*t)

	return &item, nil
}

func (uc *TeamUseCase) Get(ctx context.Context, viewerID uint, slug string) (*TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return nil, err
	}

	item := toTeamItem(*t)

	return &item, nil
}

func (uc *TeamUseCase) Update(ctx context.Context, viewerID uint, slug, rawName string) (*TeamItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return nil, err
	}

	name, newSlug, err := normalizeTeamName(rawName)
	if err != nil {
		return nil, err
	}

	if newSlug != t.Slug {
		exists, eerr := uc.teams.ExistsByOwnerSlug(viewerID, newSlug)
		if eerr != nil {
			return nil, eerr
		}

		if exists {
			return nil, ErrTeamExists
		}

		t.Slug = newSlug
	}

	t.Name = name
	if err := uc.teams.Update(t); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTeamExists
		}

		return nil, fmt.Errorf("update team: %w", err)
	}

	item := toTeamItem(*t)

	return &item, nil
}

func (uc *TeamUseCase) Delete(ctx context.Context, viewerID uint, slug string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return err
	}

	if err := uc.teamRepos.DeleteByTeam(t.ID); err != nil {
		return fmt.Errorf("delete team grants: %w", err)
	}

	if err := uc.teamMembers.DeleteByTeam(t.ID); err != nil {
		return fmt.Errorf("delete team members: %w", err)
	}

	return uc.teams.Delete(t.ID)
}

func (uc *TeamUseCase) ListMembers(ctx context.Context, viewerID uint, slug string) ([]TeamMemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return nil, err
	}

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

func (uc *TeamUseCase) AddMember(ctx context.Context, viewerID uint, slug, username string) (*TeamMemberItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return nil, err
	}

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

func (uc *TeamUseCase) RemoveMember(ctx context.Context, viewerID uint, slug, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	t, err := uc.resolve(viewerID, slug)
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

func (uc *TeamUseCase) ListRepos(ctx context.Context, viewerID uint, slug string) ([]TeamRepoGrantItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return nil, err
	}

	rows, err := uc.teamRepos.ListByTeam(t.ID)
	if err != nil {
		return nil, err
	}

	out := make([]TeamRepoGrantItem, 0, len(rows))
	for _, g := range rows {
		repo, err := uc.repos.FindByID(ctx, g.RepoID)
		if err != nil || repo == nil {
			continue
		}
		if err := uc.authz.CanReadRepo(repo, viewerID); err != nil {
			continue
		}

		out = append(out, uc.grantItem(ctx, repo, g.Role))
	}

	return out, nil
}

func (uc *TeamUseCase) SetRepo(ctx context.Context, in SetTeamRepoInput) (*TeamRepoGrantItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	t, err := uc.resolve(in.ViewerID, in.TeamSlug)
	if err != nil {
		return nil, err
	}

	role, ok := domain.ParseRepoCollabRole(in.Role)
	if !ok {
		return nil, ErrInvalidCollabRole
	}

	repo, err := uc.findRepo(ctx, in.RepoName)
	if err != nil {
		return nil, err
	}

	if err := uc.authz.CanReadRepo(repo, in.ViewerID); err != nil {
		return nil, ErrRepoNotFound
	}

	if !uc.authz.CanGrantTeamRepos(repo, in.ViewerID) {
		if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
			return nil, ErrRepoForbidden
		}
		return nil, ErrRepoForbidden
	}

	g := &domain.TeamRepoGrant{
		TeamID: t.ID,
		RepoID: repo.ID,
		Role:   role,
	}
	if err := uc.teamRepos.Upsert(g); err != nil {
		return nil, fmt.Errorf("set team repo grant: %w", err)
	}

	item := uc.grantItem(ctx, repo, role)

	return &item, nil
}

func (uc *TeamUseCase) RemoveRepo(ctx context.Context, viewerID uint, slug, repoName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	t, err := uc.resolve(viewerID, slug)
	if err != nil {
		return err
	}

	repo, err := uc.findRepo(ctx, repoName)
	if err != nil {
		return err
	}

	hasGrant := false
	if grants, gerr := uc.teamRepos.ListByTeam(t.ID); gerr == nil {
		for _, g := range grants {
			if g.RepoID == repo.ID {
				hasGrant = true
				break
			}
		}
	}

	if !hasGrant {
		if err := uc.authz.CanReadRepo(repo, viewerID); err != nil {
			return ErrRepoNotFound
		}

		if !uc.authz.CanAdminRepo(repo, viewerID) {
			return ErrRepoNotFound
		}
	}

	return uc.teamRepos.Delete(t.ID, repo.ID)
}

func (uc *TeamUseCase) ListRepoTeams(ctx context.Context, in ResolveRepositoryInput) ([]RepoTeamGrantItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := ResolveReadable(ctx, uc.repos, uc.authz, in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	rows, err := uc.teamRepos.ListByRepo(repo.ID)
	if err != nil {
		return nil, err
	}

	out := make([]RepoTeamGrantItem, 0, len(rows))
	for _, g := range rows {
		t, err := uc.teams.FindByID(g.TeamID)
		if err != nil || t == nil {
			continue
		}

		item := RepoTeamGrantItem{
			TeamID:   t.ID,
			TeamName: t.Name,
			TeamSlug: t.Slug,
			Role:     string(g.Role),
			Owned:    t.OwnerUserID == in.ViewerID,
		}
		if u, uerr := uc.users.FindByID(ctx, t.OwnerUserID); uerr == nil && u != nil {
			item.TeamOwner = u.Username
		}
		out = append(out, item)
	}

	return out, nil
}

func (uc *TeamUseCase) GrantRepoTeam(ctx context.Context, in SetRepoTeamInput) (*RepoTeamGrantItem, error) {
	item, err := uc.SetRepo(ctx, SetTeamRepoInput{
		ViewerID: in.ViewerID,
		TeamSlug: in.TeamSlug,
		RepoName: in.Name,
		Role:     in.Role,
	})
	if err != nil {
		return nil, err
	}

	t, err := uc.resolve(in.ViewerID, in.TeamSlug)
	if err != nil {
		return nil, err
	}

	return &RepoTeamGrantItem{
		TeamID:   t.ID,
		TeamName: t.Name,
		TeamSlug: t.Slug,
		Role:     item.Role,
		Owned:    true,
	}, nil
}

func (uc *TeamUseCase) RevokeRepoTeam(ctx context.Context, in ResolveRepositoryInput, teamID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo, err := ResolveReadable(ctx, uc.repos, uc.authz, in)
	if err != nil {
		return err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return ErrRepoNotFound
	}

	if teamID == 0 {
		return ErrTeamNotFound
	}

	t, err := uc.teams.FindByID(teamID)
	if err != nil || t == nil {
		return ErrTeamNotFound
	}

	rows, err := uc.teamRepos.ListByRepo(repo.ID)
	if err != nil {
		return err
	}

	for _, g := range rows {
		if g.TeamID == teamID {
			return uc.teamRepos.Delete(teamID, repo.ID)
		}
	}

	return ErrTeamNotFound
}

func (uc *TeamUseCase) findRepo(ctx context.Context, rawName string) (*domain.Repository, error) {
	name, err := normalizeRepoName(rawName)
	if err != nil {
		return nil, ErrRepoNotFound
	}

	repo, err := uc.repos.FindByRepoPath(ctx, name+".git")
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}

	return repo, nil
}

func (uc *TeamUseCase) grantItem(ctx context.Context, repo *domain.Repository, role domain.RepoCollabRole) TeamRepoGrantItem {
	item := TeamRepoGrantItem{
		RepoID:   repo.ID,
		RepoName: repo.Name,
		RepoPath: strings.TrimSuffix(repo.RepoPath, ".git"),
		Role:     string(role),
	}

	if u, err := uc.users.FindByID(ctx, repo.OwnerID); err == nil && u != nil {
		item.Owner = u.Username
	}

	return item
}

func (uc *TeamUseCase) resolve(viewerID uint, slug string) (*domain.Team, error) {
	if viewerID == 0 {
		return nil, ErrUnauthorized
	}

	t, err := uc.teams.FindByOwnerSlug(viewerID, strings.ToLower(strings.TrimSpace(slug)))
	if err != nil || t == nil {
		return nil, ErrTeamNotFound
	}

	return t, nil
}

func toTeamItem(t domain.Team) TeamItem {
	return TeamItem{
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func normalizeTeamName(raw string) (name, slug string, err error) {
	name = strings.TrimSpace(raw)
	if name == "" {
		return "", "", ErrTeamNameRequired
	}

	if len([]rune(name)) < minTeamNameLen || len([]rune(name)) > maxTeamNameLen {
		return "", "", ErrInvalidTeamName
	}

	for _, r := range name {
		if r == '/' || r == '\\' || unicode.IsControl(r) {
			return "", "", ErrInvalidTeamName
		}
	}

	slug = strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = nonSlugChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-_")
	if slug == "" || !teamSlugRe.MatchString(slug) {
		return "", "", ErrInvalidTeamName
	}

	return name, slug, nil
}
