package urlredact

import (
	"net/url"
	"strings"
)

func URL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if !strings.Contains(raw, "://") {
		return cutQueryFragment(raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return redactUnparsable(raw)
	}

	u.User = nil
	u.RawQuery = ""
	u.ForceQuery = false
	u.Fragment = ""
	u.RawFragment = ""

	return u.String()
}

func cutQueryFragment(raw string) string {
	if i := strings.IndexAny(raw, "?#"); i >= 0 {
		return raw[:i]
	}

	return raw
}

func redactUnparsable(raw string) string {
	raw = cutQueryFragment(raw)
	scheme, rest, _ := strings.Cut(raw, "://")
	authority, path, hasPath := strings.Cut(rest, "/")
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		authority = authority[at+1:]
	}

	out := scheme + "://" + authority
	if hasPath {
		out += "/" + path
	}

	return out
}
