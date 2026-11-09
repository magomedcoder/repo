package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrTokenNameRequired  = NewError(KindValidation, "token_name_required")
	ErrTokenNotFound      = NewError(KindNotFound, "token_not_found")
	ErrTokenNameTooLong   = NewError(KindValidation, "token_name_too_long")
	ErrTooManyTokens      = NewError(KindConflict, "too_many_tokens")
	ErrTokenExpiryInvalid = NewError(KindValidation, "token_expiry_invalid")
	ErrTokenScopeInvalid  = NewError(KindValidation, "token_scope_invalid")
	ErrTokenReposInvalid  = NewError(KindValidation, "token_repos_invalid")
)

const (
	maxTokensPerUser       = 50
	maxTokenNameLen        = 64
	defaultTokenExpiryDays = 90
	maxTokenRepoIDs        = 50
)

type CreateTokenInput struct {
	UserID          uint
	Name            string
	Scope           string
	RepoIDs         []uint
	ExpiresIn       *int
	ExpiresAt       *time.Time
	CurrentPassword string
}

type TokenItem struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scope      string     `json:"scope"`
	RepoIDs    []uint     `json:"repo_ids"`
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
	pepper string
	users  domain.UserStore
	hasher domain.PasswordHasher
	gate   *PasswordGate
	repos  domain.RepositoryStore
	authz  *Authz
}

func NewTokenUseCase(tokens domain.AccessTokenStore, gen domain.TokenGenerator) *TokenUseCase {
	return &TokenUseCase{
		tokens: tokens,
		gen:    gen,
	}
}

func (uc *TokenUseCase) SetPepper(pepper string) {
	if uc != nil {
		uc.pepper = pepper
	}
}

func (uc *TokenUseCase) SetStepUp(users domain.UserStore, hasher domain.PasswordHasher) {
	if uc != nil {
		uc.users, uc.hasher = users, hasher
		uc.gate = NewPasswordGate(hasher)
	}
}

func (uc *TokenUseCase) SetPasswordGate(g *PasswordGate) {
	if uc != nil && g != nil {
		uc.gate = g
	}
}

func (uc *TokenUseCase) SetRepoScope(repos domain.RepositoryStore, authz *Authz) {
	if uc != nil {
		uc.repos, uc.authz = repos, authz
	}
}

func (uc *TokenUseCase) normalizeRepoScope(ctx context.Context, userID uint, ids []uint) ([]uint, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	if uc.repos == nil || uc.authz == nil {
		return nil, ErrTokenReposInvalid
	}

	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			return nil, ErrTokenReposInvalid
		}

		if slices.Contains(out, id) {
			continue
		}

		if len(out) >= maxTokenRepoIDs {
			return nil, ErrTokenReposInvalid
		}

		repo, err := uc.repos.FindByID(ctx, id)
		if err != nil || repo == nil {
			return nil, ErrTokenReposInvalid
		}

		out = append(out, id)
	}

	return out, nil
}

func (uc *TokenUseCase) authorizeTokenRepos(ctx context.Context, userID uint, scope string, ids []uint) error {
	if len(ids) == 0 || uc.repos == nil || uc.authz == nil {
		return nil
	}

	needWrite := scope == domain.TokenScopeGitWrite
	for _, id := range ids {
		repo, err := uc.repos.FindByID(ctx, id)
		if err != nil || repo == nil {
			return ErrTokenReposInvalid
		}

		if needWrite {
			if !uc.authz.CanWriteRepo(repo, userID) {
				return ErrTokenReposInvalid
			}
			continue
		}

		if uc.authz.CanReadRepo(repo, userID) != nil {
			return ErrTokenReposInvalid
		}
	}

	return nil
}

func (uc *TokenUseCase) requireStepUp(ctx context.Context, userID uint, password string) error {
	if uc.users == nil || uc.gate == nil {
		return ErrPasswordRequired
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return ErrPasswordRequired
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return ErrUnauthorized
	}

	return uc.gate.Verify(user.ID, user.PasswordHash, password, ErrCurrentPasswordWrong)
}

func (uc *TokenUseCase) Create(ctx context.Context, in CreateTokenInput) (*CreateTokenOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.UserID == 0 {
		return nil, ErrUnauthorized
	}

	if err := uc.requireStepUp(ctx, in.UserID, in.CurrentPassword); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrTokenNameRequired
	}
	if utf8.RuneCountInString(name) > maxTokenNameLen {
		return nil, ErrTokenNameTooLong
	}

	scope := domain.NormalizeTokenScope(in.Scope)
	if scope == "" {
		return nil, ErrTokenScopeInvalid
	}

	if len(in.RepoIDs) > maxTokenRepoIDs {
		return nil, ErrTokenReposInvalid
	}

	repoIDs, err := uc.normalizeRepoScope(ctx, in.UserID, in.RepoIDs)
	if err != nil {
		return nil, err
	}

	if err := uc.authorizeTokenRepos(ctx, in.UserID, scope, repoIDs); err != nil {
		return nil, err
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
		TokenHash: hashAccessToken(raw, uc.pepper),
		Prefix:    formatTokenPrefix(raw),
		Scope:     scope,
		RepoIDs:   repoIDs,
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

func (uc *TokenUseCase) Revoke(ctx context.Context, userID, tokenID uint, currentPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	if err := uc.requireStepUp(ctx, userID, currentPassword); err != nil {
		return err
	}

	if err := uc.tokens.DeleteByUserAndID(userID, tokenID); err != nil {
		return ErrTokenNotFound
	}

	return nil
}

const accessTokenHashV2Prefix = "v2:"

func hashAccessToken(raw, pepper string) string {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(raw))
	return accessTokenHashV2Prefix + hex.EncodeToString(mac.Sum(nil))
}

func formatTokenPrefix(raw string) string {
	if len(raw) <= 8 {
		return raw
	}

	return raw[:8]
}

func toTokenItem(t *domain.AccessToken) TokenItem {
	return TokenItem{
		ID:         t.ID,
		Name:       t.Name,
		Prefix:     t.Prefix,
		Scope:      t.Scope,
		RepoIDs:    repoIDsOrEmpty(t.RepoIDs),
		CreatedAt:  t.CreatedAt,
		LastUsedAt: t.LastUsedAt,
		ExpiresAt:  t.ExpiresAt,
	}
}

func repoIDsOrEmpty(ids []uint) []uint {
	if ids == nil {
		return []uint{}
	}

	return ids
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
		if exp.Sub(now) > 3650*24*time.Hour {
			return nil, ErrTokenExpiryInvalid
		}
		return &exp, nil
	}

	exp := now.Add(time.Duration(defaultTokenExpiryDays) * 24 * time.Hour)
	return &exp, nil
}

func AccessTokenExpired(token *domain.AccessToken, now time.Time) bool {
	return token != nil && token.ExpiresAt != nil && !token.ExpiresAt.After(now)
}
