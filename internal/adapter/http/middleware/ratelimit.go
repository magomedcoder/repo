package middleware

import (
	"net"
	"net/http"
	"strings"

	"magomedcoder.ru/repo/pkg/httpware"
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
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
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
