package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrInvalidCollabRole    = NewError(KindValidation, "invalid_collaborator_role")
	ErrCollaboratorNotFound = NewError(KindNotFound, "collaborator_not_found")
	ErrAccessNotFound       = NewError(KindNotFound, "not_found")
	ErrCannotCollabSelf     = NewError(KindValidation, "cannot_collaborator_self")
)

type CollaboratorItem struct {
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccessUseCase struct {
	repos     domain.RepositoryStore
	users     domain.UserStore
	collabs   domain.RepoCollaboratorStore
	teamRepos domain.TeamRepoGrantStore
	authz     *Authz
}

func NewAccessUseCase(
	repos domain.RepositoryStore,
	users domain.UserStore,
	collabs domain.RepoCollaboratorStore,
	authz *Authz,
) *AccessUseCase {
	return &AccessUseCase{
		repos:   repos,
		users:   users,
		collabs: collabs,
		authz:   authz,
	}
}

func (uc *AccessUseCase) SetTeamRepos(store domain.TeamRepoGrantStore) {
	if uc != nil {
		uc.teamRepos = store
	}
}

func (uc *AccessUseCase) ListCollaborators(ctx context.Context, in ResolveRepositoryInput) ([]CollaboratorItem, error) {
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

	repo, err := ResolveReadable(ctx, uc.repos, uc.authz, in.ResolveRepositoryInput)
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
		return nil, ErrAccessNotFound
	}

	if isPersonalOwner(repo, user.ID) || user.ID == in.ViewerID {
		return nil, ErrCannotCollabSelf
	}

	demotedFromAdmin := false
	if existing, err := uc.collabs.Find(repo.ID, user.ID); err == nil && existing != nil {
		if existing.Role.CanAdmin() && !role.CanAdmin() {
			if err := uc.ensureOtherRepoAdmin(repo, user.ID); err != nil {
				return nil, err
			}
			demotedFromAdmin = true
		}
	}

	c := &domain.RepoCollaborator{
		RepoID: repo.ID,
		UserID: user.ID,
		Role:   role,
	}
	if err := uc.collabs.Upsert(c); err != nil {
		return nil, fmt.Errorf("add collaborator: %w", err)
	}

	if demotedFromAdmin && uc.teamRepos != nil {
		if err := uc.teamRepos.DeleteByRepoAndTeamOwner(repo.ID, user.ID); err != nil {
			return nil, fmt.Errorf("revoke team grants: %w", err)
		}
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

	repo, err := ResolveReadable(ctx, uc.repos, uc.authz, in)
	if err != nil {
		return err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return ErrRepoNotFound
	}

	user, err := uc.users.FindByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return ErrAccessNotFound
	}

	existing, err := uc.collabs.Find(repo.ID, user.ID)
	if err != nil {
		return ErrCollaboratorNotFound
	}

	if existing.Role.CanAdmin() {
		if err := uc.ensureOtherRepoAdmin(repo, user.ID); err != nil {
			return err
		}
	}

	if err := uc.collabs.Delete(repo.ID, user.ID); err != nil {
		return err
	}

	if uc.teamRepos != nil {
		if err := uc.teamRepos.DeleteByRepoAndTeamOwner(repo.ID, user.ID); err != nil {
			return fmt.Errorf("revoke team grants: %w", err)
		}
	}

	return nil
}

func (uc *AccessUseCase) ensureOtherRepoAdmin(repo *domain.Repository, exceptUserID uint) error {
	if repo == nil {
		return ErrLastRepoAdmin
	}

	if repo.OwnerID != 0 && repo.OwnerID != exceptUserID {
		return nil
	}

	if uc.collabs != nil {
		rows, err := uc.collabs.ListByRepo(repo.ID)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if r.UserID != exceptUserID && r.Role.CanAdmin() {
				return nil
			}
		}
	}

	if uc.authz != nil && uc.authz.TeamRepos != nil {
		grants, err := uc.authz.TeamRepos.ListByRepo(repo.ID)
		if err != nil {
			return err
		}

		for _, g := range grants {
			if g.Role.CanAdmin() {
				return nil
			}
		}
	}

	return ErrLastRepoAdmin
}
