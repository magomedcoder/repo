package middleware

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterHardCap(t *testing.T) {
	t.Parallel()
	l := NewRateLimiter(60, 10)

	for i := range maxRateBuckets * 2 {
		l.Allow(fmt.Sprintf("k%d", i))
	}

	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	if n > maxRateBuckets {
		t.Fatalf("buckets=%d exceeds cap %d", n, maxRateBuckets)
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
