package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/attempts"
)

var (
	ErrUsernameRequired   = NewError(KindValidation, "username_required")
	ErrEmailRequired      = NewError(KindValidation, "email_required")
	ErrPasswordRequired   = NewError(KindValidation, "password_required")
	ErrInvalidUsername    = NewError(KindValidation, "invalid_username")
	ErrInvalidEmail       = NewError(KindValidation, "invalid_email")
	ErrPasswordTooShort   = NewError(KindValidation, "password_too_short")
	ErrUsernameTaken      = NewError(KindConflict, "username_taken")
	ErrEmailTaken         = NewError(KindConflict, "email_taken")
	ErrInvalidCredentials = NewError(KindUnauthorized, "invalid_credentials")
	ErrUnauthorized       = NewError(KindUnauthorized, "unauthorized")
	ErrSessionExpired     = NewError(KindUnauthorized, "session_expired")
	ErrRegistrationClosed = NewError(KindForbidden, "registration_closed")
	ErrTooManyAttempts    = NewError(KindRateLimited, "too_many_attempts")
	ErrReservedSlug       = NewError(KindValidation, "reserved_slug")
	ErrInvalidRole        = NewError(KindValidation, "invalid_role")
	ErrLastAdmin          = NewError(KindConflict, "last_admin")
	ErrAdminRequired      = NewError(KindForbidden, "admin_required")
)

var reservedSlugs = map[string]struct{}{
	"admin":    {},
	"api":      {},
	"data":     {},
	"login":    {},
	"register": {},
	"folders":  {},
	"settings": {},
	"search":   {},
	"assets":   {},
	"static":   {},
	"users":    {},
	"me":       {},
	"issues":   {},
	"pulls":    {},
	"commits":  {},
	"commit":   {},
	"tree":     {},
	"blob":     {},
	"refs":     {},
	"raw":      {},
}

func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

const (
	minPasswordLen     = 8
	sessionAbsoluteTTL = 30 * 24 * time.Hour
	sessionIdleTTL     = 14 * 24 * time.Hour
	minUsernameLen     = 3
	maxUsernameLen     = 39
)

type AuthOutput struct {
	User  *domain.User
	Token string
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
	Role     domain.Role
}

type LoginInput struct {
	Login    string
	Password string
	Key      string
}

type ChangePasswordInput struct {
	UserID    uint
	Current   string
	Next      string
	KeepToken string
}

type AuthUseCase struct {
	users             domain.UserStore
	sessions          domain.SessionStore
	hasher            domain.PasswordHasher
	tokens            domain.TokenGenerator
	attempts          attempts.Counter
	secret            string
	allowRegistration bool
	now               func() time.Time
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
		attempts: attempts.New(),
		now:      time.Now,
	}
}

func (uc *AuthUseCase) SetSessionSecret(secret string) {
	uc.secret = secret
}

func (uc *AuthUseCase) SetAllowRegistration(allow bool) {
	uc.allowRegistration = allow
}

func (uc *AuthUseCase) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	_ = ctx.Err()
	if !uc.allowRegistration {
		return nil, ErrRegistrationClosed
	}

	in.Role = domain.RoleUser
	user, err := uc.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}

	token, err := uc.createSession(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: user, Token: token}, nil
}

func (uc *AuthUseCase) CreateUser(ctx context.Context, in RegisterInput) (*domain.User, error) {
	_ = ctx.Err()
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

	if IsReservedSlug(username) {
		return nil, ErrReservedSlug
	}

	taken, err := uc.users.ExistsByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}
	if taken {
		return nil, ErrUsernameTaken
	}

	taken, err = uc.users.ExistsByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if taken {
		return nil, ErrEmailTaken
	}

	role := in.Role
	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	}
	if err := uc.users.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return sanitizeUser(user), nil
}

func (uc *AuthUseCase) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	_ = ctx.Err()
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
		if in.Key != "" && uc.attempts != nil {
			uc.attempts.Fail(in.Key)
		}
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

func (uc *AuthUseCase) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	_ = ctx.Err()
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
	if err := uc.users.Update(user); err != nil {
		return err
	}

	if err := uc.sessions.DeleteByUserIDExcept(in.UserID, in.KeepToken); err != nil {
		return fmt.Errorf("revoke sessions: %w", err)
	}

	return nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, token string) error {
	_ = ctx.Err()
	if token == "" {
		return nil
	}

	return uc.sessions.DeleteByToken(token)
}

func (uc *AuthUseCase) Me(ctx context.Context, token string) (*domain.User, error) {
	_ = ctx.Err()
	user, _, err := uc.Authenticate(ctx, token)
	return user, err
}

func (uc *AuthUseCase) Authenticate(ctx context.Context, token string) (*domain.User, *domain.Session, error) {
	_ = ctx.Err()
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

	now := uc.now()
	if !session.ExpiresAt.After(now) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	if !session.CreatedAt.IsZero() && now.After(session.CreatedAt.Add(sessionAbsoluteTTL)) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = session.CreatedAt
	}

	if !lastSeen.IsZero() && now.After(lastSeen.Add(sessionIdleTTL)) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	absoluteEnd := session.CreatedAt.Add(sessionAbsoluteTTL)
	idleEnd := now.Add(sessionIdleTTL)
	newExpiry := absoluteEnd
	if idleEnd.Before(absoluteEnd) {
		newExpiry = idleEnd
	}

	const touchMinInterval = time.Minute
	needsTouch := session.LastSeenAt.IsZero() || now.Sub(session.LastSeenAt) >= touchMinInterval || !session.ExpiresAt.Equal(newExpiry)
	if needsTouch {
		session.LastSeenAt = now
		session.ExpiresAt = newExpiry
		_ = uc.sessions.Touch(session)
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

	now := uc.now()
	session := &domain.Session{
		Token:      token,
		UserID:     userID,
		ExpiresAt:  now.Add(sessionIdleTTL),
		LastSeenAt: now,
		CreatedAt:  now,
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

	if strings.HasPrefix(username, "repo_") {
		return ErrReservedSlug
	}

	return nil
}

func (uc *AuthUseCase) signSession(token string) string {
	mac := hmac.New(sha256.New, []byte(uc.secret))
	_, _ = mac.Write([]byte(token))
	return token + "." + hex.EncodeToString(mac.Sum(nil))
}

func (uc *AuthUseCase) validSessionToken(token string) bool {
	if uc.secret == "" {
		return false
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
