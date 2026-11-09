package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

type Session struct {
	gormutil.Repo[model.Session]
}

func NewSessionRepo(db *gorm.DB) *Session {
	return &Session{Repo: gormutil.NewRepo[model.Session](db)}
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

func (s *Session) Create(session *domain.Session) error {
	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = time.Now().UTC()
	}

	m := &model.Session{
		Token:      hashSessionToken(session.Token),
		UserID:     session.UserID,
		IP:         truncateRunes(session.IP, 64),
		UserAgent:  truncateRunes(session.UserAgent, 512),
		ExpiresAt:  session.ExpiresAt,
		LastSeenAt: lastSeen,
	}
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	session.ID = m.ID
	session.CreatedAt = m.CreatedAt
	session.LastSeenAt = m.LastSeenAt
	session.IP = m.IP
	session.UserAgent = m.UserAgent

	return nil
}

func (s *Session) FindByToken(ctx context.Context, token string) (*domain.Session, error) {
	row, err := s.FindByWhere(ctx, "token = ?", hashSessionToken(token))
	if err != nil {
		return nil, mapDBErr(err)
	}

	session := row.ToDomain()
	session.Token = token

	return session, nil
}

func (s *Session) ListByUserID(userID uint) ([]domain.Session, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "user_id = ?", "last_seen_at DESC, id DESC", userID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Session, 0, len(models))
	for i := range models {
		out = append(out, *models[i].ToDomain())
	}

	return out, nil
}

func (s *Session) DeleteByUserAndID(userID, sessionID uint) error {
	n, err := s.DeleteWhereRows(context.Background(), "user_id = ? AND id = ?", userID, sessionID)
	if err != nil {
		return err
	}

	if n == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *Session) Touch(session *domain.Session) error {
	_, err := s.UpdateByID(context.Background(), session.ID, map[string]any{
		"last_seen_at": session.LastSeenAt,
		"expires_at":   session.ExpiresAt,
	})
	return err
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}

	r := []rune(s)
	if len(r) <= max {
		return s
	}

	return string(r[:max])
}

func (s *Session) DeleteByToken(token string) error {
	return s.DeleteWhere(context.Background(), "token = ?", hashSessionToken(token))
}

func (s *Session) DeleteByUserIDExcept(userID uint, keepToken string) error {
	if keepToken == "" {
		return s.DeleteByUserID(userID)
	}

	return s.DeleteWhere(context.Background(), "user_id = ? AND token <> ?", userID, hashSessionToken(keepToken))
}

func (s *Session) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return s.DeleteWhereRows(ctx, "expires_at < ?", now.In(time.Local))
}

func (s *Session) DeleteByUserID(userID uint) error {
	return s.DeleteWhere(context.Background(), "user_id = ?", userID)
}
