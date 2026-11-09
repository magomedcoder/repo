package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/magomedcoder/repo/pkg/httpware"
)

type RateLimiter = httpware.RateLimiter

func SetTrustedProxies(nets []*net.IPNet) {
	httpware.SetTrustedProxies(nets)
}

func NewRateLimiter(perMinute int, burst int) *RateLimiter {
	return httpware.NewRateLimiter(perMinute, burst)
}

func RateLimitIP(limiter *RateLimiter) func(http.Handler) http.Handler {
	return httpware.RateLimitIP(limiter)
}

func ClientIP(r *http.Request) string {
	return httpware.ClientIP(r)
}

func ClientIPKey(r *http.Request) string {
	return httpware.ClientIPKey(r)
}

func RateLimitWithContribute(apiLimiter, authLimiter, contributeLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIPKey(r)
			if isAuthMutation(r) {
				if authLimiter != nil && !authLimiter.Allow(ip) {
					httpware.WriteRateLimited(w)
					return
				}
			}

			if contributeLimiter != nil && r.Method == http.MethodPost && isContributePath(r.URL.Path) {
				if !contributeLimiter.Allow(ip) {
					httpware.WriteRateLimited(w)
					return
				}
			}

			if apiLimiter != nil && !apiLimiter.Allow(ip) {
				httpware.WriteRateLimited(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAuthMutation(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/auth/") {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}

	return r.URL.Path != "/api/auth/logout"
}

func RateLimitCodeSearch(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter != nil && r.URL.Path == "/api/search" && strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("type")), "code") {
				if !limiter.Allow(ClientIPKey(r)) {
					httpware.WriteRateLimited(w)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isContributePath(path string) bool {
	if !strings.HasPrefix(path, "/api/repos/") {
		return false
	}

	switch {
	case strings.HasSuffix(path, "/issues"), strings.HasSuffix(path, "/pulls"), strings.HasSuffix(path, "/labels"):
		return true
	case strings.Contains(path, "/comments"):
		return true
	default:
		return false
	}
}
