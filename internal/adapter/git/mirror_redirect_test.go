package git

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
)

func testSafeLookup(_ context.Context, _ string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("203.0.113.10")}, nil
}

func TestMirrorCheckRedirectRejectsHTTPSToHTTP(t *testing.T) {
	t.Parallel()

	via := []*http.Request{{URL: &url.URL{
		Scheme: "https",
		Host:   "git.example.com",
		Path:   "/a.git",
	}}}
	req := &http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "git.example.com",
			Path:   "/b.git",
		},
	}

	if err := mirrorCheckRedirect(req, via, testSafeLookup); err == nil {
		t.Fatal("expected https-http redirect to be rejected")
	}
}

func TestMirrorCheckRedirectAllowsSameHostHTTPS(t *testing.T) {
	t.Parallel()

	via := []*http.Request{{
		URL: &url.URL{
			Scheme: "https",
			Host:   "git.example.com",
			Path:   "/a.git",
		},
	}}
	req := &http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "git.example.com",
			Path:   "/b.git",
		},
	}

	if err := mirrorCheckRedirect(req, via, testSafeLookup); err != nil {
		t.Fatalf("same-host https redirect should be allowed: %v", err)
	}
}

func TestMirrorCheckRedirectRejectsCrossHostHTTPS(t *testing.T) {
	t.Parallel()

	via := []*http.Request{{
		URL: &url.URL{
			Scheme: "https",
			Host:   "git.example.com",
			Path:   "/a.git",
		},
	}}
	req := &http.Request{
		URL: &url.URL{
			Scheme: "https",
			Host:   "cdn.example.com",
			Path:   "/b.git",
		},
	}

	if err := mirrorCheckRedirect(req, via, testSafeLookup); err == nil {
		t.Fatal("cross-host https redirect must be rejected")
	}
}

func TestMirrorCheckRedirectRejectsPlainHTTP(t *testing.T) {
	t.Parallel()

	req := &http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "git.example.com",
			Path:   "/a.git",
		},
	}
	if err := mirrorCheckRedirect(req, nil, testSafeLookup); err == nil {
		t.Fatal("expected plain http redirect to be rejected")
	}
}
