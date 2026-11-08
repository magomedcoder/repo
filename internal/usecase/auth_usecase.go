package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
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
	defaultSessionAbsoluteTTL = 30 * 24 * time.Hour
	defaultSessionIdleTTL     = 14 * 24 * time.Hour
	minUsernameLen            = 3
	maxUsernameLen            = 39
	maxSessionsPerUser        = 20
)

const dummyPasswordHash = "argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0c2FsdA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

var ErrRegistrationFailed = NewError(KindConflict, "registration_failed")
var ErrEmailUnavailable = NewError(KindConflict, "email_unavailable")

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
	IPKey     string
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
	users              domain.UserStore
	sessions           domain.SessionStore
	accessTokens       domain.AccessTokenStore
	hasher             domain.PasswordHasher
	tokens             domain.TokenGenerator
	guard              *LoginGuard
	stepUp             *PasswordGate
	secret             string
	allowRegistration  bool
	sessionAbsoluteTTL time.Duration
	sessionIdleTTL     time.Duration
	now                func() time.Time
}

func NewAuthUseCase(
	users domain.UserStore,
	sessions domain.SessionStore,
	accessTokens domain.AccessTokenStore,
	hasher domain.PasswordHasher,
	tokens domain.TokenGenerator,
) *AuthUseCase {
	return &AuthUseCase{
		users:              users,
		sessions:           sessions,
		accessTokens:       accessTokens,
		sessionAbsoluteTTL: defaultSessionAbsoluteTTL,
		sessionIdleTTL:     defaultSessionIdleTTL,
		hasher:             hasher,
		tokens:             tokens,
		guard:              NewLoginGuard(),
		stepUp:             NewPasswordGate(hasher),
		now:                time.Now,
	}
}

func (uc *AuthUseCase) StartAttemptsSweep(ctx context.Context, every time.Duration) {
	counters := append(uc.guard.Counters(), uc.stepUp.Attempts())
	attempts.StartBackgroundSweep(ctx, every, counters...)
}

func (uc *AuthUseCase) LoginRetryAfter(ctx context.Context, login, ipKey string) time.Duration {
	if uc == nil || uc.guard == nil {
		return 0
	}

	login = normalizeLogin(login)
	var user *domain.User
	if login != "" {
		var err error
		if strings.Contains(login, "@") {
			user, err = uc.users.FindByEmail(ctx, login)
		} else {
			user, err = uc.users.FindByUsername(ctx, login)
		}
		if err != nil {
			user = nil
		}
	}

	return uc.guard.RetryAfter(ipKey, AccountKey(user, login))
}

func (uc *AuthUseCase) LoginGuard() *LoginGuard {
	if uc == nil {
		return nil
	}

	return uc.guard
}

func (uc *AuthUseCase) PasswordGate() *PasswordGate {
	if uc == nil {
		return nil
	}

	return uc.stepUp
}

func (uc *AuthUseCase) StepUpRetryAfter(userID uint) time.Duration {
	if uc == nil || uc.stepUp == nil {
		return 0
	}

	return uc.stepUp.RetryAfter(userID)
}

func (uc *AuthUseCase) SetSessionSecret(secret string) {
	uc.secret = secret
}

func (uc *AuthUseCase) SetSessionTTL(absolute, idle time.Duration) {
	if uc == nil {
		return
	}

	if absolute > 0 {
		uc.sessionAbsoluteTTL = absolute
	}

	if idle > 0 {
		uc.sessionIdleTTL = idle
	}
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
	password := strings.TrimSpace(in.Password)

	if err := validateUsername(username); err != nil {
		return nil, err
	}

	if email == "" {
		return nil, ErrEmailRequired
	}

	if !emailPattern.MatchString(email) {
		return nil, ErrInvalidEmail
	}

	if err := validateNewPassword(password); err != nil {
		return nil, err
	}

	if IsReservedSlug(username) {
		return nil, ErrReservedSlug
	}

	role := in.Role
	if !role.Valid() {
		return nil, ErrInvalidRole
	}

	taken, err := uc.users.ExistsByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("check username: %w", err)
	}

	if !taken {
		taken, err = uc.users.ExistsByEmail(email)
		if err != nil {
			return nil, fmt.Errorf("check email: %w", err)
		}

		if taken {
			uc.burnHash(password)
			return nil, ErrEmailTaken
		}
	} else {
		uc.burnHash(password)
		return nil, ErrUsernameTaken
	}

	hash, err := uc.hasher.Hash(password)
	if err != nil {
		if isHasherBusy(err) {
			return nil, ErrBusy
		}

		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	}
	if err := uc.users.Create(user); err != nil {
		if isUniqueViolation(err) {
			if usernameConflict(err) {
				return nil, ErrUsernameTaken
			}

			return nil, ErrEmailTaken
		}

		return nil, fmt.Errorf("create user: %w", err)
	}

	return sanitizeUser(user), nil
}

func usernameConflict(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "username")
}

func (uc *AuthUseCase) burnHash(password string) {
	_, _ = uc.hasher.Hash(password)
}

func (uc *AuthUseCase) Login(ctx context.Context, in LoginInput) (*AuthOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	login := normalizeLogin(in.Login)
	in.Password = strings.TrimSpace(in.Password)

	ipKey := in.IPKey
	if ipKey == "" {
		ipKey = in.IP
	}

	if uc.guard.IPLocked(ipKey) {
		return nil, ErrTooManyAttempts
	}

	if login == "" {
		return nil, ErrInvalidCredentials
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
		user = nil
	}

	acctKey := AccountKey(user, login)
	if uc.guard.AccountLocked(acctKey) {
		return nil, ErrTooManyAttempts
	}

	if in.Password == "" || len(in.Password) > maxPasswordLen {
		return nil, ErrInvalidCredentials
	}

	if user == nil {
		if err := uc.hasher.Compare(dummyPasswordHash, in.Password); isHasherBusy(err) {
			return nil, ErrBusy
		}

		uc.guard.Fail(ipKey, acctKey)
		return nil, ErrInvalidCredentials
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		if isHasherBusy(err) {
			return nil, ErrBusy
		}

		uc.guard.Fail(ipKey, acctKey)
		return nil, ErrInvalidCredentials
	}

	uc.guard.Success(acctKey)

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

func (uc *AuthUseCase) ChangePassword(ctx context.Context, in ChangePasswordInput) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if in.UserID == 0 {
		return ErrUnauthorized
	}

	in.Current = strings.TrimSpace(in.Current)
	in.Next = strings.TrimSpace(in.Next)
	if in.Current == "" {
		return ErrPasswordRequired
	}

	if len(in.Current) > maxPasswordLen {
		return ErrPasswordTooLong
	}

	if err := validateNewPassword(in.Next); err != nil {
		return err
	}

	user, err := uc.users.FindByID(ctx, in.UserID)
	if err != nil {
		return ErrUnauthorized
	}

	if err := uc.stepUp.Verify(user.ID, user.PasswordHash, in.Current, ErrCurrentPasswordWrong); err != nil {
		return err
	}

	hash, err := uc.hasher.Hash(in.Next)
	if err != nil {
		if isHasherBusy(err) {
			return ErrBusy
		}

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

	if !session.CreatedAt.IsZero() && now.After(session.CreatedAt.Add(uc.sessionAbsoluteTTL)) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	lastSeen := session.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = session.CreatedAt
	}

	if !lastSeen.IsZero() && now.After(lastSeen.Add(uc.sessionIdleTTL)) {
		_ = uc.sessions.DeleteByToken(token)
		return nil, nil, ErrSessionExpired
	}

	absoluteEnd := session.CreatedAt.Add(uc.sessionAbsoluteTTL)
	idleEnd := now.Add(uc.sessionIdleTTL)
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
		ExpiresAt:  now.Add(uc.sessionIdleTTL),
		LastSeenAt: now,
		CreatedAt:  now,
	}
	if err := uc.sessions.Create(session); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	uc.pruneSessions(userID, session.ID)

	return token, nil
}

func (uc *AuthUseCase) pruneSessions(userID, keepID uint) {
	list, err := uc.sessions.ListByUserID(userID)
	if err != nil || len(list) <= maxSessionsPerUser {
		return
	}

	sort.SliceStable(list, func(i, j int) bool {
		return sessionActivity(list[i]).Before(sessionActivity(list[j]))
	})

	excess := len(list) - maxSessionsPerUser
	for _, old := range list {
		if excess <= 0 {
			break
		}

		if old.ID == keepID {
			continue
		}

		if err := uc.sessions.DeleteByUserAndID(userID, old.ID); err == nil || errors.Is(err, domain.ErrNotFound) {
			excess--
		}
	}
}

func sessionActivity(s domain.Session) time.Time {
	if !s.LastSeenAt.IsZero() {
		return s.LastSeenAt
	}

	return s.CreatedAt
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

func (uc *AuthUseCase) RevokeOtherSessions(ctx context.Context, userID uint, keepToken, currentPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	if keepToken == "" {
		return ErrUnauthorized
	}

	if err := uc.requireStepUp(ctx, userID, currentPassword); err != nil {
		return err
	}

	return uc.sessions.DeleteByUserIDExcept(userID, keepToken)
}

func (uc *AuthUseCase) RevokeAllSessions(ctx context.Context, userID uint, currentPassword string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if userID == 0 {
		return ErrUnauthorized
	}

	if err := uc.requireStepUp(ctx, userID, currentPassword); err != nil {
		return err
	}

	return uc.sessions.DeleteByUserID(userID)
}

func (uc *AuthUseCase) requireStepUp(ctx context.Context, userID uint, password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return ErrPasswordRequired
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return ErrUnauthorized
	}

	return uc.stepUp.Verify(user.ID, user.PasswordHash, password, ErrCurrentPasswordWrong)
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
