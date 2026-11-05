package git

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
)

func TestFindReadmeMissingIsNotError(t *testing.T) {
	root := t.TempDir()
	r := NewRepository(root)
	work := filepath.Join(r.Root, "work")

	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}

	repo, err := gogit.PlainInit(work, false)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(work, "note.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := w.Add("note.txt"); err != nil {
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
	_ = repo.Close()

	branch, err := r.DetectHEADBranch(work)
	if err != nil || branch == "" {
		t.Fatalf("DetectHEADBranch: branch=%q err=%v", branch, err)
	}

	blob, err := r.FindReadme(work, branch)
	if err != nil {
		t.Fatalf("FindReadme: %v", err)
	}

	if blob != nil {
		t.Fatalf("expected nil blob, got %+v", blob)
	}
}

func TestFindReadmeEmptyRepoPropagates(t *testing.T) {
	root := t.TempDir()
	r := NewRepository(root)
	bare := filepath.Join(r.Root, "empty.git")
	if err := os.MkdirAll(r.Root, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := gogit.PlainInit(bare, true); err != nil {
		t.Fatal(err)
	}

	_, err := r.FindReadme(bare, "main")
	if err == nil {
		t.Fatal("expected error for empty repository")
	}

	if !errors.Is(err, ErrEmptyRepository) && !errors.Is(err, ErrRefNotFound) {
		t.Fatalf("want empty/ref error, got %v", err)
	}
}
