package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/attempts"
)

var (
	ErrUsernameRequired     = NewError(KindValidation, "username_required")
	ErrEmailRequired        = NewError(KindValidation, "email_required")
	ErrPasswordRequired     = NewError(KindValidation, "password_required")
	ErrInvalidUsername      = NewError(KindValidation, "invalid_username")
	ErrInvalidEmail         = NewError(KindValidation, "invalid_email")
	ErrPasswordTooShort     = NewError(KindValidation, "password_too_short")
	ErrUsernameTaken        = NewError(KindConflict, "username_taken")
	ErrEmailTaken           = NewError(KindConflict, "email_taken")
	ErrInvalidCredentials   = NewError(KindUnauthorized, "invalid_credentials")
	ErrCurrentPasswordWrong = NewError(KindValidation, "current_password_wrong")
	ErrUnauthorized         = NewError(KindUnauthorized, "unauthorized")
	ErrSessionExpired       = NewError(KindUnauthorized, "session_expired")
	ErrSessionNotFound      = NewError(KindNotFound, "session_not_found")
	ErrRegistrationClosed   = NewError(KindForbidden, "registration_closed")
	ErrTooManyAttempts      = NewError(KindRateLimited, "too_many_attempts")
	ErrReservedSlug         = NewError(KindValidation, "reserved_slug")
	ErrInvalidRole          = NewError(KindValidation, "invalid_role")
	ErrLastAdmin            = NewError(KindConflict, "last_admin")
	ErrCannotModifySelf     = NewError(KindForbidden, "cannot_modify_self")
	ErrConfirmRequired      = NewError(KindValidation, "confirm_required")
	ErrConfirmMismatch      = NewError(KindValidation, "confirm_mismatch")
	ErrLastRepoAdmin        = NewError(KindForbidden, "last_repo_admin")
	ErrAdminRequired        = NewError(KindForbidden, "admin_required")
)

func IsReservedSlug(slug string) bool {
	return domain.IsReservedSlug(slug)
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
	userMaxFails       = 20
)

const dummyPasswordHash = "argon2id$v=19$m=65536,t=2,p=4$c2FsdHNhbHRzYWx0c2FsdA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

var ErrRegistrationFailed = NewError(KindConflict, "registration_failed")

type AuthOutput struct {
	User  *domain.User
	Token string
}

type RegisterInput struct {
	Username  string
	Email     string
	Password  string
	Role      domain.Role
	IP        string
	UserAgent string
}

type LoginInput struct {
	Login     string
	Password  string
	Key       string
	IP        string
	UserAgent string
}

type SessionItem struct {
	ID         uint      `json:"id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	Current    bool      `json:"current"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
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
	accessTokens      domain.AccessTokenStore
	hasher            domain.PasswordHasher
	tokens            domain.TokenGenerator
	attempts          attempts.Counter
	userAttempts      attempts.Counter
	secret            string
	allowRegistration bool
	now               func() time.Time
}

func NewAuthUseCase(
	users domain.UserStore,
	sessions domain.SessionStore,
	accessTokens domain.AccessTokenStore,
	hasher domain.PasswordHasher,
	tokens domain.TokenGenerator,
) *AuthUseCase {
	return &AuthUseCase{
		users:        users,
		sessions:     sessions,
		accessTokens: accessTokens,
		hasher:       hasher,
		tokens:       tokens,
		attempts:     attempts.New(),
		userAttempts: attempts.New(attempts.Config{MaxFails: userMaxFails}),
		now:          time.Now,
	}
}

func (uc *AuthUseCase) StartAttemptsSweep(ctx context.Context, every time.Duration) {
	attempts.StartBackgroundSweep(ctx, every, uc.attempts, uc.userAttempts)
}

func (uc *AuthUseCase) SetSessionSecret(secret string) {
	uc.secret = secret
}

func (uc *AuthUseCase) SetAllowRegistration(allow bool) {
	uc.allowRegistration = allow
}

func (uc *AuthUseCase) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if !uc.allowRegistration {
		return nil, ErrRegistrationClosed
	}

	in.Role = domain.RoleUser
	user, err := uc.CreateUser(ctx, in)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrEmailTaken) {
			return nil, ErrRegistrationFailed
		}
		return nil, err
	}

	token, err := uc.createSession(user.ID, in.IP, in.UserAgent)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: user, Token: token}, nil
}

func (uc *AuthUseCase) CreateUser(ctx context.Context, in RegisterInput) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	login := strings.ToLower(strings.TrimSpace(in.Login))
	if login == "" || in.Password == "" {
		return nil, ErrInvalidCredentials
	}

	userKey := "login-user\n" + login
	if uc.loginLocked(in.Key, userKey) {
		return nil, ErrTooManyAttempts
	}

	var (
		user *domain.User
		err  error
	)
	if strings.Contains(login, "@") {
		user, err = uc.users.FindByEmail(ctx, login)
	} else {
		user, err = uc.users.FindByUsername(ctx, login)
	}

	if err != nil {
		_ = uc.hasher.Compare(dummyPasswordHash, in.Password)
		uc.loginFailed(in.Key, userKey)
		return nil, ErrInvalidCredentials
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		uc.loginFailed(in.Key, userKey)
		return nil, ErrInvalidCredentials
	}

	if in.Key != "" && uc.attempts != nil {
		uc.attempts.Reset(in.Key)
	}

	if uc.userAttempts != nil {
		uc.userAttempts.Reset(userKey)
	}

	if uc.hasher.NeedsRehash(user.PasswordHash) {
		if hash, herr := uc.hasher.Hash(in.Password); herr == nil {
			user.PasswordHash = hash
			_ = uc.users.Update(user)
		}
	}

	token, err := uc.createSession(user.ID, in.IP, in.UserAgent)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{User: sanitizeUser(user), Token: token}, nil
}

func (uc *AuthUseCase) loginLocked(key, userKey string) bool {
	if key != "" && uc.attempts != nil && uc.attempts.Locked(key) {
		return true
	}

	return uc.userAttempts != nil && uc.userAttempts.Locked(userKey)
}

func (uc *AuthUseCase) loginFailed(key, userKey string) {
	if key != "" && uc.attempts != nil {
		uc.attempts.Fail(key)
	}

	if uc.userAttempts != nil {
		uc.userAttempts.Fail(userKey)
	}
}

func (uc *AuthUseCase) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if in.UserID == 0 {
		return ErrUnauthorized
	}

	if len(in.Next) < minPasswordLen {
		return ErrPasswordTooShort
	}

	user, err := uc.users.FindByID(ctx, in.UserID)
	if err != nil {
		return ErrUnauthorized
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Current); err != nil {
		return ErrCurrentPasswordWrong
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

	if uc.accessTokens != nil {
		if err := uc.accessTokens.DeleteByUserID(in.UserID); err != nil {
			return fmt.Errorf("revoke access tokens: %w", err)
		}
	}

	return nil
}

func (uc *AuthUseCase) Logout(ctx context.Context, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return nil
	}

	return uc.sessions.DeleteByToken(token)
}

func (uc *AuthUseCase) Me(ctx context.Context, token string) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	user, _, err := uc.Authenticate(ctx, token)
	return user, err
}

func (uc *AuthUseCase) Authenticate(ctx context.Context, token string) (*domain.User, *domain.Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if token == "" {
		return nil, nil, ErrUnauthorized
	}

	if !uc.validSessionToken(token) {
		return nil, nil, ErrUnauthorized
	}

	session, err := uc.sessions.FindByToken(ctx, token)
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

	user, err := uc.users.FindByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, ErrUnauthorized
	}

	return sanitizeUser(user), session, nil
}

func (uc *AuthUseCase) createSession(userID uint, ip, userAgent string) (string, error) {
	token, err := uc.tokens.Generate()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	token = uc.signSession(token)

	now := uc.now()
	session := &domain.Session{
		Token:      token,
		UserID:     userID,
		IP:         strings.TrimSpace(ip),
		UserAgent:  strings.TrimSpace(userAgent),
		ExpiresAt:  now.Add(sessionIdleTTL),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := uc.sessions.Create(session); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

func (uc *AuthUseCase) ListSessions(ctx context.Context, userID uint, currentToken string) ([]SessionItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if userID == 0 {
		return nil, ErrUnauthorized
	}

	sessions, err := uc.sessions.ListByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var currentID uint
	if currentToken != "" {
		if cur, err := uc.sessions.FindByToken(ctx, currentToken); err == nil && cur.UserID == userID {
			currentID = cur.ID
		}
	}

	out := make([]SessionItem, 0, len(sessions))
	now := uc.now()
	for _, s := range sessions {
		if !s.ExpiresAt.IsZero() && !s.ExpiresAt.After(now) {
			_ = uc.sessions.DeleteByUserAndID(userID, s.ID)
			continue
		}
		out = append(out, SessionItem{
			ID:         s.ID,
			IP:         s.IP,
			UserAgent:  s.UserAgent,
			Current:    s.ID == currentID && currentID != 0,
			CreatedAt:  s.CreatedAt,
			LastSeenAt: s.LastSeenAt,
			ExpiresAt:  s.ExpiresAt,
		})
	}
	return out, nil
}

func (uc *AuthUseCase) RevokeSession(ctx context.Context, userID, sessionID uint, currentToken string) (revokedCurrent bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	if userID == 0 || sessionID == 0 {
		return false, ErrUnauthorized
	}

	var currentID uint
	if currentToken != "" {
		if cur, ferr := uc.sessions.FindByToken(ctx, currentToken); ferr == nil && cur.UserID == userID {
			currentID = cur.ID
		}
	}

	if err := uc.sessions.DeleteByUserAndID(userID, sessionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return false, ErrSessionNotFound
		}
		return false, err
	}

	return currentID != 0 && sessionID == currentID, nil
}

func (uc *AuthUseCase) RevokeOtherSessions(ctx context.Context, userID uint, keepToken string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	if keepToken == "" {
		return ErrUnauthorized
	}

	return uc.sessions.DeleteByUserIDExcept(userID, keepToken)
}

func (uc *AuthUseCase) RevokeAllSessions(ctx context.Context, userID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	return uc.sessions.DeleteByUserID(userID)
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
