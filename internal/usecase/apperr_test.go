package usecase_test

import (
	"errors"
	"testing"

	"github.com/magomedcoder/repo/internal/usecase"
)

func TestErrorIs(t *testing.T) {
	t.Parallel()
	err := usecase.ErrRepoNotFound
	if !errors.Is(err, usecase.ErrRepoNotFound) {
		t.Fatal("expected Is to match sentinel")
	}

	app, ok := errors.AsType[*usecase.Error](err)
	if !ok || app.Kind != usecase.KindNotFound || app.Code != "repository_not_found" {
		t.Fatalf("AsType failed: %#v", app)
	}
}
