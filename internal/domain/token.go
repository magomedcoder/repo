package domain

import (
	"context"
	"strings"
	"time"
)

const (
	TokenScopeGitRead  = "git:read"
	TokenScopeGitWrite = "git:write"
)

type AccessToken struct {
	ID         uint
	UserID     uint
	Name       string
	TokenHash  string
	Prefix     string
	Scope      string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  *time.Time
}

func NormalizeTokenScope(scope string) string {
	switch strings.TrimSpace(strings.ToLower(scope)) {
	case "", TokenScopeGitWrite, "write":
		return TokenScopeGitWrite
	case TokenScopeGitRead, "read":
		return TokenScopeGitRead
	default:
		return ""
	}
}

type AccessTokenStore interface {
	Create(token *AccessToken) error

	ListByUserID(userID uint) ([]AccessToken, error)

	FindByHash(hash string) (*AccessToken, error)

	DeleteByUserAndID(userID, id uint) error

	DeleteByUserID(userID uint) error

	TouchLastUsed(id uint, at time.Time) error

	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
