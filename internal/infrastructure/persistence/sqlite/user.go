package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type userModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Username     string         `gorm:"uniqueIndex;size:39;not null"`
	Email        string         `gorm:"uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	AvatarPath   string         `gorm:"size:255"`
}

func (userModel) TableName() string {
	return "users"
}

func (m userModel) toDomain() *domain.User {
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		AvatarPath:   m.AvatarPath,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type sessionModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Token     string `gorm:"uniqueIndex;size:64;not null"`
	UserID    uint   `gorm:"index;not null"`
	ExpiresAt time.Time
}

func (sessionModel) TableName() string {
	return "sessions"
}

func (m sessionModel) toDomain() *domain.Session {
	return &domain.Session{
		ID:        m.ID,
		Token:     m.Token,
		UserID:    m.UserID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

type UserStore struct {
	db *gorm.DB
}

func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(user *domain.User) error {
	model := &userModel{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	user.ID = model.ID
	user.CreatedAt = model.CreatedAt
	user.UpdatedAt = model.UpdatedAt
	return nil
}

func (s *UserStore) FindByID(id uint) (*domain.User, error) {
	var model userModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *UserStore) FindByUsername(username string) (*domain.User, error) {
	var model userModel
	if err := s.db.Where("username = ?", username).First(&model).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *UserStore) FindByEmail(email string) (*domain.User, error) {
	var model userModel
	if err := s.db.Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}

	return model.toDomain(), nil
}

func (s *UserStore) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := s.db.Model(&userModel{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (s *UserStore) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := s.db.Model(&userModel{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (s *UserStore) Update(user *domain.User) error {
	return s.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]any{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"avatar_path":   user.AvatarPath,
	}).Error
}

type SessionStore struct {
	db *gorm.DB
}

func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(session *domain.Session) error {
	model := &sessionModel{
		Token:     session.Token,
		UserID:    session.UserID,
		ExpiresAt: session.ExpiresAt,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	session.ID = model.ID
	session.CreatedAt = model.CreatedAt
	return nil
}

func (s *SessionStore) FindByToken(token string) (*domain.Session, error) {
	var model sessionModel
	if err := s.db.Where("token = ?", token).First(&model).Error; err != nil {
		return nil, err
	}
	return model.toDomain(), nil
}

func (s *SessionStore) DeleteByToken(token string) error {
	return s.db.Where("token = ?", token).Delete(&sessionModel{}).Error
}
