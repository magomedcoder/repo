package middleware

import (
	"net/http"

	"magomedcoder.ru/repo/pkg/httpware"
)

const CSRFCookie = "repo_csrf"

func csrfConfig(secure bool, secret string) httpware.CSRFConfig {
	return httpware.CSRFConfig{
		CookieName:   CSRFCookie,
		Secure:       secure,
		Secret:       secret,
		SessionToken: SessionToken,
	}
}

func CSRF(secure bool, secret string) func(http.Handler) http.Handler {
	return httpware.CSRF(csrfConfig(secure, secret))
}

func RotateAnonymousCSRF(w http.ResponseWriter, r *http.Request, secure bool) string {
	return httpware.RotateAnonymousCSRF(w, r, csrfConfig(secure, ""))
}

func RotateSessionCSRF(w http.ResponseWriter, r *http.Request, secure bool, secret, sessionToken string) string {
	return httpware.RotateSessionCSRF(w, r, csrfConfig(secure, secret), sessionToken)
}

func EnsureCSRFCookie(w http.ResponseWriter, r *http.Request, secure bool, secret string) string {
	return httpware.EnsureCSRFCookie(w, r, csrfConfig(secure, secret))
}
