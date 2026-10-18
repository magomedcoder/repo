package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"git.magomedcoder.ru/repo/internal/domain"
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
	ErrRegistrationClosed = errors.New("registration_closed")
	ErrTooManyAttempts    = errors.New("too_many_attempts")
	ErrReservedSlug       = errors.New("reserved_slug")
)

var reservedSlugs = map[string]struct{}{
	"api":      {},
	"login":    {},
	"register": {},
	"folders":  {},
	"settings": {},
	"search":   {},
	"assets":   {},
	"static":   {},
	"users":    {},
	"me":       {},
}

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
	Key      string
}

type ChangePasswordInput struct {
	UserID  uint
	Current string
	Next    string
}

type AuthUseCase struct {
	users    domain.UserStore
	sessions domain.SessionStore
	hasher   domain.PasswordHasher
	tokens   domain.TokenGenerator
	attempts AttemptCounter
	secret   string
	now      func() time.Time
}

func NewAuthUseCase(
	users domain.UserStore,
	sessions domain.SessionStore,
	hasher domain.PasswordHasher,
	tokens domain.TokenGenerator,
) *AuthUseCase {
	return &AuthUseCase{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		tokens:   tokens,
		attempts: NewAttemptCounter(),
		now:      time.Now,
	}
}

func (uc *AuthUseCase) SetSessionSecret(secret string) {
	uc.secret = secret
}

func (uc *AuthUseCase) EnsureOwner(username, email, password string) error {
	if strings.TrimSpace(password) == "" {
		return nil
	}

	n, err := uc.users.Count()
	if err != nil || n > 0 {
		return err
	}

	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" {
		username = "owner"
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		email = username + "@localhost"
	}

	_, err = uc.Register(RegisterInput{
		Username: username,
		Email:    email,
		Password: password,
	})
	return err
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

	count, err := uc.users.Count()
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	if count > 0 {
		return nil, ErrRegistrationClosed
	}

	taken, err := uc.users.ExistsByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}

	if taken {
		return nil, ErrUsernameTaken
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

	if in.Key != "" && uc.attempts != nil && uc.attempts.Locked(in.Key) {
		return nil, ErrTooManyAttempts
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
		if in.Key != "" && uc.attempts != nil {
			uc.attempts.Fail(in.Key)
		}
		return nil, ErrInvalidCredentials
	}

	if in.Key != "" && uc.attempts != nil {
		uc.attempts.Reset(in.Key)
	}

	token, err := uc.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: sanitizeUser(user), Token: token}, nil
}

func (uc *AuthUseCase) ChangePassword(in ChangePasswordInput) error {
	if in.UserID == 0 {
		return ErrUnauthorized
	}

	if len(in.Next) < minPasswordLen {
		return ErrPasswordTooShort
	}

	user, err := uc.users.FindByID(in.UserID)
	if err != nil {
		return ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Current); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := uc.hasher.Hash(in.Next)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = hash
	return uc.users.Update(user)
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

	if !uc.validSessionToken(token) {
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
	token = uc.signSession(token)

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

func (uc *AuthUseCase) signSession(token string) string {
	if uc.secret == "" {
		return token
	}

	mac := hmac.New(sha256.New, []byte(uc.secret))
	_, _ = mac.Write([]byte(token))
	return token + "." + hex.EncodeToString(mac.Sum(nil))
}

func (uc *AuthUseCase) validSessionToken(token string) bool {
	if uc.secret == "" {
		return true
	}

	raw, sig, ok := strings.Cut(token, ".")
	if !ok || raw == "" || sig == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(uc.secret))
	_, _ = mac.Write([]byte(raw))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(want))
}

func sanitizeUser(user *domain.User) *domain.User {
	if user == nil {
		return nil
	}

	cp := *user
	cp.PasswordHash = ""
	return &cp
}
