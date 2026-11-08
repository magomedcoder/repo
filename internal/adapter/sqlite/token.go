package sqlite

import (
	"context"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type AccessToken struct {
	gormutil.Repo[model.AccessToken]
}

func NewAccessTokenRepo(db *gorm.DB) *AccessToken {
	return &AccessToken{Repo: gormutil.NewRepo[model.AccessToken](db)}
}

func (s *AccessToken) Create(token *domain.AccessToken) error {
	scope := token.Scope
	if scope == "" {
		scope = domain.TokenScopeGitRead
	}

	m := &model.AccessToken{
		UserID:    token.UserID,
		Name:      token.Name,
		TokenHash: token.TokenHash,
		Prefix:    token.Prefix,
		Scope:     scope,
		RepoIDs:   model.EncodeRepoIDs(token.RepoIDs),
		ExpiresAt: token.ExpiresAt,
	}
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	*token = m.ToDomain()
	return nil
}

func (s *AccessToken) ListByUserID(userID uint) ([]domain.AccessToken, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "user_id = ?", "created_at DESC", userID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.AccessToken, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *AccessToken) FindByHash(hash string) (*domain.AccessToken, error) {
	row, err := s.FindByWhere(context.Background(), "token_hash = ?", hash)
	if err != nil {
		return nil, mapDBErr(err)
	}

	t := row.ToDomain()
	return &t, nil
}

func (s *AccessToken) DeleteByUserAndID(userID, id uint) error {
	n, err := s.DeleteWhereRows(context.Background(), "user_id = ? AND id = ?", userID, id)
	if err != nil {
		return err
	}

	if n == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *AccessToken) DeleteByUserID(userID uint) error {
	return s.DeleteWhere(context.Background(), "user_id = ?", userID)
}

func (s *AccessToken) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.DeleteWhereRows(ctx, "expires_at IS NOT NULL AND expires_at < ?", now.In(time.Local))
}

func (s *AccessToken) TouchLastUsed(id uint, at time.Time) error {
	_, err := s.UpdateColumnByID(context.Background(), id, "last_used_at", at)
	return err
}
