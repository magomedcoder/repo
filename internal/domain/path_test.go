package domain

import "testing"

func TestParseRepoPath(t *testing.T) {
	t.Parallel()
	parts, err := ParseRepoPath("diary.git")
	if err != nil || len(parts) != 1 || parts[0] != "diary.git" {
		t.Fatalf("got %v %v", parts, err)
	}

	for _, bad := range []string{"", "../x.git", "/abs.git", "a/../../b.git", "no-suffix", "a.git/b.git", "a/./b.git", "a/b/c.git", "notes/diary.git"} {
		if _, err := ParseRepoPath(bad); err != ErrInvalidPath {
			t.Fatalf("%q: %v", bad, err)
		}
	}
}

func TestCheckRefUpdate(t *testing.T) {
	t.Parallel()
	if err := CheckRefUpdate(true, "refs/heads/main", "aaa", "0000", false); err != ErrBranchProtected {
		t.Fatal("delete")
	}

	if err := CheckRefUpdate(true, "refs/heads/main", "aaa", "bbb", false); err != ErrBranchProtected {
		t.Fatal("force")
	}

	if err := CheckRefUpdate(true, "refs/heads/main", "aaa", "bbb", true); err != nil {
		t.Fatal(err)
	}

	if err := CheckRefUpdate(false, "refs/heads/main", "aaa", "0000", false); err != nil {
		t.Fatal("flag off")
	}

	if err := CheckRefUpdate(true, "refs/heads/topic", "aaa", "0000", false); err != ErrBranchProtected {
		t.Fatal("other branch delete must be protected")
	}

	if err := CheckRefUpdate(true, "refs/heads/topic", "aaa", "bbb", false); err != ErrBranchProtected {
		t.Fatal("other branch force-push must be protected")
	}

	if err := CheckRefUpdate(true, "refs/tags/v1", "aaa", "0000", false); err != ErrBranchProtected {
		t.Fatal("tag delete must be protected")
	}

	if err := CheckRefUpdate(true, "refs/tags/v1", "aaa", "bbb", false); err != ErrBranchProtected {
		t.Fatal("tag force-update must be protected")
	}

	if err := CheckRefUpdate(true, "refs/tags/v1", "0000", "bbb", false); err != nil {
		t.Fatal("creating a new tag must be allowed")
	}

	if err := CheckRefUpdate(false, "refs/tags/v1", "aaa", "0000", false); err != nil {
		t.Fatal("protection off must allow tag delete")
	}
}
