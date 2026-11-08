package usecase

import (
	"context"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
)

func TestAdminCannotDeleteOrDemoteSelf(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})
	_ = users.Create(&domain.User{
		Username:     "boss",
		Email:        "b@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})

	if err := uc.Delete(context.Background(), 1, 1, "secret"); err != ErrCannotModifySelf {
		t.Fatalf("self delete: %v", err)
	}

	role := domain.RoleUser
	if _, err := uc.Update(context.Background(), 1, 1, AdminUpdateInput{
		Role:            &role,
		CurrentPassword: "secret",
	}); err != ErrCannotModifySelf {
		t.Fatalf("self demote: %v", err)
	}
}

func TestAdminDeleteRequiresPassword(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})
	_ = users.Create(&domain.User{
		Username:     "user",
		Email:        "u@x",
		PasswordHash: "h:x",
		Role:         domain.RoleUser,
	})

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})

	if err := uc.Delete(context.Background(), 1, 2, ""); err != ErrPasswordRequired {
		t.Fatalf("empty password: %v", err)
	}

	if err := uc.Delete(context.Background(), 1, 2, "wrong"); err != ErrCurrentPasswordWrong {
		t.Fatalf("wrong password: %v", err)
	}

	if err := uc.Delete(context.Background(), 1, 2, "secret"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestAdminCreateRequiresPassword(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})

	if _, err := uc.Create(context.Background(), 1, AdminCreateInput{
		Username: "newbie",
		Email:    "newbie@example.com",
		Password: "test-pass-1",
		Role:     domain.RoleUser,
	}); err != ErrPasswordRequired {
		t.Fatalf("empty step-up: %v", err)
	}

	if _, err := uc.Create(context.Background(), 1, AdminCreateInput{
		Username:        "newbie",
		Email:           "newbie@example.com",
		Password:        "test-pass-1",
		Role:            domain.RoleAdmin,
		CurrentPassword: "wrong",
	}); err != ErrCurrentPasswordWrong {
		t.Fatalf("wrong step-up: %v", err)
	}

	item, err := uc.Create(context.Background(), 1, AdminCreateInput{
		Username:        "newbie",
		Email:           "newbie@example.com",
		Password:        "test-pass-1",
		Role:            domain.RoleAdmin,
		CurrentPassword: "secret",
	})
	if err != nil || item == nil || item.Username != "newbie" || item.Role != domain.RoleAdmin {
		t.Fatalf("create: %+v %v", item, err)
	}
}

func TestAdminEmailChangeRequiresStepUp(t *testing.T) {
	t.Parallel()
	users := &memUsers{}
	_ = users.Create(&domain.User{
		Username:     "admin",
		Email:        "a@x",
		PasswordHash: "h:secret",
		Role:         domain.RoleAdmin,
	})
	_ = users.Create(&domain.User{
		Username:     "user",
		Email:        "u@example.com",
		PasswordHash: "h:x",
		Role:         domain.RoleUser,
	})

	sessions := &memSessions{}
	uc := NewAdminUseCase(users, sessions, nil, NewAuthUseCase(users, sessions, nil, memHasher{}, nil), memHasher{})

	email := "new@example.com"
	if _, err := uc.Update(context.Background(), 1, 2, AdminUpdateInput{Email: &email}); err != ErrPasswordRequired {
		t.Fatalf("email change without step-up: %v", err)
	}

	if _, err := uc.Update(context.Background(), 1, 2, AdminUpdateInput{
		Email:           &email,
		CurrentPassword: "wrong",
	}); err != ErrCurrentPasswordWrong {
		t.Fatalf("email change wrong step-up: %v", err)
	}

	got, err := uc.Update(context.Background(), 1, 2, AdminUpdateInput{
		Email:           &email,
		CurrentPassword: "secret",
	})
	if err != nil || got == nil || got.Email != email {
		t.Fatalf("email change: %+v %v", got, err)
	}
}
