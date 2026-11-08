package gitref

import "testing"

func TestValidate(t *testing.T) {
	t.Parallel()

	ok := []string{"", "main", "feature/x-1", "v1.0.0", "refs/heads/main", "user@feat", "0123456789abcdef0123"}
	for _, ref := range ok {
		if err := Validate(ref); err != nil {
			t.Errorf("%q rejected: %v", ref, err)
		}
	}

	bad := []string{"main^{tree}", "a b", "HEAD~2", "a..b", "-x", "a:b", "x@{1}", "a\x00b", "a//b", "x.lock", "/x", "x/", "a*b", "a?b", "a[0]", "a\\b"}
	for _, ref := range bad {
		if err := Validate(ref); err == nil {
			t.Errorf("%q accepted", ref)
		}
	}
}
