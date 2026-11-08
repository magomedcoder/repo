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

func TestInternalIPEmbeddedIPv4(t *testing.T) {
	t.Parallel()

	blocked := []string{
		"64:ff9b::7f00:1",           // NAT64 -> 127.0.0.1
		"64:ff9b::a9fe:a9fe",        // NAT64 -> 169.254.169.254
		"64:ff9b::0a00:0001",        // NAT64 -> 10.0.0.1
		"64:ff9b:1::1",              // local-use NAT64
		"2002:7f00:1::1",            // 6to4 -> 127.0.0.1
		"2002:a9fe:a9fe::1",         // 6to4 -> metadata
		"2001:0:7f00:1::",           // Teredo server 127.0.0.1
		"2001:0:808:808::80ff:fffe", // Teredo client 127.0.0.1 (obfuscated)
		"::7f00:1",                  // IPv4-compatible loopback-ish
	}
	for _, raw := range blocked {
		ip := net.ParseIP(raw)
		if ip == nil {
			t.Fatalf("bad test ip %q", raw)
		}

		if !InternalIP(ip) {
			t.Errorf("%s must be blocked", raw)
		}
	}

	allowed := []string{
		"64:ff9b::0808:0808",   // NAT64 -> 8.8.8.8
		"2002:0808:0808::1",    // 6to4 -> 8.8.8.8
		"2606:4700:4700::1111", // public v6
	}
	for _, raw := range allowed {
		if InternalIP(net.ParseIP(raw)) {
			t.Errorf("%s must be allowed", raw)
		}
	}
}
