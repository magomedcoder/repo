package middleware

import (
	"net"
	"net/http"
)

const contentSecurityPolicy = "" +
	"default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"worker-src 'self'; " +
	"manifest-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"frame-ancestors 'none'; " +
	"form-action 'self'"

const strictTransportSecurity = "max-age=31536000; includeSubDomains; preload"

func SecurityHeaders(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			csp := contentSecurityPolicy
			https := secure || IsHTTPS(r)
			if https {
				csp += "; upgrade-insecure-requests"
				h.Set("Strict-Transport-Security", strictTransportSecurity)
			}
			h.Set("Content-Security-Policy", csp)
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("X-XSS-Protection", "0")
			next.ServeHTTP(w, r)
		})
	}
}

func IsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	remote := net.ParseIP(host)
	if remote == nil || !isTrustedProxy(remote) {
		return false
	}

	return r.Header.Get("X-Forwarded-Proto") == "https"
}
