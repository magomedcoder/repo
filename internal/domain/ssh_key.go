package domain

import "time"

type SSHKey struct {
	ID          uint
	UserID      uint
	Title       string
	PublicKey   string
	Fingerprint string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

type SSHKeyStore interface {
	Create(key *SSHKey) error

	ListByUserID(userID uint) ([]SSHKey, error)

	FindByFingerprint(fingerprint string) (*SSHKey, error)

	DeleteByUserAndID(userID, id uint) error

	TouchLastUsed(id uint, at time.Time) error
}
