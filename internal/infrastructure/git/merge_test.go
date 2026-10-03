package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/magomedcoder/repo/internal/domain"
)

func TestCompareAndMerge(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git required")
	}

	dir := t.TempDir()
	bare := filepath.Join(dir, "repo.git")
	r := NewRepository()
	if err := r.InitBare(bare, domain.BareInitOptions{
		DefaultBranch: "main",
	}); err != nil {
		t.Fatal(err)
	}

	seed := filepath.Join(dir, "seed")
	run(t, dir, "git", "clone", bare, seed)
	write(t, filepath.Join(seed, "a.txt"), "base\n")
	run(t, seed, "git", "add", ".")
	run(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "base")
	run(t, seed, "git", "push", "origin", "main")

	run(t, seed, "git", "checkout", "-b", "feature")
	write(t, filepath.Join(seed, "b.txt"), "feature\n")
	run(t, seed, "git", "add", ".")
	run(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "feat")
	run(t, seed, "git", "push", "origin", "feature")

	cmp, err := r.Compare(bare, "main", "feature")
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.CanFastForward {
		t.Fatal("expected ff")
	}

	if len(cmp.Commits) != 1 {
		t.Fatalf("commits %d", len(cmp.Commits))
	}

	if len(cmp.Files) == 0 {
		t.Fatal("expected file diff")
	}

	sha, err := r.Merge(bare, "main", "feature", "ff-only", "A", "a@b.c", "")
	if err != nil {
		t.Fatal(err)
	}

	if sha == "" {
		t.Fatal("empty sha")
	}

	run(t, seed, "git", "fetch")
	run(t, seed, "git", "checkout", "main")
	run(t, seed, "git", "pull")
	write(t, filepath.Join(seed, "main.txt"), "on main\n")
	run(t, seed, "git", "add", ".")
	run(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "main2")
	run(t, seed, "git", "push")

	run(t, seed, "git", "checkout", "feature")
	run(t, seed, "git", "pull", "origin", "feature")
	write(t, filepath.Join(seed, "feat2.txt"), "feat2\n")
	run(t, seed, "git", "add", ".")
	run(t, seed, "git", "-c", "user.email=a@b.c", "-c", "user.name=A", "commit", "-m", "feat2")
	run(t, seed, "git", "push", "-u", "origin", "feature")

	ff, err := r.CanFastForward(bare, "main", "feature")
	if err != nil {
		t.Fatal(err)
	}

	if ff {
		t.Fatal("expected not ff")
	}

	if _, err := r.Merge(bare, "main", "feature", "ff-only", "A", "a@b.c", ""); err == nil {
		t.Fatal("ff-only should fail")
	}

	sha, err = r.Merge(bare, "main", "feature", "merge", "A", "a@b.c", "Merge feature")
	if err != nil {
		t.Fatal(err)
	}

	if sha == "" {
		t.Fatal("empty merge sha")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
