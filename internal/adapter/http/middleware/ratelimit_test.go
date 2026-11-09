package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magomedcoder/repo/pkg/httpware"
)

func TestRateLimiterHardCap(t *testing.T) {
	t.Parallel()
	l := NewRateLimiter(60, 10)

	for i := range httpware.MaxBuckets * 2 {
		l.Allow(fmt.Sprintf("k%d", i))
	}

	if n := l.BucketCount(); n > httpware.MaxBuckets {
		t.Fatalf("buckets=%d exceeds cap %d", n, httpware.MaxBuckets)
	}
}

func TestClientIPKeyCollapsesIPv6(t *testing.T) {
	t.Parallel()
	a := httptest.NewRequest("GET", "/", nil)
	a.RemoteAddr = "[2001:db8:1:2::1]:1234"
	b := httptest.NewRequest("GET", "/", nil)
	b.RemoteAddr = "[2001:db8:1:2::ffff]:99"

	if ClientIPKey(a) != ClientIPKey(b) {
		t.Fatalf("same /64 must share key: %q vs %q", ClientIPKey(a), ClientIPKey(b))
	}
}

func TestAuthLimiterOnlyMutations(t *testing.T) {
	t.Parallel()

	auth := NewRateLimiter(1, 1)
	h := RateLimitWithContribute(nil, auth, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = "192.0.2.10:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		return rec.Code
	}

	for range 10 {
		if code := do(http.MethodGet, "/api/auth/me"); code != http.StatusOK {
			t.Fatalf("GET /api/auth/me must not be auth-limited, got %d", code)
		}

		if code := do(http.MethodGet, "/api/auth/csrf"); code != http.StatusOK {
			t.Fatalf("GET /api/auth/csrf must not be auth-limited, got %d", code)
		}
	}

	if code := do(http.MethodPost, "/api/auth/login"); code != http.StatusOK {
		t.Fatalf("first login: %d", code)
	}

	if code := do(http.MethodPost, "/api/auth/login"); code != http.StatusTooManyRequests {
		t.Fatalf("second login must be limited, got %d", code)
	}
}
