package middleware

import (
	"net"
	"net/http"
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

func RateLimit(apiLimiter, authLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				if authLimiter != nil && !authLimiter.Allow(ip) {
					w.Header().Set("Retry-After", "60")
					http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
					return
				}
			}

			if apiLimiter != nil && !apiLimiter.Allow(ip) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate_limited"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
