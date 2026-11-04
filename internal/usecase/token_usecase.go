package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrTokenNameRequired  = NewError(KindValidation, "token_name_required")
	ErrTokenNotFound      = NewError(KindNotFound, "token_not_found")
	ErrTokenNameTooLong   = NewError(KindValidation, "token_name_too_long")
	ErrTooManyTokens      = NewError(KindConflict, "too_many_tokens")
	ErrTokenExpiryInvalid = NewError(KindValidation, "token_expiry_invalid")
)

const (
	maxTokensPerUser = 50
	maxTokenNameLen  = 64
)

type CreateTokenInput struct {
	UserID    uint
	Name      string
	ExpiresIn *int
	ExpiresAt *time.Time
}

type TokenItem struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitzero"`
	ExpiresAt  *time.Time `json:"expires_at,omitzero"`
}

type CreateTokenOutput struct {
	TokenItem
	Token string `json:"token"`
}

type TokenUseCase struct {
	tokens domain.AccessTokenStore
	gen    domain.TokenGenerator
}

func NewTokenUseCase(tokens domain.AccessTokenStore, gen domain.TokenGenerator) *TokenUseCase {
	return &TokenUseCase{
		tokens: tokens,
		gen:    gen,
	}
}

func (uc *TokenUseCase) Create(ctx context.Context, in CreateTokenInput) (*CreateTokenOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.UserID == 0 {
		return nil, ErrUnauthorized
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrTokenNameRequired
	}
	if utf8.RuneCountInString(name) > maxTokenNameLen {
		return nil, ErrTokenNameTooLong
	}

	existing, err := uc.tokens.ListByUserID(in.UserID)
	if err != nil {
		return nil, fmt.Errorf("list tokens: %w", err)
	}

	if len(existing) >= maxTokensPerUser {
		return nil, ErrTooManyTokens
	}

	expiresAt, err := resolveTokenExpiry(in.ExpiresIn, in.ExpiresAt, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	raw, err := uc.gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	raw = "repo_" + raw

	token := &domain.AccessToken{
		UserID:    in.UserID,
		Name:      name,
		TokenHash: hashAccessToken(raw),
		Prefix:    formatTokenPrefix(raw),
		ExpiresAt: expiresAt,
	}

	if err := uc.tokens.Create(token); err != nil {
		return nil, fmt.Errorf("save token: %w", err)
	}

	return &CreateTokenOutput{
		TokenItem: toTokenItem(token),
		Token:     raw,
	}, nil
}

func (uc *TokenUseCase) List(ctx context.Context, userID uint) ([]TokenItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if userID == 0 {
		return nil, ErrUnauthorized
	}

	tokens, err := uc.tokens.ListByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("list tokens: %w", err)
	}

	items := make([]TokenItem, 0, len(tokens))
	for i := range tokens {
		items = append(items, toTokenItem(&tokens[i]))
	}

	return items, nil
}

func (uc *TokenUseCase) Revoke(ctx context.Context, userID, tokenID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	if err := uc.tokens.DeleteByUserAndID(userID, tokenID); err != nil {
		return ErrTokenNotFound
	}

	return nil
}

func hashAccessToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func formatTokenPrefix(raw string) string {
	if len(raw) <= 12 {
		return raw
	}

	return raw[:12]
}

func toTokenItem(t *domain.AccessToken) TokenItem {
	return TokenItem{
		ID:         t.ID,
		Name:       t.Name,
		Prefix:     t.Prefix,
		CreatedAt:  t.CreatedAt,
		LastUsedAt: t.LastUsedAt,
		ExpiresAt:  t.ExpiresAt,
	}
}

func resolveTokenExpiry(days *int, at *time.Time, now time.Time) (*time.Time, error) {
	if days != nil && at != nil {
		return nil, ErrTokenExpiryInvalid
	}

	if days != nil {
		if *days <= 0 || *days > 3650 {
			return nil, ErrTokenExpiryInvalid
		}

		exp := now.Add(time.Duration(*days) * 24 * time.Hour)
		return &exp, nil
	}

	if at != nil {
		if !at.After(now) {
			return nil, ErrTokenExpiryInvalid
		}
		exp := at.UTC()
		return &exp, nil
	}

	return nil, nil
}

func AccessTokenExpired(token *domain.AccessToken, now time.Time) bool {
	return token != nil && token.ExpiresAt != nil && !token.ExpiresAt.After(now)
}
