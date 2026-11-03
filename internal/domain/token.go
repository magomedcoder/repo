package domain

import "time"

type AccessToken struct {
	ID         uint
	UserID     uint
	Name       string
	TokenHash  string
	Prefix     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

type AccessTokenStore interface {
	Create(token *AccessToken) error

	ListByUserID(userID uint) ([]AccessToken, error)

	FindByHash(hash string) (*AccessToken, error)

	DeleteByUserAndID(userID, id uint) error

	DeleteByUserID(userID uint) error

	TouchLastUsed(id uint, at time.Time) error
}
