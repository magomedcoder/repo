package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type AdminUserItem struct {
	ID        uint        `json:"id"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	HasAvatar bool        `json:"has_avatar"`
	CreatedAt time.Time   `json:"created_at"`
}

type AdminListResult struct {
	Users  []AdminUserItem `json:"users"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type AdminCreateInput struct {
	Username        string
	Email           string
	Password        string
	Role            domain.Role
	CurrentPassword string
}

type AdminUpdateInput struct {
	Email           *string
	Role            *domain.Role
	Password        *string
	CurrentPassword string
}

type AdminOverview struct {
	Users        int             `json:"users"`
	Repos        int             `json:"repos"`
	MirrorsError int64           `json:"mirrors_error"`
	Events24h    int             `json:"events_24h"`
	RecentLogs   []AccessLogItem `json:"recent_logs"`
}

type AdminRepoItem struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Owner           string     `json:"owner"`
	RepoPath        string     `json:"repo_path"`
	IsPrivate       bool       `json:"is_private"`
	ShowOnHome      bool       `json:"show_on_home"`
	MirrorDirection string     `json:"mirror_direction,omitzero"`
	MirrorStatus    string     `json:"mirror_status,omitzero"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastActivityAt  *time.Time `json:"last_activity_at,omitzero"`
}

type AdminRepoListResult struct {
	Repos  []AdminRepoItem `json:"repos"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type AdminUseCase struct {
	users    domain.UserStore
	sessions domain.SessionStore
	tokens   domain.AccessTokenStore
	auth     *AuthUseCase
	hasher   domain.PasswordHasher
	gate     *PasswordGate
	repos    domain.RepositoryStore
	logs     domain.AccessLogStore
	files    repoFileRemover

	collabs     domain.RepoCollaboratorStore
	teams       domain.TeamStore
	teamMembers domain.TeamMemberStore
	teamGrants  domain.TeamRepoGrantStore
	nameHolds   domain.RepoNameHoldStore
}

func (uc *AdminUseCase) SetAccessCleanup(collabs domain.RepoCollaboratorStore, teams domain.TeamStore, members domain.TeamMemberStore, grants domain.TeamRepoGrantStore) {
	uc.collabs = collabs
	uc.teams = teams
	uc.teamMembers = members
	uc.teamGrants = grants
}

func (uc *AdminUseCase) cleanupUserAccess(userID uint) error {
	if uc.collabs != nil {
		if err := uc.collabs.DeleteByUserID(userID); err != nil {
			return fmt.Errorf("delete collaborators: %w", err)
		}
	}

	if uc.teams != nil {
		owned, err := uc.teams.ListByOwner(userID)
		if err != nil {
			return fmt.Errorf("list owned teams: %w", err)
		}

		for _, t := range owned {
			if uc.teamGrants != nil {
				if err := uc.teamGrants.DeleteByTeam(t.ID); err != nil {
					return fmt.Errorf("delete team grants: %w", err)
				}
			}

			if uc.teamMembers != nil {
				if err := uc.teamMembers.DeleteByTeam(t.ID); err != nil {
					return fmt.Errorf("delete team members: %w", err)
				}
			}

			if err := uc.teams.Delete(t.ID); err != nil {
				return fmt.Errorf("delete team: %w", err)
			}
		}
	}

	return nil
}

func NewAdminUseCase(users domain.UserStore, sessions domain.SessionStore, tokens domain.AccessTokenStore, auth *AuthUseCase, hasher domain.PasswordHasher) *AdminUseCase {
	return &AdminUseCase{
		users:    users,
		sessions: sessions,
		tokens:   tokens,
		auth:     auth,
		hasher:   hasher,
		gate:     NewPasswordGate(hasher),
	}
}

func (uc *AdminUseCase) SetPasswordGate(g *PasswordGate) {
	if uc != nil && g != nil {
		uc.gate = g
	}
}

type repoFileRemover interface {
	Remove(path string) error
}

func (uc *AdminUseCase) SetRepoFiles(files repoFileRemover) {
	if uc != nil {
		uc.files = files
	}
}

func (uc *AdminUseCase) SetNameHolds(h domain.RepoNameHoldStore) {
	if uc != nil {
		uc.nameHolds = h
	}
}

func (uc *AdminUseCase) SetDataStores(repos domain.RepositoryStore, logs domain.AccessLogStore) {
	uc.repos = repos
	uc.logs = logs
}

func (uc *AdminUseCase) requireSiteAdmin(ctx context.Context, actorID uint) error {
	if actorID == 0 {
		return ErrUnauthorized
	}

	actor, err := uc.users.FindByID(ctx, actorID)
	if err != nil || actor == nil {
		return ErrUnauthorized
	}

	if !actor.IsAdmin() {
		return ErrAdminRequired
	}

	return nil
}

func (uc *AdminUseCase) Overview(ctx context.Context, actorID uint) (*AdminOverview, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return nil, err
	}

	out := &AdminOverview{
		RecentLogs: []AccessLogItem{},
	}

	if _, total, err := uc.users.List(domain.UserListFilter{Limit: 1}); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	} else {
		out.Users = total
	}

	if uc.repos != nil {
		if n, err := uc.repos.CountAll(ctx); err != nil {
			return nil, fmt.Errorf("count repos: %w", err)
		} else {
			out.Repos = int(n)
		}

		if n, err := uc.repos.CountMirrorsInError(ctx, time.Now().Add(-time.Hour)); err != nil {
			return nil, fmt.Errorf("count mirror errors: %w", err)
		} else {
			out.MirrorsError = n
		}
	}

	if uc.logs != nil {
		since := time.Now().Add(-24 * time.Hour)
		items, total, err := uc.logs.List(domain.AccessLogFilter{
			Since: &since,
			Limit: 8,
		})
		if err != nil {
			return nil, fmt.Errorf("list access logs: %w", err)
		}

		out.Events24h = total
		out.RecentLogs = make([]AccessLogItem, 0, len(items))
		for _, item := range items {
			out.RecentLogs = append(out.RecentLogs, AccessLogItem{
				ID:        item.ID,
				Action:    item.Action,
				IP:        item.IP,
				UserAgent: item.UserAgent,
				UserID:    item.UserID,
				Username:  item.Username,
				Path:      item.Path,
				Repo:      item.Repo,
				Ref:       item.Ref,
				CreatedAt: item.CreatedAt,
			})
		}
	}

	return out, nil
}

type AdminListInput struct {
	Query  string
	Role   *domain.Role
	Sort   string
	Limit  int
	Offset int
}

func (uc *AdminUseCase) ListUsers(ctx context.Context, actorID uint, in AdminListInput) (*AdminListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return nil, err
	}

	offset, limit := ClampPagination(in.Offset, in.Limit, DefaultPageLimit, 100)

	users, total, err := uc.users.List(domain.UserListFilter{
		Query:  in.Query,
		Role:   in.Role,
		Sort:   in.Sort,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}

	items := make([]AdminUserItem, 0, len(users))
	for i := range users {
		items = append(items, toAdminUserItem(&users[i]))
	}

	return &AdminListResult{
		Users:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (uc *AdminUseCase) ListRepos(ctx context.Context, actorID uint, filter domain.AdminRepoFilter) (*AdminRepoListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return nil, err
	}

	if uc.repos == nil {
		return nil, ErrUnauthorized
	}

	offset, limit := ClampPagination(filter.Offset, filter.Limit, DefaultPageLimit, 100)
	filter.Limit = limit
	filter.Offset = offset

	repos, total, err := uc.repos.ListAdmin(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}

	owners := map[uint]string{}
	items := make([]AdminRepoItem, 0, len(repos))
	for i := range repos {
		r := &repos[i]
		owner, ok := owners[r.OwnerID]
		if !ok {
			if u, uerr := uc.users.FindByID(ctx, r.OwnerID); uerr == nil {
				owner = u.Username
			}

			owners[r.OwnerID] = owner
		}

		items = append(items, AdminRepoItem{
			ID:              r.ID,
			Name:            r.Name,
			Description:     adminRepoDescription(r),
			Owner:           owner,
			RepoPath:        r.RepoPath,
			IsPrivate:       r.IsPrivate,
			ShowOnHome:      r.ShowOnHome,
			MirrorDirection: r.MirrorDirection,
			MirrorStatus:    r.MirrorStatus,
			UpdatedAt:       r.UpdatedAt,
			LastActivityAt:  r.LastActivityAt,
		})
	}

	return &AdminRepoListResult{
		Repos:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (uc *AdminUseCase) Create(ctx context.Context, actorID uint, in AdminCreateInput) (*AdminUserItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return nil, err
	}

	if !in.Role.Valid() {
		return nil, ErrInvalidRole
	}

	if err := uc.requireStepUp(ctx, actorID, in.CurrentPassword); err != nil {
		return nil, err
	}

	user, err := uc.auth.CreateUser(ctx, RegisterInput{
		Username: in.Username,
		Email:    in.Email,
		Password: in.Password,
		Role:     in.Role,
	})
	if err != nil {
		return nil, err
	}

	item := toAdminUserItem(user)

	return &item, nil
}

func (uc *AdminUseCase) Update(ctx context.Context, actorID, id uint, in AdminUpdateInput) (*AdminUserItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	prevRole := user.Role
	if !prevRole.Valid() {
		prevRole = domain.RoleUser
	}

	roleChange := in.Role != nil && *in.Role != prevRole
	passwordChange := in.Password != nil
	emailChange := false
	if in.Email != nil {
		nextEmail := strings.ToLower(strings.TrimSpace(*in.Email))
		emailChange = nextEmail != "" && !strings.EqualFold(nextEmail, user.Email)
	}

	if roleChange || passwordChange || emailChange {
		if err := uc.requireStepUp(ctx, actorID, in.CurrentPassword); err != nil {
			return nil, err
		}
	}

	if in.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*in.Email))
		if email == "" {
			return nil, ErrEmailRequired
		}

		if !emailPattern.MatchString(email) {
			return nil, ErrInvalidEmail
		}

		if email != user.Email {
			taken, err := uc.users.ExistsByEmail(email)
			if err != nil {
				return nil, fmt.Errorf("check email: %w", err)
			}

			if taken {
				return nil, ErrEmailTaken
			}

			user.Email = email
		}
	}

	if in.Role != nil {
		role := *in.Role
		if !role.Valid() {
			return nil, ErrInvalidRole
		}

		if role != prevRole {
			if actorID == id && prevRole == domain.RoleAdmin && role != domain.RoleAdmin {
				return nil, ErrCannotModifySelf
			}

			if prevRole == domain.RoleAdmin && role != domain.RoleAdmin {
				if err := uc.ensureNotLastAdmin(); err != nil {
					return nil, err
				}
			}

			user.Role = role
			if err := uc.revokeCredentials(id); err != nil {
				return nil, err
			}
		}
	}

	if in.Password != nil {
		password := strings.TrimSpace(*in.Password)
		if err := validateNewPassword(password); err != nil {
			return nil, err
		}

		hash, err := uc.hasher.Hash(password)
		if err != nil {
			if isHasherBusy(err) {
				return nil, ErrBusy
			}

			return nil, fmt.Errorf("hash password: %w", err)
		}

		user.PasswordHash = hash
		if err := uc.revokeCredentials(id); err != nil {
			return nil, err
		}

		slog.Info("password reset", "actor_id", actorID, "user_id", id)
	}

	if err := uc.users.Update(user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	item := toAdminUserItem(user)

	return &item, nil
}

func (uc *AdminUseCase) Delete(ctx context.Context, actorID, id uint, currentPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := uc.requireSiteAdmin(ctx, actorID); err != nil {
		return err
	}

	if actorID == id {
		return ErrCannotModifySelf
	}

	if err := uc.requireStepUp(ctx, actorID, currentPassword); err != nil {
		return err
	}

	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	if user.IsAdmin() {
		if err := uc.ensureNotLastAdmin(); err != nil {
			return err
		}
	}

	if err := uc.sessions.DeleteByUserID(id); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	if uc.tokens != nil {
		if err := uc.tokens.DeleteByUserID(id); err != nil {
			return fmt.Errorf("revoke access tokens: %w", err)
		}
	}

	if err := uc.cleanupUserAccess(id); err != nil {
		return err
	}

	if err := uc.deleteOwnedRepos(ctx, id); err != nil {
		return err
	}

	if err := uc.users.Delete(id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
}

func (uc *AdminUseCase) deleteOwnedRepos(ctx context.Context, ownerID uint) error {
	if uc.repos == nil {
		return nil
	}

	for range 1000 {
		repos, err := uc.repos.ListByOwnerID(ctx, ownerID, 100, 0, domain.RepoOrderUpdated)
		if err != nil {
			return fmt.Errorf("list owned repositories: %w", err)
		}

		if len(repos) == 0 {
			return nil
		}

		for i := range repos {
			r := &repos[i]
			if uc.files != nil && r.Path != "" {
				if err := uc.files.Remove(r.Path); err != nil {
					return fmt.Errorf("remove repository files repo_id=%d: %w", r.ID, err)
				}
			}

			if err := uc.repos.Delete(r.ID); err != nil {
				return fmt.Errorf("delete owned repository: %w", err)
			}

			if uc.nameHolds != nil && r.Name != "" {
				if err := uc.nameHolds.Hold(ctx, r.Name, time.Now().UTC().Add(nameHoldTTL)); err != nil {
					slog.Warn("name hold", "name", r.Name, "err", err)
				}
			}
		}
	}

	return nil
}

func adminRepoDescription(r *domain.Repository) string {
	if r.IsPrivate {
		return ""
	}

	return r.Description
}

func (uc *AdminUseCase) revokeCredentials(userID uint) error {
	if err := uc.sessions.DeleteByUserID(userID); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	if uc.tokens != nil {
		if err := uc.tokens.DeleteByUserID(userID); err != nil {
			return fmt.Errorf("revoke access tokens: %w", err)
		}
	}

	return nil
}

func (uc *AdminUseCase) ensureNotLastAdmin() error {
	n, err := uc.users.CountByRole(domain.RoleAdmin)
	if err != nil {
		return err
	}

	if n <= 1 {
		return ErrLastAdmin
	}

	return nil
}

func (uc *AdminUseCase) requireStepUp(ctx context.Context, actorID uint, currentPassword string) error {
	currentPassword = strings.TrimSpace(currentPassword)
	if currentPassword == "" {
		return ErrPasswordRequired
	}

	actor, err := uc.users.FindByID(ctx, actorID)
	if err != nil {
		return ErrUnauthorized
	}

	return uc.gate.Verify(actor.ID, actor.PasswordHash, currentPassword, ErrCurrentPasswordWrong)
}

func toAdminUserItem(user *domain.User) AdminUserItem {
	role := user.Role
	if !role.Valid() {
		role = domain.RoleUser
	}

	return AdminUserItem{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      role,
		HasAvatar: user.AvatarPath != "",
		CreatedAt: user.CreatedAt,
	}
}
