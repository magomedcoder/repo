package domain

import (
	"strings"
	"time"
)

type Role int

const (
	RoleUser  Role = 0
	RoleAdmin Role = 1
)

func (r Role) Valid() bool {
	return r == RoleUser || r == RoleAdmin
}

func (r Role) String() string {
	if r == RoleAdmin {
		return "admin"
	}

	return "user"
}

func ParseRole(s string) (Role, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "admin":
		return RoleAdmin, true
	case "user", "":
		return RoleUser, true
	default:
		return 0, false
	}
}

type User struct {
	ID           uint
	Username     string
	Email        string
	PasswordHash string
	AvatarPath   string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (u *User) IsAdmin() bool {
	return u != nil && u.Role == RoleAdmin
}

type Session struct {
	ID         uint
	Token      string
	UserID     uint
	ExpiresAt  time.Time
	LastSeenAt time.Time
	CreatedAt  time.Time
}

type UserListFilter struct {
	Query  string
	Limit  int
	Offset int
}

type UserStore interface {
	Create(user *User) error

	FindByID(id uint) (*User, error)

	FindByIDs(ids []uint) (map[uint]*User, error)

	FindByUsername(username string) (*User, error)

	FindByEmail(email string) (*User, error)

	ExistsByUsername(username string) (bool, error)

	ExistsByEmail(email string) (bool, error)

	Update(user *User) error

	Delete(id uint) error

	List(filter UserListFilter) ([]User, int, error)

	CountByRole(role Role) (int64, error)
}

type SessionStore interface {
	Create(session *Session) error

	FindByToken(token string) (*Session, error)

	Touch(session *Session) error

	DeleteByToken(token string) error

	DeleteByUserIDExcept(userID uint, keepToken string) error

	DeleteByUserID(userID uint) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)

	Compare(hash, password string) error
}

type TokenGenerator interface {
	Generate() (string, error)
}
