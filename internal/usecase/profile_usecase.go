package usecase

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/fsutil"
)

var (
	ErrUserNotFound   = NewError(KindNotFound, "user_not_found")
	ErrInvalidAvatar  = NewError(KindValidation, "invalid_avatar")
	ErrAvatarTooLarge = NewError(KindValidation, "avatar_too_large")
	ErrAvatarNotFound = NewError(KindNotFound, "avatar_not_found")
)

const maxAvatarBytes = 1 << 20 // 1 MiB

type ProfileItem struct {
	ID        uint        `json:"id"`
	Username  string      `json:"username"`
	Email     string      `json:"email,omitzero"`
	Role      domain.Role `json:"role"`
	HasAvatar bool        `json:"has_avatar"`
	CreatedAt time.Time   `json:"created_at"`
}

type ProfileUseCase struct {
	users     domain.UserStore
	avatarDir string
}

func NewProfileUseCase(users domain.UserStore, dataDir string) *ProfileUseCase {
	return &ProfileUseCase{
		users:     users,
		avatarDir: filepath.Join(dataDir, "avatars"),
	}
}

func (uc *ProfileUseCase) GetByUsername(ctx context.Context, username string, viewerID uint) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, ErrUserNotFound
	}

	return toProfileItem(user, viewerID == user.ID), nil
}

func (uc *ProfileUseCase) UpdateEmail(ctx context.Context, userID uint, email string) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, ErrEmailRequired
	}

	if !emailPattern.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if !strings.EqualFold(user.Email, email) {
		taken, err := uc.users.ExistsByEmail(email)
		if err != nil {
			return nil, err
		}

		if taken {
			return nil, ErrEmailTaken
		}

		user.Email = email
		if err := uc.users.Update(user); err != nil {
			if isUniqueViolation(err) {
				return nil, ErrEmailTaken
			}
			return nil, fmt.Errorf("update user: %w", err)
		}
	}

	return toProfileItem(user, true), nil
}

func (uc *ProfileUseCase) SetAvatar(ctx context.Context, userID uint, r io.Reader, contentType string, size int64) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if size > maxAvatarBytes {
		return nil, ErrAvatarTooLarge
	}

	ext, err := avatarExt(contentType)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if err := os.MkdirAll(uc.avatarDir, 0o755); err != nil {
		return nil, fmt.Errorf("avatar dir: %w", err)
	}

	filename := fmt.Sprintf("%d%s", user.ID, ext)
	dest := filepath.Join(uc.avatarDir, filename)
	tmp := dest + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}

	written, copyErr := io.Copy(f, io.LimitReader(r, maxAvatarBytes+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return nil, copyErr
	}

	if closeErr != nil {
		_ = os.Remove(tmp)
		return nil, closeErr
	}

	if written > maxAvatarBytes {
		_ = os.Remove(tmp)
		return nil, ErrAvatarTooLarge
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}

	if user.AvatarPath != "" && user.AvatarPath != filename {
		_ = os.Remove(filepath.Join(uc.avatarDir, user.AvatarPath))
	}

	user.AvatarPath = filename
	if err := uc.users.Update(user); err != nil {
		return nil, fmt.Errorf("update avatar: %w", err)
	}

	return toProfileItem(user, true), nil
}

func (uc *ProfileUseCase) ClearAvatar(ctx context.Context, userID uint) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if user.AvatarPath != "" {
		_ = os.Remove(filepath.Join(uc.avatarDir, user.AvatarPath))
		user.AvatarPath = ""
		if err := uc.users.Update(user); err != nil {
			return nil, err
		}
	}

	return toProfileItem(user, true), nil
}

func (uc *ProfileUseCase) OpenAvatar(ctx context.Context, username string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}

	user, err := uc.users.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil || user.AvatarPath == "" {
		return nil, "", ErrAvatarNotFound
	}

	path, err := fsutil.SafePath(uc.avatarDir, filepath.Join(uc.avatarDir, user.AvatarPath))
	if err != nil {
		return nil, "", ErrAvatarNotFound
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, "", ErrAvatarNotFound
	}

	return f, contentTypeForExt(filepath.Ext(user.AvatarPath)), nil
}

func toProfileItem(user *domain.User, includeEmail bool) *ProfileItem {
	role := user.Role
	if !role.Valid() {
		role = domain.RoleUser
	}

	item := &ProfileItem{
		ID:        user.ID,
		Username:  user.Username,
		Role:      role,
		HasAvatar: user.AvatarPath != "",
		CreatedAt: user.CreatedAt,
	}

	if includeEmail {
		item.Email = user.Email
	}

	return item
}

func avatarExt(contentType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/jpeg", "image/jpg":
		return ".jpg", nil
	case "image/png":
		return ".png", nil
	case "image/webp":
		return ".webp", nil
	case "image/gif":
		return ".gif", nil
	default:
		return "", ErrInvalidAvatar
	}
}

func contentTypeForExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
