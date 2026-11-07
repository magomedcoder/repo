package middleware

import (
	"net/http"

	"magomedcoder.ru/repo/pkg/httpware"
)

func SecurityHeaders(secure bool) func(http.Handler) http.Handler {
	return httpware.SecurityHeaders(httpware.SecurityConfig{
		Secure:  secure,
		Trusted: httpware.DefaultTrust.Contains,
	})
}

func IsHTTPS(r *http.Request) bool {
	return httpware.IsHTTPS(r, httpware.DefaultTrust.Contains)
}
