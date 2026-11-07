package domain

import (
	"errors"
	"strings"
)

const MaxPathDepth = 1

var (
	ErrInvalidPath     = errors.New("invalid_path")
	ErrBranchProtected = errors.New("branch_protected")
)

func ParseRepoPath(raw string) (segments []string, err error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "/") || strings.Contains(raw, "\\") {
		return nil, ErrInvalidPath
	}

	raw = strings.Trim(raw, "/")
	if raw == "" {
		return nil, ErrInvalidPath
	}

	parts := strings.Split(raw, "/")
	if len(parts) > MaxPathDepth {
		return nil, ErrInvalidPath
	}

	for _, part := range parts {
		if !validSegment(part) {
			return nil, ErrInvalidPath
		}

		if !strings.HasSuffix(part, ".git") {
			return nil, ErrInvalidPath
		}
	}

	return parts, nil
}

func validSegment(part string) bool {
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

func isZeroHash(hash string) bool {
	hash = strings.TrimSpace(hash)
	if len(hash) < 4 {
		return false
	}

	for _, r := range hash {
		if r != '0' {
			return false
		}
	}

	return true
}

func CheckRefUpdate(protect bool, refname, oldRev, newRev string, ancestor bool) error {
	if !protect {
		return nil
	}

	isHead := strings.HasPrefix(refname, "refs/heads/")
	isTag := strings.HasPrefix(refname, "refs/tags/")
	if !isHead && !isTag {
		return nil
	}

	if isZeroHash(newRev) {
		return ErrBranchProtected
	}

	if isZeroHash(oldRev) {
		return nil
	}

	if isTag {
		if !sameRev(oldRev, newRev) {
			return ErrBranchProtected
		}

		return nil
	}

	if !ancestor {
		return ErrBranchProtected
	}

	return nil
}

func sameRev(a, b string) bool {
	a = strings.TrimSpace(strings.ToLower(a))
	b = strings.TrimSpace(strings.ToLower(b))
	if a == "" || b == "" {
		return false
	}

	if len(a) > len(b) {
		return strings.HasPrefix(a, b)
	}

	return strings.HasPrefix(b, a) || a == b
}
