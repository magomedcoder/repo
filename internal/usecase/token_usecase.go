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
	ErrTokenNameRequired = NewError(KindValidation, "token_name_required")
	ErrTokenNotFound     = NewError(KindNotFound, "token_not_found")
	ErrTokenNameTooLong  = NewError(KindValidation, "token_name_too_long")
	ErrTooManyTokens     = NewError(KindConflict, "too_many_tokens")
)

const (
	maxTokensPerUser = 50
	maxTokenNameLen  = 64
)

type CreateTokenInput struct {
	UserID uint
	Name   string
}

type TokenItem struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitzero"`
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
	}
}
