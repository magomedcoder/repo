package httpware

import (
	"net"
	"net/http"
	"strings"
)

const DefaultCSP = "" +
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
	"form-action 'self'; " +
	"report-uri /api/csp-report; report-to csp-endpoint"

const DefaultCSPReportTo = `{"group":"csp-endpoint","max_age":10886400,"endpoints":[{"url":"/api/csp-report"}]}`

const DefaultHSTS = "max-age=31536000; includeSubDomains; preload"

type SecurityConfig struct {
	Secure  bool
	CSP     string
	HSTS    string
	Trusted func(net.IP) bool
}

func SecurityHeaders(cfg SecurityConfig) func(http.Handler) http.Handler {
	csp := cfg.CSP
	if csp == "" {
		csp = DefaultCSP
	}

	hsts := cfg.HSTS
	if hsts == "" {
		hsts = DefaultHSTS
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			policy := csp
			https := cfg.Secure || IsHTTPS(r, cfg.Trusted)
			if https {
				policy += "; upgrade-insecure-requests"
				h.Set("Strict-Transport-Security", hsts)
			}

			h.Set("Content-Security-Policy", policy)
			h.Set("Report-To", DefaultCSPReportTo)
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			h.Set("X-XSS-Protection", "0")
			if strings.HasPrefix(r.URL.Path, "/api/") {
				h.Set("Cache-Control", "private, no-store")
				h.Add("Vary", "Cookie")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func IsHTTPS(r *http.Request, trusted func(net.IP) bool) bool {
	if r.TLS != nil {
		return true
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	remote := net.ParseIP(host)
	if remote == nil || trusted == nil || !trusted(remote) {
		return false
	}

	return r.Header.Get("X-Forwarded-Proto") == "https"
}

func MaxBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}
