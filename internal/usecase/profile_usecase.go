package usecase

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrUserNotFound    = errors.New("user_not_found")
	ErrInvalidAvatar   = errors.New("invalid_avatar")
	ErrAvatarTooLarge  = errors.New("avatar_too_large")
	ErrAvatarNotFound  = errors.New("avatar_not_found")
)

const maxAvatarBytes = 1 << 20 // 1 MiB

type ProfileItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email,omitempty"`
	HasAvatar bool      `json:"has_avatar"`
	CreatedAt time.Time `json:"created_at"`
}

type ProfileUseCase struct {
	users     domain.UserStore
	avatarDir string
}

func NewProfileUseCase(users domain.UserStore, avatarDir string) *ProfileUseCase {
	return &ProfileUseCase{users: users, avatarDir: avatarDir}
}

func (uc *ProfileUseCase) GetByUsername(username string, viewerID uint) (*ProfileItem, error) {
	user, err := uc.users.FindByUsername(strings.TrimSpace(username))
	if err != nil {
		return nil, ErrUserNotFound
	}

	return toProfileItem(user, viewerID == user.ID), nil
}

func (uc *ProfileUseCase) GetMe(userID uint) (*ProfileItem, error) {
	user, err := uc.users.FindByID(userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return toProfileItem(user, true), nil
}

func (uc *ProfileUseCase) UpdateEmail(userID uint, email string) (*ProfileItem, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, ErrEmailRequired
	}

	if !strings.Contains(email, "@") {
		return nil, ErrInvalidEmail
	}

	user, err := uc.users.FindByID(userID)
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
			return nil, fmt.Errorf("update user: %w", err)
		}
	}

	return toProfileItem(user, true), nil
}

func (uc *ProfileUseCase) SetAvatar(userID uint, r io.Reader, contentType string, size int64) (*ProfileItem, error) {
	if size > maxAvatarBytes {
		return nil, ErrAvatarTooLarge
	}

	ext, err := avatarExt(contentType)
	if err != nil {
		return nil, err
	}

	user, err := uc.users.FindByID(userID)
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

func (uc *ProfileUseCase) ClearAvatar(userID uint) (*ProfileItem, error) {
	user, err := uc.users.FindByID(userID)
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

func (uc *ProfileUseCase) OpenAvatar(username string) (io.ReadCloser, string, error) {
	user, err := uc.users.FindByUsername(strings.TrimSpace(username))
	if err != nil || user.AvatarPath == "" {
		return nil, "", ErrAvatarNotFound
	}

	path := filepath.Join(uc.avatarDir, user.AvatarPath)
	f, err := os.Open(path)
	if err != nil {
		return nil, "", ErrAvatarNotFound
	}

	return f, contentTypeForExt(filepath.Ext(user.AvatarPath)), nil
}

func toProfileItem(user *domain.User, includeEmail bool) *ProfileItem {
	item := &ProfileItem{
		ID:        user.ID,
		Username:  user.Username,
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
