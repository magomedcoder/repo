package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

func TestCodeIndexSearchAndCursor(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}

	repos := NewRepositoryRepo(db)
	idx := NewCodeIndex(db)

	owner := &domain.User{
		Username:     "alice",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := NewUserRepo(db).Create(owner); err != nil {
		t.Fatal(err)
	}

	pub := &domain.Repository{
		Name:      "pub",
		OwnerID:   owner.ID,
		RepoPath:  "pub.git",
		Path:      filepath.Join(dir, "pub.git"),
		IsPrivate: false,
	}
	priv := &domain.Repository{
		Name:      "priv",
		OwnerID:   owner.ID,
		RepoPath:  "priv.git",
		Path:      filepath.Join(dir, "priv.git"),
		IsPrivate: true,
	}
	if err := repos.Create(pub); err != nil {
		t.Fatal(err)
	}

	if err := repos.Create(priv); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := idx.ReplaceRepoIndex(ctx, pub.ID, "aaa", []domain.CodeIndexFile{
		{
			Path: "a.go",
			Body: "package main\nfunc Hello() {}\n",
		},
		{
			Path: "b.go",
			Body: "package main\nfunc World() {}\n",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := idx.ReplaceRepoIndex(ctx, priv.ID, "bbb", []domain.CodeIndexFile{
		{
			Path: "secret.go",
			Body: "func HelloSecret() {}\n",
		},
	}); err != nil {
		t.Fatal(err)
	}

	anon, err := idx.Search(ctx, 0, []string{"Hello"}, 0, "", 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(anon) != 1 || anon[0].Path != "a.go" {
		t.Fatalf("anon hits=%v", anon)
	}

	ownerHits, err := idx.Search(ctx, owner.ID, []string{"Hello"}, 0, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerHits) != 2 {
		t.Fatalf("owner hits=%d %#v", len(ownerHits), ownerHits)
	}

	page1, err := idx.Search(ctx, owner.ID, []string{"func"}, 0, "", 1)
	if err != nil || len(page1) != 1 {
		t.Fatalf("page1: %v %#v", err, page1)
	}

	page2, err := idx.Search(ctx, owner.ID, []string{"func"}, page1[0].RepoID, page1[0].Path, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(page2) < 1 {
		t.Fatal("expected more pages")
	}

	if page2[0].RepoID == page1[0].RepoID && page2[0].Path <= page1[0].Path {
		t.Fatalf("cursor did not advance: %#v -> %#v", page1[0], page2[0])
	}
}
