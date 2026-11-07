package gitpath

import (
	"errors"
	"strings"
)

var ErrInvalidPath = errors.New("invalid_path")

type Options struct {
	MaxDepth         int
	RequireGitSuffix bool
}

func Parse(raw string, opts Options) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "/") || strings.Contains(raw, "\\") {
		return nil, ErrInvalidPath
	}

	raw = strings.Trim(raw, "/")
	if raw == "" {
		return nil, ErrInvalidPath
	}

	parts := strings.Split(raw, "/")
	if opts.MaxDepth > 0 && len(parts) > opts.MaxDepth {
		return nil, ErrInvalidPath
	}

	for _, part := range parts {
		if !ValidSegment(part) {
			return nil, ErrInvalidPath
		}

		if opts.RequireGitSuffix && !strings.HasSuffix(part, ".git") {
			return nil, ErrInvalidPath
		}
	}

	return parts, nil
}

func ValidSegment(part string) bool {
	if part == "" || part == "." || part == ".." {
		return false
	}

	for _, r := range part {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}

	return true
}
