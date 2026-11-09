package handler

import (
	json "encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/httpx"
)

func TestStatusFor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind usecase.Kind
		want int
	}{
		{usecase.KindValidation, http.StatusBadRequest},
		{usecase.KindUnauthorized, http.StatusUnauthorized},
		{usecase.KindForbidden, http.StatusForbidden},
		{usecase.KindNotFound, http.StatusNotFound},
		{usecase.KindConflict, http.StatusConflict},
		{usecase.KindRateLimited, http.StatusTooManyRequests},
		{usecase.Kind(99), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := statusFor(tc.kind); got != tc.want {
			t.Fatalf("statusFor(%v)=%d want %d", tc.kind, got, tc.want)
		}
	}
}

func TestMapError(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := &httpx.Context{
		W: rec,
		R: req,
	}

	if err := mapError(c, usecase.ErrRepoNotFound); err != nil {
		t.Fatalf("mapError: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}

	var body httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	if body.Code != "repository_not_found" {
		t.Fatalf("code=%q", body.Code)
	}

	rec2 := httptest.NewRecorder()
	c2 := &httpx.Context{W: rec2, R: req}
	_ = mapError(c2, errString("boom"))
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d want 500", rec2.Code)
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
