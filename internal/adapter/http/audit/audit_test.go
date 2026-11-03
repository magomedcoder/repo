package audit

import "testing"

func TestValidVisitPath(t *testing.T) {
	t.Parallel()

	for _, p := range []string{"/", "/ada/repo.git", "/ada/repo.git/tree/main/src"} {
		if !validVisitPath(p) {
			t.Errorf("%q rejected", p)
		}
	}

	for _, p := range []string{"", "x", "/api", "/api/me", "/assets/a.js", "/a/../b", "/a//b", "/a%2e", "/a?b", "/a\\b", "/a\nb"} {
		if validVisitPath(p) {
			t.Errorf("%q accepted", p)
		}
	}
}
