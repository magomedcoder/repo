package httpware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

const DefaultCSRFCookie = "csrf"

type CSRFConfig struct {
	CookieName   string
	Secure       bool
	Secret       string
	SessionToken func(*http.Request) string
	ErrorBody    []byte
}

func (c CSRFConfig) cookieName() string {
	if c.CookieName != "" {
		return c.CookieName
	}

	return DefaultCSRFCookie
}

func (c CSRFConfig) errorBody() []byte {
	if len(c.ErrorBody) > 0 {
		return c.ErrorBody
	}

	return []byte(`{"error":"CSRF token invalid","code":"csrf"}`)
}

func CSRF(cfg CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/csp-report" {
				next.ServeHTTP(w, r)
				return
			}

			token := EnsureCSRFCookie(w, r, cfg)
			if needsCSRF(r) && !validCSRF(r, token, cfg) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write(cfg.errorBody())
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

func setCSRFCookie(w http.ResponseWriter, r *http.Request, cfg CSRFConfig, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cfg.cookieName(),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   cfg.Secure || IsHTTPS(r, DefaultTrust.Contains),
	})
}

func RotateAnonymousCSRF(w http.ResponseWriter, r *http.Request, cfg CSRFConfig) string {
	token := randomCSRFToken()
	setCSRFCookie(w, r, cfg, token)
	return token
}

func RotateSessionCSRF(w http.ResponseWriter, r *http.Request, cfg CSRFConfig, sessionToken string) string {
	token := BoundCSRFToken(cfg.Secret, sessionToken)
	setCSRFCookie(w, r, cfg, token)
	return token
}

func EnsureCSRFCookie(w http.ResponseWriter, r *http.Request, cfg CSRFConfig) string {
	session := ""
	if cfg.SessionToken != nil {
		session = cfg.SessionToken(r)
	}

	if session != "" && cfg.Secret != "" {
		want := BoundCSRFToken(cfg.Secret, session)
		if c, err := r.Cookie(cfg.cookieName()); err != nil || c.Value != want {
			setCSRFCookie(w, r, cfg, want)
		}
		return want
	}

	if c, err := r.Cookie(cfg.cookieName()); err == nil && c.Value != "" {
		return c.Value
	}

	token := randomCSRFToken()
	setCSRFCookie(w, r, cfg, token)
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

func validCSRF(r *http.Request, cookieToken string, cfg CSRFConfig) bool {
	header := r.Header.Get("X-CSRF-Token")
	if header == "" || cookieToken == "" {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(cookieToken)) != 1 {
		return false
	}

	session := ""
	if cfg.SessionToken != nil {
		session = cfg.SessionToken(r)
	}

	if session == "" {
		return true
	}

	if cfg.Secret == "" {
		return false
	}

	want := BoundCSRFToken(cfg.Secret, session)

	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(want)) == 1
}
