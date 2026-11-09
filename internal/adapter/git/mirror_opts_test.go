package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/magomedcoder/repo/internal/domain"
)

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func commitFile(t *testing.T, dir, name, body string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	gitCmd(t, dir, "add", name)
	gitCmd(t, dir, "commit", "-m", "add "+name)
}

func newRemote(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	dir := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	gitCmd(t, dir, "init", "-b", "main")
	commitFile(t, dir, "README", "hi\n")
	gitCmd(t, dir, "branch", "dev")
	gitCmd(t, dir, "tag", "v1")

	return dir
}

func localRefs(t *testing.T, path string) map[string]bool {
	t.Helper()

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	iter, err := repo.References()
	if err != nil {
		t.Fatal(err)
	}

	out := map[string]bool{}
	_ = iter.ForEach(func(r *plumbing.Reference) error {
		if r.Type() == plumbing.HashReference {
			out[r.Name().String()] = true
		}

		return nil
	})

	return out
}

func TestMirrorCheckAndDryRun(t *testing.T) {
	remote := newRemote(t)
	r := NewRepository(t.TempDir())

	info, err := r.MirrorCheck(t.Context(), remote, nil)
	if err != nil {
		t.Fatal(err)
	}

	if info.DefaultBranch != "main" || info.Head == "" || len(info.Refs) < 3 {
		t.Fatalf("info=%+v", info)
	}

	if _, err := r.MirrorCheck(t.Context(), filepath.Join(remote, "nope"), nil); err == nil {
		t.Fatal("check of missing remote must fail")
	}

	dst := filepath.Join(r.Root, "dst.git")
	if _, err := r.MirrorClone(t.Context(), dst, remote, nil); err != nil {
		t.Fatal(err)
	}

	_, would, err := r.MirrorDryRunFetch(t.Context(), dst, remote, nil)
	if err != nil || len(would) != 0 {
		t.Fatalf("in-sync dry run: %v %v", would, err)
	}

	commitFile(t, remote, "MORE", "x\n")
	gitCmd(t, remote, "branch", "feature")

	_, would, err = r.MirrorDryRunFetch(t.Context(), dst, remote, nil)
	if err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for _, w := range would {
		names[w.Name] = true
	}

	if !names["refs/heads/main"] || !names["refs/heads/feature"] {
		t.Fatalf("dry run must list changed/new refs, got %v", would)
	}

	if localRefs(t, dst)["refs/heads/feature"] {
		t.Fatal("dry run must not modify the repository")
	}
}

func TestMirrorSelectiveRefspecsAndPrune(t *testing.T) {
	remote := newRemote(t)
	r := NewRepository(t.TempDir())
	dst := filepath.Join(r.Root, "sel.git")

	keep := false
	opts := &domain.MirrorOptions{RefSpecs: []string{"refs/heads/main:refs/heads/main"}, Prune: &keep}

	if _, err := r.MirrorClone(t.Context(), dst, remote, opts); err != nil {
		t.Fatal(err)
	}

	refs := localRefs(t, dst)
	if !refs["refs/heads/main"] || refs["refs/heads/dev"] || refs["refs/tags/v1"] {
		t.Fatalf("selective refspec must only fetch main: %v", refs)
	}

	full := filepath.Join(r.Root, "full.git")
	if _, err := r.MirrorClone(t.Context(), full, remote, nil); err != nil {
		t.Fatal(err)
	}

	if !localRefs(t, full)["refs/heads/dev"] {
		t.Fatal("full mirror should have dev")
	}

	gitCmd(t, remote, "branch", "-D", "dev")

	if err := r.MirrorFetch(t.Context(), full, remote, &domain.MirrorOptions{Prune: &keep}); err != nil {
		t.Fatal(err)
	}

	if !localRefs(t, full)["refs/heads/dev"] {
		t.Fatal("prune=false must keep deleted remote branch")
	}

	doPrune := true
	if err := r.MirrorFetch(t.Context(), full, remote, &domain.MirrorOptions{Prune: &doPrune}); err != nil {
		t.Fatal(err)
	}

	if localRefs(t, full)["refs/heads/dev"] {
		t.Fatal("prune=true must remove the deleted branch")
	}
}

func TestMirrorPushDivergedRejectedWithoutForce(t *testing.T) {
	src := newRemote(t)
	r := NewRepository(t.TempDir())

	local := filepath.Join(r.Root, "local.git")
	if _, err := r.MirrorClone(t.Context(), local, src, nil); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(r.Root, "target.git")
	if _, err := gogit.PlainInit(target, true); err != nil {
		t.Fatal(err)
	}

	if err := r.MirrorPush(t.Context(), local, target, nil); err != nil {
		t.Fatal(err)
	}

	scratch := filepath.Join(t.TempDir(), "scratch")
	gitCmd(t, filepath.Dir(scratch), "clone", target, scratch)
	gitCmd(t, scratch, "checkout", "-B", "main", "origin/main")
	commitFile(t, scratch, "OTHER", "y\n")
	gitCmd(t, scratch, "push", "origin", "main")

	commitFile(t, src, "LOCAL", "z\n")

	if err := r.MirrorFetch(t.Context(), local, src, nil); err != nil {
		t.Fatal(err)
	}

	no := false
	err := r.MirrorPush(t.Context(), local, target, &domain.MirrorOptions{Force: &no})
	if !errors.Is(err, domain.ErrMirrorDiverged) {
		t.Fatalf("want ErrMirrorDiverged, got %v", err)
	}

	yes := true
	if err := r.MirrorPush(t.Context(), local, target, &domain.MirrorOptions{Force: &yes}); err != nil {
		t.Fatalf("force push: %v", err)
	}
}

func TestMirrorCloneMaxBytesCleansUp(t *testing.T) {
	remote := newRemote(t)
	r := NewRepository(t.TempDir())
	dst := filepath.Join(r.Root, "big.git")

	_, err := r.MirrorClone(t.Context(), dst, remote, &domain.MirrorOptions{MaxBytes: 1})
	if !errors.Is(err, domain.ErrMirrorTooLarge) {
		t.Fatalf("want ErrMirrorTooLarge, got %v", err)
	}

	if _, serr := os.Stat(dst); !os.IsNotExist(serr) {
		t.Fatal("failed clone must be removed from disk")
	}
}

func TestMirrorCloneFailureCleansUp(t *testing.T) {
	r := NewRepository(t.TempDir())
	dst := filepath.Join(r.Root, "missing.git")

	if _, err := r.MirrorClone(t.Context(), dst, filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Fatal("expected clone failure")
	}

	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("failed clone must not leave a directory behind")
	}
}

func TestMirrorLFSAndPartialUnavailable(t *testing.T) {
	remote := newRemote(t)
	r := NewRepository(t.TempDir())

	plain := filepath.Join(r.Root, "plain.git")
	if _, err := r.MirrorClone(t.Context(), plain, remote, &domain.MirrorOptions{}); err != nil {
		t.Fatalf("plain mirror must work: %v", err)
	}

	withLFS := filepath.Join(r.Root, "lfs.git")
	if _, err := r.MirrorClone(t.Context(), withLFS, remote, &domain.MirrorOptions{LFS: true}); !errors.Is(err, domain.ErrMirrorLFSUnavailable) {
		t.Fatalf("LFS must be fail-closed: %v", err)
	}

	if _, err := os.Stat(withLFS); !os.IsNotExist(err) {
		t.Fatal("failed LFS clone must be cleaned up")
	}

	withPartial := filepath.Join(r.Root, "partial.git")
	if _, err := r.MirrorClone(t.Context(), withPartial, remote, &domain.MirrorOptions{Partial: true}); !errors.Is(err, domain.ErrMirrorPartialUnavailable) {
		t.Fatalf("partial must be fail-closed: %v", err)
	}

	if _, err := os.Stat(withPartial); !os.IsNotExist(err) {
		t.Fatal("failed partial clone must be cleaned up")
	}
}

func TestMirrorVerifyPullAndDetectDrift(t *testing.T) {
	remote := newRemote(t)
	r := NewRepository(t.TempDir())
	dst := filepath.Join(r.Root, "dst.git")

	if _, err := r.MirrorClone(t.Context(), dst, remote, nil); err != nil {
		t.Fatal(err)
	}

	ok, err := r.MirrorVerify(t.Context(), dst, remote, domain.MirrorPull, nil)
	if err != nil || ok == nil || !ok.OK || ok.Checked == 0 {
		t.Fatalf("in-sync verify: %+v err=%v", ok, err)
	}

	commitFile(t, remote, "DRIFT", "x\n")

	bad, err := r.MirrorVerify(t.Context(), dst, remote, domain.MirrorPull, nil)
	if err != nil {
		t.Fatal(err)
	}
	if bad.OK || len(bad.Diffs) == 0 {
		t.Fatalf("expected drift diffs, got %+v", bad)
	}

	found := false
	for _, d := range bad.Diffs {
		if d.Name == "refs/heads/main" && d.Reason == "mismatch" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want main mismatch, got %+v", bad.Diffs)
	}
}

func TestMirrorVerifyPush(t *testing.T) {
	src := newRemote(t)
	r := NewRepository(t.TempDir())
	local := filepath.Join(r.Root, "local.git")
	if _, err := r.MirrorClone(t.Context(), local, src, nil); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(r.Root, "target.git")
	if _, err := gogit.PlainInit(target, true); err != nil {
		t.Fatal(err)
	}

	before, err := r.MirrorVerify(t.Context(), local, target, domain.MirrorPush, nil)
	if err != nil {
		t.Fatal(err)
	}
	if before.OK {
		t.Fatal("empty push target must fail verify")
	}

	if err := r.MirrorPush(t.Context(), local, target, nil); err != nil {
		t.Fatal(err)
	}

	after, err := r.MirrorVerify(t.Context(), local, target, domain.MirrorPush, nil)
	if err != nil || after == nil || !after.OK {
		t.Fatalf("after push verify: %+v err=%v", after, err)
	}
}
