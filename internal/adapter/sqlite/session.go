package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
)

type sessionModel struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	Token      string    `gorm:"uniqueIndex;size:128;not null"`
	UserID     uint      `gorm:"index;not null;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	User       userModel `gorm:"foreignKey:UserID;references:ID"`
	IP         string    `gorm:"size:64;not null;default:''"`
	UserAgent  string    `gorm:"size:512;not null;default:''"`
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

func (sessionModel) TableName() string {
	return "sessions"
}

func (m sessionModel) toDomain() *domain.Session {
	lastSeen := m.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = m.CreatedAt
	}

	return &domain.Session{
		ID:         m.ID,
		Token:      m.Token,
		UserID:     m.UserID,
		IP:         m.IP,
		UserAgent:  m.UserAgent,
		ExpiresAt:  m.ExpiresAt,
		LastSeenAt: lastSeen,
		CreatedAt:  m.CreatedAt,
	}
}

type SessionStore struct {
	db *gorm.DB
}

func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

func (s *SessionStore) Create(session *domain.Session) error {
	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = time.Now().UTC()
	}

	model := &sessionModel{
		Token:      hashSessionToken(session.Token),
		UserID:     session.UserID,
		IP:         truncateRunes(session.IP, 64),
		UserAgent:  truncateRunes(session.UserAgent, 512),
		ExpiresAt:  session.ExpiresAt,
		LastSeenAt: lastSeen,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	session.ID = model.ID
	session.CreatedAt = model.CreatedAt
	session.LastSeenAt = model.LastSeenAt
	session.IP = model.IP
	session.UserAgent = model.UserAgent

	return nil
}

func (s *SessionStore) FindByToken(ctx context.Context, token string) (*domain.Session, error) {
	var model sessionModel
	if err := s.db.WithContext(ctx).Where("token = ?", hashSessionToken(token)).First(&model).Error; err != nil {
		return nil, mapDBErr(err)
	}

	session := model.toDomain()
	session.Token = token

	return session, nil
}

func (s *SessionStore) ListByUserID(userID uint) ([]domain.Session, error) {
	var models []sessionModel
	if err := s.db.Where("user_id = ?", userID).Order("last_seen_at DESC, id DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Session, 0, len(models))
	for i := range models {
		out = append(out, *models[i].toDomain())
	}

	return out, nil
}

func (s *SessionStore) DeleteByUserAndID(userID, sessionID uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, sessionID).Delete(&sessionModel{})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *SessionStore) Touch(session *domain.Session) error {
	return s.db.Model(&sessionModel{}).Where("id = ?", session.ID).Updates(map[string]any{
		"last_seen_at": session.LastSeenAt,
		"expires_at":   session.ExpiresAt,
	}).Error
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

func (s *SessionStore) DeleteByToken(token string) error {
	return s.db.Where("token = ?", hashSessionToken(token)).Delete(&sessionModel{}).Error
}

func (s *SessionStore) DeleteByUserIDExcept(userID uint, keepToken string) error {
	if keepToken == "" {
		return s.DeleteByUserID(userID)
	}

	return s.db.Where("user_id = ? AND token <> ?", userID, hashSessionToken(keepToken)).Delete(&sessionModel{}).Error
}

func (s *SessionStore) DeleteByUserID(userID uint) error {
	return s.db.Where("user_id = ?", userID).Delete(&sessionModel{}).Error
}
