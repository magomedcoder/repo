package middleware

import (
	"context"
	"errors"
	"net/http"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/httpx"
)

type contextKey string

const (
	userContextKey contextKey = "user"
	SessionCookie             = "session"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.User, *domain.Session, error)
}

func UserFromContext(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)
	return user, ok && user != nil
}

func SessionToken(r *http.Request) string {
	c, err := r.Cookie(SessionCookie)
	if err != nil || c.Value == "" {
		return ""
	}

	return c.Value
}

func RequireAuth(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _, err := auth.Authenticate(r.Context(), SessionToken(r))
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

func OptionalAuth(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := SessionToken(r); token != "" {
				if user, _, err := auth.Authenticate(r.Context(), token); err == nil {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
