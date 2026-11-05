package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"magomedcoder.ru/repo/internal/domain"
)

func TestMirrorFetchOnBareWithoutOrigin(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	dst := filepath.Join(root, "dst.git")

	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}

	workRepo, err := gogit.PlainInit(work, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := workRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(work, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add("README"); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "t",
			Email: "t@example.com",
		},
	}); err != nil {
		t.Fatal(err)
	}
	_ = workRepo.Close()

	r := NewRepository(root)
	if err := r.InitBare(dst, domain.BareInitOptions{DefaultBranch: "main"}); err != nil {
		t.Fatal(err)
	}

	opened, err := gogit.PlainOpen(dst)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := opened.Remote("origin"); err == nil {
		t.Fatal("expected no origin on fresh bare")
	}
	_ = opened.Close()

	if err := r.MirrorFetch(t.Context(), dst, work, nil); err != nil {
		t.Fatalf("fetch without origin: %v", err)
	}

	opened, err = gogit.PlainOpen(dst)
	if err != nil {
		t.Fatal(err)
	}

	defer opened.Close()

	if _, err := opened.Reference(plumbing.NewBranchReferenceName("master"), true); err != nil {
		t.Fatalf("expected fetched branch: %v", err)
	}
}

func TestMirrorCloneLocal(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	dst := filepath.Join(root, "dst.git")

	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}

	workRepo, err := gogit.PlainInit(work, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := workRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(work, "README"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add("README"); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "t",
			Email: "t@example.com",
		},
	}); err != nil {
		t.Fatal(err)
	}
	_ = workRepo.Close()

	r := NewRepository(root)
	branch, err := r.MirrorClone(t.Context(), dst, work, nil)
	if err != nil {
		t.Fatal(err)
	}

	if branch != "master" && branch != "main" {
		t.Fatalf("detected branch=%q", branch)
	}

	opened, err := gogit.PlainOpen(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	if _, err := opened.Head(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(work, "MORE"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	workRepo2, err := gogit.PlainOpen(work)
	if err != nil {
		t.Fatal(err)
	}

	w2, err := workRepo2.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w2.Add("MORE"); err != nil {
		t.Fatal(err)
	}

	if _, err := w2.Commit("more", &gogit.CommitOptions{
		Author: &object.Signature{Name: "t", Email: "t@example.com"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = workRepo2.Close()

	if err := r.MirrorFetch(t.Context(), dst, work, nil); err != nil {
		t.Fatal(err)
	}

	pushDst := filepath.Join(root, "push.git")
	if _, err := gogit.PlainInit(pushDst, true); err != nil {
		t.Fatal(err)
	}

	if err := r.MirrorPush(t.Context(), dst, pushDst, nil); err != nil {
		t.Fatal(err)
	}

	pushed, err := gogit.PlainOpen(pushDst)
	if err != nil {
		t.Fatal(err)
	}
	defer pushed.Close()

	if _, err := pushed.Head(); err != nil {
		t.Fatal(err)
	}
}
