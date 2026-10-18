package domain

import "testing"

func TestParseRepoPath(t *testing.T) {
	parts, err := ParseRepoPath("notes/diary.git")
	if err != nil || len(parts) != 2 || parts[1] != "diary.git" {
		t.Fatalf("got %v %v", parts, err)
	}

	if SectionPrefix("notes/diary.git") != "notes" {
		t.Fatal("section")
	}

	for _, bad := range []string{"", "../x.git", "/abs.git", "a/../../b.git", "no-suffix", "a.git/b.git", "a/./b.git"} {
		if _, err := ParseRepoPath(bad); err != ErrInvalidPath {
			t.Fatalf("%q: %v", bad, err)
		}
	}
}

func TestCheckRefUpdate(t *testing.T) {
	if err := CheckRefUpdate(true, "main", "refs/heads/main", "aaa", "0000", false); err != ErrBranchProtected {
		t.Fatal("delete")
	}

	if err := CheckRefUpdate(true, "main", "refs/heads/main", "aaa", "bbb", false); err != ErrBranchProtected {
		t.Fatal("force")
	}

	if err := CheckRefUpdate(true, "main", "refs/heads/main", "aaa", "bbb", true); err != nil {
		t.Fatal(err)
	}

	if err := CheckRefUpdate(false, "main", "refs/heads/main", "aaa", "0000", false); err != nil {
		t.Fatal("flag off")
	}

	if err := CheckRefUpdate(true, "main", "refs/heads/topic", "aaa", "0000", false); err != nil {
		t.Fatal("other branch")
	}
}
