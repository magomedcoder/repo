package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type accessTokenModel struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UserID     uint   `gorm:"index;not null"`
	Name       string `gorm:"size:100;not null"`
	TokenHash  string `gorm:"uniqueIndex;size:64;not null"`
	Prefix     string `gorm:"size:16;not null"`
	LastUsedAt *time.Time
}

func (accessTokenModel) TableName() string {
	return "access_tokens"
}

func (m accessTokenModel) toDomain() domain.AccessToken {
	return domain.AccessToken{
		ID:         m.ID,
		UserID:     m.UserID,
		Name:       m.Name,
		TokenHash:  m.TokenHash,
		Prefix:     m.Prefix,
		CreatedAt:  m.CreatedAt,
		LastUsedAt: m.LastUsedAt,
	}
}

type AccessTokenStore struct {
	db *gorm.DB
}

func NewAccessTokenStore(db *gorm.DB) *AccessTokenStore {
	return &AccessTokenStore{db: db}
}

func (s *AccessTokenStore) Create(token *domain.AccessToken) error {
	model := &accessTokenModel{
		UserID:    token.UserID,
		Name:      token.Name,
		TokenHash: token.TokenHash,
		Prefix:    token.Prefix,
	}
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*token = model.toDomain()
	return nil
}

func (s *AccessTokenStore) ListByUserID(userID uint) ([]domain.AccessToken, error) {
	var models []accessTokenModel
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.AccessToken, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *AccessTokenStore) FindByHash(hash string) (*domain.AccessToken, error) {
	var model accessTokenModel
	if err := s.db.Where("token_hash = ?", hash).First(&model).Error; err != nil {
		return nil, err
	}

	t := model.toDomain()
	return &t, nil
}

func (s *AccessTokenStore) DeleteByUserAndID(userID, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&accessTokenModel{})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *AccessTokenStore) TouchLastUsed(id uint, at time.Time) error {
	return s.db.Model(&accessTokenModel{}).Where("id = ?", id).Update("last_used_at", at).Error
}
