package sqlite

import (
	"path/filepath"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

func TestUserStoreRefusesInvalidRole(t *testing.T) {
	dir := t.TempDir()
	db, err := NewDB(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	store := NewUserStore(db)

	err = store.Create(&domain.User{
		Username:     "badrole",
		Email:        "badrole@example.com",
		PasswordHash: "x",
		Role:         domain.Role(99),
	})
	if err == nil {
		t.Fatal("expected invalid role error")
	}

	user := &domain.User{
		Username:     "ok",
		Email:        "ok@example.com",
		PasswordHash: "x",
		Role:         domain.RoleUser,
	}
	if err := store.Create(user); err != nil {
		t.Fatal(err)
	}

	user.Role = domain.Role(7)
	if err := store.Update(user); err == nil {
		t.Fatal("expected invalid role on update")
	}
}
