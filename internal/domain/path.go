package domain

import (
	"errors"
	"strings"

	"github.com/magomedcoder/repo/pkg/gitpath"
)

const MaxPathDepth = 1

var (
	ErrInvalidPath     = gitpath.ErrInvalidPath
	ErrBranchProtected = errors.New("branch_protected")
)

func ParseRepoPath(raw string) (segments []string, err error) {
	return gitpath.Parse(raw, gitpath.Options{
		MaxDepth:         MaxPathDepth,
		RequireGitSuffix: true,
	})
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
