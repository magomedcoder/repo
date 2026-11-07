package ssrf

import (
	"context"
	"net"
	"testing"
)

func TestHostBlockedLiteral(t *testing.T) {
	t.Parallel()

	blocked := []string{"localhost", "foo.localhost", "127.0.0.1", "10.0.0.1", "192.168.1.1", "169.254.169.254", "::1"}
	for _, h := range blocked {
		if !HostBlockedLiteral(h) {
			t.Fatalf("%s should be blocked", h)
		}
	}

	if HostBlockedLiteral("github.com") {
		t.Fatal("github.com must not be literal-blocked")
	}
}

func TestHostAllowedLookup(t *testing.T) {
	t.Parallel()

	lookup := func(_ context.Context, host string) ([]net.IP, error) {
		switch host {
		case "127.0.0.1.nip.io", "10-0-0-1.sslip.io":
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		case "git.example.com":
			return []net.IP{net.ParseIP("203.0.113.10")}, nil
		default:
			return nil, ErrBlockedHost
		}
	}

	if err := HostAllowed(context.Background(), "127.0.0.1.nip.io", lookup); err != ErrBlockedHost {
		t.Fatalf("nip.io: %v", err)
	}

	if err := HostAllowed(context.Background(), "10-0-0-1.sslip.io", lookup); err != ErrBlockedHost {
		t.Fatalf("sslip: %v", err)
	}

	if err := HostAllowed(context.Background(), "git.example.com", lookup); err != nil {
		t.Fatalf("public host: %v", err)
	}

	if err := HostAllowed(context.Background(), "127.0.0.1", lookup); err != ErrBlockedHost {
		t.Fatalf("literal loopback: %v", err)
	}
}

func TestSafeDialContextRejectsPrivate(t *testing.T) {
	t.Parallel()

	dial := SafeDialContext(nil, func(_ context.Context, host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("10.0.0.5")}, nil
	})

	_, err := dial(context.Background(), "tcp", "evil.example:443")
	if err != ErrBlockedHost {
		t.Fatalf("want ErrBlockedHost, got %v", err)
	}
}

func TestAssertGitRemoteAllowed(t *testing.T) {
	t.Parallel()

	lookup := func(_ context.Context, host string) ([]net.IP, error) {
		if host == "127.0.0.1.nip.io" {
			return []net.IP{net.ParseIP("127.0.0.1")}, nil
		}
		return []net.IP{net.ParseIP("203.0.113.9")}, nil
	}

	if err := AssertGitRemoteAllowed(context.Background(), "/tmp/local.git", lookup); err != nil {
		t.Fatalf("local path: %v", err)
	}

	if err := AssertGitRemoteAllowed(context.Background(), "https://git.example.com/a.git", lookup); err != nil {
		t.Fatalf("public: %v", err)
	}

	if err := AssertGitRemoteAllowed(context.Background(), "https://127.0.0.1.nip.io/x.git", lookup); err != ErrBlockedHost {
		t.Fatalf("nip.io: %v", err)
	}

	if err := AssertGitRemoteAllowed(context.Background(), "git@10.0.0.1:org/repo.git", lookup); err != ErrBlockedHost {
		t.Fatalf("scp private: %v", err)
	}
}
