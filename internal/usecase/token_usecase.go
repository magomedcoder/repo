package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrTokenNameRequired = errors.New("token name required")
	ErrTokenNotFound     = errors.New("token not found")
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
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

type CreateTokenOutput struct {
	TokenItem
	Token string `json:"token"` // shown once
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

func (uc *TokenUseCase) Create(in CreateTokenInput) (*CreateTokenOutput, error) {
	if in.UserID == 0 {
		return nil, ErrUnauthorized
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrTokenNameRequired
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

func (uc *TokenUseCase) List(userID uint) ([]TokenItem, error) {
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

func (uc *TokenUseCase) Revoke(userID, tokenID uint) error {
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
