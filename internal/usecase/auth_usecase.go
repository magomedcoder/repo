package usecase

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrUsernameRequired   = errors.New("username_required")
	ErrEmailRequired      = errors.New("email_required")
	ErrPasswordRequired   = errors.New("password_required")
	ErrInvalidUsername    = errors.New("invalid_username")
	ErrInvalidEmail       = errors.New("invalid_email")
	ErrPasswordTooShort   = errors.New("password_too_short")
	ErrUsernameTaken      = errors.New("username_taken")
	ErrEmailTaken         = errors.New("email_taken")
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrSessionExpired     = errors.New("session_expired")
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const (
	minPasswordLen = 8
	sessionTTL     = 30 * 24 * time.Hour
	minUsernameLen = 3
	maxUsernameLen = 39
)

type AuthOutput struct {
	User  *domain.User
	Token string
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Login    string // username or email
	Password string
}

type AuthUseCase struct {
	users    domain.UserStore
	sessions domain.SessionStore
	orgs     domain.OrganizationStore
	hasher   domain.PasswordHasher
	tokens   domain.TokenGenerator
	now      func() time.Time
}

func NewAuthUseCase(
	users domain.UserStore,
	sessions domain.SessionStore,
	orgs domain.OrganizationStore,
	hasher domain.PasswordHasher,
	tokens domain.TokenGenerator,
) *AuthUseCase {
	return &AuthUseCase{
		users:    users,
		sessions: sessions,
		orgs:     orgs,
		hasher:   hasher,
		tokens:   tokens,
		now:      time.Now,
	}
}

func (uc *AuthUseCase) Register(in RegisterInput) (*AuthOutput, error) {
	username := strings.ToLower(strings.TrimSpace(in.Username))
	email := strings.ToLower(strings.TrimSpace(in.Email))
	password := in.Password

	if err := validateUsername(username); err != nil {
		return nil, err
	}

	if email == "" {
		return nil, ErrEmailRequired
	}

	if !emailPattern.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	if password == "" {
		return nil, ErrPasswordRequired
	}

	if len(password) < minPasswordLen {
		return nil, ErrPasswordTooShort
	}

	taken, err := uc.users.ExistsByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}

	if taken {
		return nil, ErrUsernameTaken
	}

	if uc.orgs != nil {
		taken, err = uc.orgs.ExistsBySlug(username)
		if err != nil {
			return nil, fmt.Errorf("check org slug: %w", err)
		}
		
		if taken {
			return nil, ErrUsernameTaken
		}
	}

	if _, ok := reservedSlugs[username]; ok {
		return nil, ErrReservedSlug
	}

	taken, err = uc.users.ExistsByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if taken {
		return nil, ErrEmailTaken
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
	}
	if err := uc.users.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := uc.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: sanitizeUser(user), Token: token}, nil
}

func (uc *AuthUseCase) Login(in LoginInput) (*AuthOutput, error) {
	login := strings.ToLower(strings.TrimSpace(in.Login))
	if login == "" || in.Password == "" {
		return nil, ErrInvalidCredentials
	}

	var (
		user *domain.User
		err  error
	)
	if strings.Contains(login, "@") {
		user, err = uc.users.FindByEmail(login)
	} else {
		user, err = uc.users.FindByUsername(login)
	}
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := uc.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: sanitizeUser(user), Token: token}, nil
}

func (uc *AuthUseCase) Logout(token string) error {
	if token == "" {
		return nil
	}

	return uc.sessions.DeleteByToken(token)
}

func (uc *AuthUseCase) Me(token string) (*domain.User, error) {
	user, _, err := uc.Authenticate(token)
	return user, err
}

func (uc *AuthUseCase) Authenticate(token string) (*domain.User, *domain.Session, error) {
	if token == "" {
		return nil, nil, ErrUnauthorized
	}

	session, err := uc.sessions.FindByToken(token)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}

	if !session.ExpiresAt.After(uc.now()) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	user, err := uc.users.FindByID(session.UserID)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}

	return sanitizeUser(user), session, nil
}

func (uc *AuthUseCase) createSession(userID uint) (string, error) {
	token, err := uc.tokens.Generate()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	session := &domain.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: uc.now().Add(sessionTTL),
	}
	if err := uc.sessions.Create(session); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

func validateUsername(username string) error {
	if username == "" {
		return ErrUsernameRequired
	}

	if len(username) < minUsernameLen || len(username) > maxUsernameLen {
		return ErrInvalidUsername
	}

	if !usernamePattern.MatchString(username) {
		return ErrInvalidUsername
	}

	return nil
}

func sanitizeUser(user *domain.User) *domain.User {
	if user == nil {
		return nil
	}

	cp := *user
	cp.PasswordHash = ""
	return &cp
}
