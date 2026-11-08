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

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127: // 100.64.0.0/10 CGNAT
			return true
		case ip4[0] == 0: // 0.0.0.0/8
			return true
		case ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0: // 192.0.0.0/24
			return true
		case ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19): // 198.18.0.0/15
			return true
		case ip4[0] >= 224:
			return true
		}
		return false
	}

	if ip[0] == 0xff { // multicast
		return true
	}

	if blockedEmbeddedIPv4(ip) {
		return true
	}

	if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x0d && ip[3] == 0xb8 { // 2001:db8::/32
		return true
	}

	return false
}

func blockedEmbeddedIPv4(ip net.IP) bool {
	ip = ip.To16()
	if ip == nil {
		return false
	}

	check := func(b []byte) bool {
		v4 := net.IPv4(b[0], b[1], b[2], b[3])
		return InternalIP(v4) || CloudMetadataHost(v4.String())
	}

	switch {
	case ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b: // 64:ff9b::/16
		if ip[4] == 0 && ip[5] == 0 && ip[6] == 0 && ip[7] == 0 && ip[8] == 0 && ip[9] == 0 && ip[10] == 0 && ip[11] == 0 {
			return check(ip[12:16]) // 64:ff9b::/96
		}

		return ip[4] == 0x00 && ip[5] == 0x01 // 64:ff9b:1::/48
	case ip[0] == 0x20 && ip[1] == 0x02: // 2002::/16 6to4
		return check(ip[2:6])
	case ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x00: // 2001::/32
		client := []byte{ip[12] ^ 0xff, ip[13] ^ 0xff, ip[14] ^ 0xff, ip[15] ^ 0xff}
		return check(ip[4:8]) || check(client)
	}

	for _, b := range ip[:12] {
		if b != 0 {
			return false
		}
	}

	return check(ip[12:16])
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
