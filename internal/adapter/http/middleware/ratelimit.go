package middleware

import (
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

type rateBucket struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*rateBucket
	rate      float64
	burst     float64
	now       func() time.Time
	lastSweep time.Time
}

var trustedProxies []*net.IPNet

func SetTrustedProxies(nets []*net.IPNet) {
	trustedProxies = nets
}

func NewRateLimiter(perMinute int, burst int) *RateLimiter {
	if perMinute <= 0 {
		perMinute = 120
	}

	if burst <= 0 {
		burst = max(perMinute/4, 5)
	}

	return &RateLimiter{
		buckets: make(map[string]*rateBucket),
		rate:    float64(perMinute) / 60.0,
		burst:   float64(burst),
		now:     time.Now,
	}
}

func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > 10*time.Minute {
		l.sweep(now)
		l.lastSweep = now
	}

	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &rateBucket{
			tokens: l.burst - 1,
			last:   now,
		}
		return true
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}

	b.last = now
	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func (l *RateLimiter) sweep(now time.Time) {
	for k, b := range l.buckets {
		if now.Sub(b.last) > 30*time.Minute {
			delete(l.buckets, k)
		}
	}
}

func writeRateLimited(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write([]byte(`{"error":"Too many requests","code":"rate_limited"}`))
}

func RateLimit(apiLimiter, authLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := ClientIP(r)
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				if authLimiter != nil && !authLimiter.Allow(ip) {
					writeRateLimited(w)
					return
				}
			}

			if apiLimiter != nil && !apiLimiter.Allow(ip) {
				writeRateLimited(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RateLimitIP(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter != nil && !limiter.Allow(ClientIP(r)) {
				writeRateLimited(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	remote := net.ParseIP(host)
	if remote == nil || !isTrustedProxy(remote) {
		return host
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return host
	}

	parts := strings.Split(xff, ",")
	for _, part := range slices.Backward(parts) {
		cand := strings.TrimSpace(part)
		if cand == "" {
			continue
		}
		ip := net.ParseIP(cand)
		if ip == nil {
			continue
		}
		if isTrustedProxy(ip) {
			continue
		}
		return cand
	}

	return host
}

func ClientIPKey(r *http.Request) string {
	return NormalizeIPKey(ClientIP(r))
}

func NormalizeIPKey(host string) string {
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}

	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}

	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

func isTrustedProxy(ip net.IP) bool {
	for _, network := range trustedProxies {
		if network != nil && network.Contains(ip) {
			return true
		}
	}

	return false
}
