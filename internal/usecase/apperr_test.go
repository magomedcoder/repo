package usecase_test

import (
	"errors"
	"testing"

	"magomedcoder.ru/repo/internal/usecase"
)

func TestErrorIs(t *testing.T) {
	t.Parallel()
	err := usecase.ErrRepoNotFound
	if !errors.Is(err, usecase.ErrRepoNotFound) {
		t.Fatal("expected Is to match sentinel")
	}

	var app *usecase.Error
	if !errors.As(err, &app) || app.Kind != usecase.KindNotFound || app.Code != "repository_not_found" {
		t.Fatalf("As failed: %#v", app)
	}
}
