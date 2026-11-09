package usecase

import (
	"context"
	"net"
	"testing"

	"github.com/magomedcoder/repo/pkg/ssrf"
)

func testMirrorLookup(_ context.Context, host string) ([]net.IP, error) {
	switch host {
	case "github.com", "git.example.com":
		return []net.IP{net.ParseIP("203.0.113.10")}, nil
	case "127.0.0.1.nip.io", "10-0-0-1.sslip.io":
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	default:
		if ip := net.ParseIP(host); ip != nil {
			return []net.IP{ip}, nil
		}

		return []net.IP{net.ParseIP("203.0.113.20")}, nil
	}
}

func TestValidateMirrorURL(t *testing.T) {
	t.Parallel()

	p := mirrorURLPolicy{
		lookup: testMirrorLookup,
	}

	ok := []string{
		"https://github.com/magomedcoder/repo.git",
		"ssh://git@github.com/magomedcoder/repo.git",
		"git@github.com:org/repo.git",
		"deploy@git.example.com:team/app.git",
	}
	for _, u := range ok {
		if err := p.validate(u); err != nil {
			t.Fatalf("%s: %v", u, err)
		}
	}

	if err := p.validate("http://git.example.com/a/b"); err != ErrMirrorHTTPForbidden {
		t.Fatalf("http mirror: want ErrMirrorHTTPForbidden, got %v", err)
	}

	bad := []string{
		"",
		"ftp://example.com/repo.git",
		"file:///tmp/repo.git",
		"https://localhost/repo.git",
		"https://10.0.0.1/repo.git",
		"https://192.168.1.1/x.git",
		"https://user:pass@github.com/magomedcoder/repo.git",
		"ssh://git@127.0.0.1/repo.git",
		"git@10.0.0.5:org/repo.git",
		"https://127.0.0.1.nip.io/evil.git",
		"not-a-url",
		"git@host: spaced",
	}
	for _, u := range bad {
		if err := p.validate(u); err != ErrInvalidMirrorURL {
			t.Fatalf("%s: want ErrInvalidMirrorURL, got %v", u, err)
		}
	}

	for _, u := range []string{
		"http://127.0.0.1/repo.git",
		"http://169.254.169.254/latest",
		"http://10-0-0-1.sslip.io/x.git",
	} {
		if err := p.validate(u); err != ErrMirrorHTTPForbidden && err != ErrInvalidMirrorURL {
			t.Fatalf("%s: want http forbidden or invalid, got %v", u, err)
		}
	}
}

func TestParsePinnedHosts(t *testing.T) {
	t.Parallel()

	p := mirrorURLPolicy{}

	got, err := p.parsePinnedHosts("git.example.com=203.0.113.7, Other.Example=198.51.100.1")
	if err != nil || got["git.example.com"] != "203.0.113.7" || got["other.example"] != "198.51.100.1" {
		t.Fatalf("got=%v err=%v", got, err)
	}

	if _, err := p.parsePinnedHosts("git.example.com=10.0.0.1"); err != ErrInvalidMirrorURL {
		t.Fatalf("private pin must be rejected: %v", err)
	}

	if _, err := p.parsePinnedHosts("nonsense"); err != ErrInvalidMirrorOption {
		t.Fatalf("bad pin: %v", err)
	}
}

func TestHostAllowedUsesSSRF(t *testing.T) {
	t.Parallel()

	if err := ssrf.HostAllowed(context.Background(), "127.0.0.1.nip.io", testMirrorLookup); err != ssrf.ErrBlockedHost {
		t.Fatalf("got %v", err)
	}
}
