package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type sshKeyModel struct {
	ID          uint `gorm:"primaryKey"`
	CreatedAt   time.Time
	UserID      uint   `gorm:"index;not null"`
	Title       string `gorm:"size:100;not null"`
	PublicKey   string `gorm:"size:2048;not null"`
	Fingerprint string `gorm:"uniqueIndex;size:80;not null"`
	LastUsedAt  *time.Time
}

func (sshKeyModel) TableName() string {
	return "ssh_keys"
}

func (m sshKeyModel) toDomain() domain.SSHKey {
	return domain.SSHKey{
		ID:          m.ID,
		UserID:      m.UserID,
		Title:       m.Title,
		PublicKey:   m.PublicKey,
		Fingerprint: m.Fingerprint,
		CreatedAt:   m.CreatedAt,
		LastUsedAt:  m.LastUsedAt,
	}
}

type SSHKeyStore struct {
	db *gorm.DB
}

func NewSSHKeyStore(db *gorm.DB) *SSHKeyStore {
	return &SSHKeyStore{db: db}
}

func (s *SSHKeyStore) Create(key *domain.SSHKey) error {
	model := &sshKeyModel{
		UserID:      key.UserID,
		Title:       key.Title,
		PublicKey:   key.PublicKey,
		Fingerprint: key.Fingerprint,
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*key = model.toDomain()

	return nil
}

func (s *SSHKeyStore) ListByUserID(userID uint) ([]domain.SSHKey, error) {
	var models []sshKeyModel
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.SSHKey, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *SSHKeyStore) FindByFingerprint(fingerprint string) (*domain.SSHKey, error) {
	var model sshKeyModel
	if err := s.db.Where("fingerprint = ?", fingerprint).First(&model).Error; err != nil {
		return nil, err
	}

	key := model.toDomain()

	return &key, nil
}

func (s *SSHKeyStore) DeleteByUserAndID(userID, id uint) error {
	res := s.db.Where("user_id = ? AND id = ?", userID, id).Delete(&sshKeyModel{})
	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (s *SSHKeyStore) TouchLastUsed(id uint, at time.Time) error {
	return s.db.Model(&sshKeyModel{}).Where("id = ?", id).Update("last_used_at", at).Error
}
