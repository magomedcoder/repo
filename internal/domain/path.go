package domain

import (
	"errors"
	"strings"
)

const MaxPathDepth = 8

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

	for i, part := range parts {
		if !validSegment(part) {
			return nil, ErrInvalidPath
		}

		if i == len(parts)-1 && !strings.HasSuffix(part, ".git") {
			return nil, ErrInvalidPath
		}

		if i < len(parts)-1 && strings.HasSuffix(part, ".git") {
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

func IsZeroHash(hash string) bool {
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

func CheckRefUpdate(protect bool, defaultBranch, refname, oldRev, newRev string, ancestor bool) error {
	if !protect {
		return nil
	}

	if defaultBranch == "" {
		defaultBranch = "main"
	}

	if refname != "refs/heads/"+defaultBranch {
		return nil
	}

	if IsZeroHash(newRev) {
		return ErrBranchProtected
	}

	if IsZeroHash(oldRev) {
		return nil
	}

	if !ancestor {
		return ErrBranchProtected
	}

	return nil
}
