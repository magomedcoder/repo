package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const CSRFCookie = "repo_csrf"

func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ensureCSRFCookie(w, r, secure)
			if needsCSRF(r) && !validCSRF(r, token) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"CSRF token invalid","code":"csrf"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool) string {
	if c, err := r.Cookie(CSRFCookie); err == nil && c.Value != "" {
		return c.Value
	}

	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := hex.EncodeToString(buf)
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookie,
		Value:    token,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})

	return token
}

func needsCSRF(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func validCSRF(r *http.Request, cookieToken string) bool {
	header := r.Header.Get("X-CSRF-Token")
	if header == "" || cookieToken == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(header), []byte(cookieToken)) == 1
}
