package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const CSRFCookie = "repo_csrf"

func CSRF(secure bool, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ensureCSRFCookie(w, r, secure, secret)
			if needsCSRF(r) && !validCSRF(r, token, secret) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"CSRF token invalid","code":"csrf"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func BoundCSRFToken(secret, sessionToken string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

func SetCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookie,
		Value:    value,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   secure || IsHTTPS(r),
	})
}

func RotateAnonymousCSRF(w http.ResponseWriter, r *http.Request, secure bool) string {
	token := randomCSRFToken()
	SetCSRFCookie(w, r, secure, token)
	return token
}

func RotateSessionCSRF(w http.ResponseWriter, r *http.Request, secure bool, secret, sessionToken string) string {
	token := BoundCSRFToken(secret, sessionToken)
	SetCSRFCookie(w, r, secure, token)
	return token
}

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool, secret string) string {
	if session := SessionToken(r); session != "" && secret != "" {
		want := BoundCSRFToken(secret, session)
		if c, err := r.Cookie(CSRFCookie); err != nil || c.Value != want {
			SetCSRFCookie(w, r, secure, want)
		}
		return want
	}

	if c, err := r.Cookie(CSRFCookie); err == nil && c.Value != "" {
		return c.Value
	}

	token := randomCSRFToken()
	SetCSRFCookie(w, r, secure, token)
	return token
}

func randomCSRFToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func needsCSRF(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func validCSRF(r *http.Request, cookieToken string, secret string) bool {
	header := r.Header.Get("X-CSRF-Token")
	if header == "" || cookieToken == "" {
		return false
	}

	if subtle.ConstantTimeCompare([]byte(header), []byte(cookieToken)) != 1 {
		return false
	}

	session := SessionToken(r)
	if session == "" {
		return true
	}

	if secret == "" {
		return false
	}

	want := BoundCSRFToken(secret, session)
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(want)) == 1
}
