package usecase

import (
	"context"
	"fmt"
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
	Username string
	Email    string
	Password string
	Role     domain.Role
}

type AdminUpdateInput struct {
	Email    *string
	Role     *domain.Role
	Password *string
}

type AdminUseCase struct {
	users    domain.UserStore
	sessions domain.SessionStore
	tokens   domain.AccessTokenStore
	auth     *AuthUseCase
	hasher   domain.PasswordHasher
}

func NewAdminUseCase(users domain.UserStore, sessions domain.SessionStore, tokens domain.AccessTokenStore, auth *AuthUseCase, hasher domain.PasswordHasher) *AdminUseCase {
	return &AdminUseCase{
		users:    users,
		sessions: sessions,
		tokens:   tokens,
		auth:     auth,
		hasher:   hasher,
	}
}

func (uc *AdminUseCase) List(ctx context.Context, query string, limit, offset int) (*AdminListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	if offset < 0 {
		offset = 0
	}

	users, total, err := uc.users.List(domain.UserListFilter{
		Query:  query,
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

func (uc *AdminUseCase) Get(ctx context.Context, id uint) (*AdminUserItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	item := toAdminUserItem(user)

	return &item, nil
}

func (uc *AdminUseCase) Create(ctx context.Context, in AdminCreateInput) (*AdminUserItem, error) {
	if !in.Role.Valid() {
		return nil, ErrInvalidRole
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

	user, err := uc.users.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	prevRole := user.Role
	if !prevRole.Valid() {
		prevRole = domain.RoleUser
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
		password := *in.Password
		if password == "" {
			return nil, ErrPasswordRequired
		}

		if len(password) < minPasswordLen {
			return nil, ErrPasswordTooShort
		}

		hash, err := uc.hasher.Hash(password)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}

		user.PasswordHash = hash
		if err := uc.revokeCredentials(id); err != nil {
			return nil, err
		}
	}

	if err := uc.users.Update(user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	item := toAdminUserItem(user)

	return &item, nil
}

func (uc *AdminUseCase) Delete(ctx context.Context, actorID, id uint) error {
	if err := ctx.Err(); err != nil {
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

	if err := uc.users.Delete(id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	return nil
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
