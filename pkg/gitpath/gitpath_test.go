package gitpath

import "testing"

func TestParseRequireGitSuffix(t *testing.T) {
	t.Parallel()

	parts, err := Parse("foo.git", Options{
		MaxDepth:         1,
		RequireGitSuffix: true,
	})
	if err != nil || len(parts) != 1 || parts[0] != "foo.git" {
		t.Fatalf("got %v %v", parts, err)
	}

	if _, err := Parse("foo", Options{MaxDepth: 1, RequireGitSuffix: true}); err == nil {
		t.Fatal("expected error")
	}

	if _, err := Parse("../x.git", Options{MaxDepth: 1, RequireGitSuffix: true}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidSegment(t *testing.T) {
	t.Parallel()
	if !ValidSegment("a-b_c.1") {
		t.Fatal("expected valid")
	}

	if ValidSegment("..") || ValidSegment("a/b") {
		t.Fatal("expected invalid")
	}
}
