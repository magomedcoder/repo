package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

type contextKey string

const (
	userContextKey    contextKey = "user"
	SessionCookie                = "session"
	HostSessionCookie            = "__Host-session"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.User, *domain.Session, error)
}

func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	return user, ok && user != nil
}

func SessionCookieName(r *http.Request, forceSecure bool) string {
	if forceSecure || IsHTTPS(r) {
		return HostSessionCookie
	}
	return SessionCookie
}

func SessionTokenFor(r *http.Request, forceSecure bool) string {
	c, err := r.Cookie(SessionCookieName(r, forceSecure))
	if err != nil {
		return ""
	}
	return c.Value
}

func RequireAuthSecure(auth Authenticator, forceSecure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _, err := auth.Authenticate(r.Context(), SessionTokenFor(r, forceSecure))
			if err != nil {
				msg := "unauthorized"
				if errors.Is(err, usecase.ErrSessionExpired) {
					msg = err.Error()
				}
				c := &httpx.Context{W: w, R: r}
				_ = c.ErrorPlain(http.StatusUnauthorized, msg)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuthSecure(auth Authenticator, forceSecure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := SessionTokenFor(r, forceSecure); token != "" {
				if user, _, err := auth.Authenticate(r.Context(), token); err == nil {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			c := &httpx.Context{W: w, R: r}
			if !ok || user == nil {
				_ = c.Error(http.StatusUnauthorized, "unauthorized")
				return
			}

			if !user.IsAdmin() {
				_ = c.Error(http.StatusForbidden, "admin_required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
