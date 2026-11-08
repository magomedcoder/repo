package usecase

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/argon2id"
	"magomedcoder.ru/repo/pkg/attempts"
)

const (
	minPasswordLen = 10
	maxPasswordLen = 1024

	loginIPMaxFails   = 30
	loginAcctMaxFails = 20
	gitTokenMaxFails  = 10
	stepUpMaxFails    = 5
	stepUpBaseLock    = 15 * time.Minute
	loginAcctMaxKeys  = 32768
)

var (
	ErrPasswordTooLong   = NewError(KindValidation, "password_too_long")
	ErrPasswordTooCommon = NewError(KindValidation, "password_too_common")
)

var commonPasswords = map[string]struct{}{
	"1234567890":       {},
	"0123456789":       {},
	"12345678910":      {},
	"1q2w3e4r5t":       {},
	"1qaz2wsx3e":       {},
	"qwertyuiop":       {},
	"qwerty1234":       {},
	"qwerty12345":      {},
	"password12":       {},
	"password123":      {},
	"password1234":     {},
	"passw0rd12":       {},
	"passwordpassword": {},
	"iloveyou12":       {},
	"abcdefghij":       {},
	"administrator":    {},
	"changeme123":      {},
	"letmein1234":      {},
	"welcome1234":      {},
	"0000000000":       {},
	"1111111111":       {},
	"aaaaaaaaaa":       {},
}

func validateNewPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	if len(password) > maxPasswordLen {
		return ErrPasswordTooLong
	}

	if len(password) < minPasswordLen {
		return ErrPasswordTooShort
	}

	if _, bad := commonPasswords[strings.ToLower(password)]; bad {
		return ErrPasswordTooCommon
	}

	return nil
}

func isHasherBusy(err error) bool {
	return errors.Is(err, argon2id.ErrBusy)
}

type LoginGuard struct {
	ip   attempts.Counter
	acct attempts.Counter
}

func NewLoginGuard() *LoginGuard {
	return &LoginGuard{
		ip: attempts.New(attempts.Config{
			MaxFails: loginIPMaxFails,
		}),
		acct: attempts.New(attempts.Config{
			MaxFails: loginAcctMaxFails,
			MaxKeys:  loginAcctMaxKeys,
		}),
	}
}

func (g *LoginGuard) Counters() []attempts.Counter {
	if g == nil {
		return nil
	}

	return []attempts.Counter{g.ip, g.acct}
}

func AccountKey(user *domain.User, login string) string {
	if user != nil && user.ID != 0 {
		return "login-uid\n" + strconv.FormatUint(uint64(user.ID), 10)
	}

	return "login-acct\n" + normalizeLogin(login)
}

func normalizeLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func loginIPKey(ip string) string {
	return "login-ip\n" + ip
}

func (g *LoginGuard) IPLocked(ip string) bool {
	return g != nil && ip != "" && g.ip.Locked(loginIPKey(ip))
}

func (g *LoginGuard) AccountLocked(acctKey string) bool {
	return g != nil && acctKey != "" && g.acct.Locked(acctKey)
}

func (g *LoginGuard) RetryAfter(ip, acctKey string) time.Duration {
	if g == nil {
		return 0
	}

	var d time.Duration
	if ip != "" {
		if ra := g.ip.RetryAfter(loginIPKey(ip)); ra > d {
			d = ra
		}
	}

	if acctKey != "" {
		if ra := g.acct.RetryAfter(acctKey); ra > d {
			d = ra
		}
	}

	return d
}

func (g *LoginGuard) Fail(ip, acctKey string) {
	if g == nil {
		return
	}

	if ip != "" {
		g.ip.Fail(loginIPKey(ip))
	}

	if acctKey != "" {
		g.acct.Fail(acctKey)
	}
}

func (g *LoginGuard) Success(acctKey string) {
	if g != nil && acctKey != "" {
		g.acct.Reset(acctKey)
	}
}

type PasswordGate struct {
	hasher   domain.PasswordHasher
	attempts attempts.Counter
	onLock   func(userID uint)
}

func NewPasswordGate(hasher domain.PasswordHasher) *PasswordGate {
	return &PasswordGate{
		hasher: hasher,
		attempts: attempts.New(attempts.Config{
			MaxFails: stepUpMaxFails,
			BaseLock: stepUpBaseLock,
		}),
	}
}

func (g *PasswordGate) SetOnLock(fn func(userID uint)) {
	if g != nil {
		g.onLock = fn
	}
}

func (g *PasswordGate) Attempts() attempts.Counter {
	if g == nil {
		return nil
	}

	return g.attempts
}

func (g *PasswordGate) RetryAfter(userID uint) time.Duration {
	if g == nil || g.attempts == nil || userID == 0 {
		return 0
	}

	return g.attempts.RetryAfter(stepUpKey(userID))
}

func stepUpKey(userID uint) string {
	return "stepup\n" + strconv.FormatUint(uint64(userID), 10)
}

func (g *PasswordGate) Verify(userID uint, hash, password string, wrong error) error {
	if g == nil || g.hasher == nil {
		return ErrPasswordRequired
	}

	if len(password) > maxPasswordLen {
		return ErrPasswordTooLong
	}

	key := stepUpKey(userID)
	if g.attempts.Locked(key) {
		return ErrTooManyAttempts
	}

	if err := g.hasher.Compare(hash, password); err != nil {
		if isHasherBusy(err) {
			return ErrBusy
		}

		locked := g.attempts.Fail(key)
		slog.Warn("step-up failed", "user_id", userID, "locked", locked)
		if locked {
			if g.onLock != nil {
				g.onLock(userID)
			}
			return ErrTooManyAttempts
		}

		return wrong
	}

	g.attempts.Reset(key)
	return nil
}
