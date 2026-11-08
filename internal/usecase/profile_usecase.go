package usecase

import (
	"bytes"
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
	ID        uint         `json:"id"`
	Username  string       `json:"username"`
	Email     string       `json:"email,omitzero"`
	Role      *domain.Role `json:"role,omitzero"`
	HasAvatar bool         `json:"has_avatar"`
	CreatedAt time.Time    `json:"created_at"`
}

type ProfileUseCase struct {
	users        domain.UserStore
	hasher       domain.PasswordHasher
	sessions     domain.SessionStore
	accessTokens domain.AccessTokenStore
	avatarDir    string
}

func NewProfileUseCase(users domain.UserStore, hasher domain.PasswordHasher, dataDir string) *ProfileUseCase {
	return &ProfileUseCase{
		users:     users,
		hasher:    hasher,
		avatarDir: filepath.Join(dataDir, "avatars"),
	}
}

func (uc *ProfileUseCase) SetSessionStores(sessions domain.SessionStore, tokens domain.AccessTokenStore) {
	uc.sessions = sessions
	uc.accessTokens = tokens
}

func (uc *ProfileUseCase) GetByUsername(ctx context.Context, username string, viewerID uint) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, err := uc.users.FindByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return nil, ErrUserNotFound
	}

	includeEmail := viewerID == user.ID
	includeRole := includeEmail
	if !includeRole && viewerID != 0 {
		if viewer, verr := uc.users.FindByID(ctx, viewerID); verr == nil && viewer.IsAdmin() {
			includeRole = true
		}
	}

	return toProfileItem(user, includeEmail, includeRole), nil
}

func (uc *ProfileUseCase) UpdateEmail(ctx context.Context, userID uint, email, currentPassword, keepSessionToken string) (*ProfileItem, error) {
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

	if strings.EqualFold(user.Email, email) {
		return toProfileItem(user, true, true), nil
	}

	if strings.TrimSpace(currentPassword) == "" {
		return nil, ErrPasswordRequired
	}

	if uc.hasher == nil || uc.hasher.Compare(user.PasswordHash, currentPassword) != nil {
		return nil, ErrCurrentPasswordWrong
	}

	taken, err := uc.users.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}

	if taken {
		return nil, ErrEmailUnavailable
	}

	user.Email = email
	if err := uc.users.Update(user); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailUnavailable
		}
		return nil, fmt.Errorf("update user: %w", err)
	}

	if uc.sessions != nil {
		if err := uc.sessions.DeleteByUserIDExcept(userID, keepSessionToken); err != nil {
			return nil, fmt.Errorf("revoke sessions: %w", err)
		}
	}

	if uc.accessTokens != nil {
		if err := uc.accessTokens.DeleteByUserID(userID); err != nil {
			return nil, fmt.Errorf("revoke access tokens: %w", err)
		}
	}

	return toProfileItem(user, true, true), nil
}

func (uc *ProfileUseCase) SetAvatar(ctx context.Context, userID uint, r io.Reader, size int64) (*ProfileItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if size > maxAvatarBytes {
		return nil, ErrAvatarTooLarge
	}

	head := make([]byte, 16)
	n, _ := io.ReadFull(r, head)
	if n < 3 {
		return nil, ErrInvalidAvatar
	}

	ext, err := avatarExtFromMagic(head[:n])
	if err != nil {
		return nil, err
	}

	r = io.MultiReader(bytes.NewReader(head[:n]), r)

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if err := os.MkdirAll(uc.avatarDir, 0o700); err != nil {
		return nil, fmt.Errorf("avatar dir: %w", err)
	}
	_ = os.Chmod(uc.avatarDir, 0o700)

	filename := fmt.Sprintf("%d%s", user.ID, ext)
	dest := filepath.Join(uc.avatarDir, filename)
	tmp := dest + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
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

	return toProfileItem(user, true, true), nil
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

	return toProfileItem(user, true, true), nil
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

func toProfileItem(user *domain.User, includeEmail, includeRole bool) *ProfileItem {
	item := &ProfileItem{
		ID:        user.ID,
		Username:  user.Username,
		HasAvatar: user.AvatarPath != "",
		CreatedAt: user.CreatedAt,
	}

	if includeEmail {
		item.Email = user.Email
	}

	if includeRole {
		role := user.Role
		if !role.Valid() {
			role = domain.RoleUser
		}
		item.Role = &role
	}

	return item
}

func avatarExtFromMagic(header []byte) (string, error) {
	if len(header) >= 3 && header[0] == 0xff && header[1] == 0xd8 && header[2] == 0xff {
		return ".jpg", nil
	}

	if len(header) >= 8 && string(header[:8]) == "\x89PNG\r\n\x1a\n" {
		return ".png", nil
	}

	if len(header) >= 6 && (string(header[:6]) == "GIF87a" || string(header[:6]) == "GIF89a") {
		return ".gif", nil
	}

	if len(header) >= 12 && string(header[:4]) == "RIFF" && string(header[8:12]) == "WEBP" {
		return ".webp", nil
	}

	return "", ErrInvalidAvatar
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
