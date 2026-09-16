package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
)

type contextKey string

const (
	userContextKey contextKey = "user"
	SessionCookie             = "session"
)

type Authenticator interface {
	Authenticate(token string) (*domain.User, *domain.Session, error)
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
			user, _, err := auth.Authenticate(SessionToken(r))
			if err != nil {
				msg := "unauthorized"
				if errors.Is(err, usecase.ErrSessionExpired) {
					msg = err.Error()
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
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
				if user, _, err := auth.Authenticate(token); err == nil {
					ctx := context.WithValue(r.Context(), userContextKey, user)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
