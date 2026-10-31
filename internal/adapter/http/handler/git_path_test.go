package handler

import "testing"

func TestIsGitHTTPPathSuffixOnly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		ok   bool
	}{
		{"/work/demo.git/info/refs", true},
		{"/demo.git/git-upload-pack", true},
		{"/demo.git/git-receive-pack", true},
		{"/demo.git/info/refs-extra", false},
		{"/not-git/tree", false},
		{"/api/repos/x", false},
	}
	for _, tc := range cases {
		if got := IsGitHTTPPath(tc.path); got != tc.ok {
			t.Fatalf("%s: got %v want %v", tc.path, got, tc.ok)
		}
	}
}
