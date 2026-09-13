package domain

import "time"

type User struct {
	ID           uint
	Username     string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID        uint
	Token     string
	UserID    uint
	ExpiresAt time.Time
	CreatedAt time.Time
}

type UserStore interface {
	Create(user *User) error

	FindByID(id uint) (*User, error)

	FindByUsername(username string) (*User, error)

	FindByEmail(email string) (*User, error)

	ExistsByUsername(username string) (bool, error)

	ExistsByEmail(email string) (bool, error)
}

type SessionStore interface {
	Create(session *Session) error

	FindByToken(token string) (*Session, error)

	DeleteByToken(token string) error

	DeleteExpired() error
}

type PasswordHasher interface {
	Hash(password string) (string, error)

	Compare(hash, password string) error
}

type TokenGenerator interface {
	Generate() (string, error)
}
