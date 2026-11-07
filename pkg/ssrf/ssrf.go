package ssrf

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrBlockedHost = fmt.Errorf("blocked host")
	scpLikeURL     = regexp.MustCompile(`^[A-Za-z0-9._-]+@(\[[0-9a-fA-F:]+\]|[A-Za-z0-9.-]+):[^\s]+$`)
)

type LookupIPFunc func(ctx context.Context, host string) ([]net.IP, error)

func DefaultLookupIP(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	out := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.IP)
	}

	return out, nil
}

func InternalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func CloudMetadataHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "metadata.google.internal" || host == "169.254.169.254" || host == "fd00:ec2::254"
}

func HostBlockedLiteral(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || CloudMetadataHost(host) {
		return true
	}

	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return InternalIP(ip)
	}

	return false
}

func NeedsHostCheck(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return false
	}

	if strings.Contains(raw, "://") {
		return true
	}

	return scpLikeURL.MatchString(raw)
}

func HostFromGitRemote(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrBlockedHost
	}

	if !strings.Contains(raw, "://") {
		if !scpLikeURL.MatchString(raw) {
			return "", ErrBlockedHost
		}

		after := raw[strings.Index(raw, "@")+1:]
		var host string
		if strings.HasPrefix(after, "[") {
			end := strings.Index(after, "]")
			if end <= 1 {
				return "", ErrBlockedHost
			}

			host = after[1:end]
		} else {
			host = after[:strings.Index(after, ":")]
		}

		return strings.ToLower(host), nil
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return "", ErrBlockedHost
	}

	switch strings.ToLower(u.Scheme) {
	case "https", "http", "ssh":
	default:
		return "", ErrBlockedHost
	}

	return strings.ToLower(u.Hostname()), nil
}

func AssertGitRemoteAllowed(ctx context.Context, raw string, lookup LookupIPFunc) error {
	if !NeedsHostCheck(raw) {
		return nil
	}

	host, err := HostFromGitRemote(raw)
	if err != nil {
		return err
	}

	return HostAllowed(ctx, host, lookup)
}

func HostAllowed(ctx context.Context, host string, lookup LookupIPFunc) error {
	host = strings.ToLower(strings.TrimSpace(host))
	if HostBlockedLiteral(host) {
		return ErrBlockedHost
	}

	if lookup == nil {
		lookup = DefaultLookupIP
	}

	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ips, err := lookup(ctx, host)
	if err != nil {
		return ErrBlockedHost
	}

	if len(ips) == 0 {
		return ErrBlockedHost
	}

	for _, ip := range ips {
		if InternalIP(ip) || CloudMetadataHost(ip.String()) {
			return ErrBlockedHost
		}
	}

	return nil
}

func SafeDialContext(pins map[string]string, lookup LookupIPFunc) func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second}
	if lookup == nil {
		lookup = DefaultLookupIP
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		hostLower := strings.ToLower(host)
		if ip, ok := pins[hostLower]; ok {
			if parsed := net.ParseIP(ip); parsed == nil || InternalIP(parsed) {
				return nil, ErrBlockedHost
			}

			return d.DialContext(ctx, network, net.JoinHostPort(ip, port))
		}

		if literal := net.ParseIP(host); literal != nil {
			if InternalIP(literal) {
				return nil, ErrBlockedHost
			}

			return d.DialContext(ctx, network, addr)
		}

		if err := HostAllowed(ctx, host, lookup); err != nil {
			return nil, err
		}

		ips, err := lookup(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, ErrBlockedHost
		}

		var last error
		for _, ip := range ips {
			if InternalIP(ip) {
				last = ErrBlockedHost
				continue
			}

			conn, derr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if derr == nil {
				return conn, nil
			}

			last = derr
		}

		if last == nil {
			last = ErrBlockedHost
		}

		return nil, last
	}
}
