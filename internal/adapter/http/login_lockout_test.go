package http_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
)

func TestLoginSoftLockReturns429(t *testing.T) {
	env := newTestServer(t)
	user, err := env.Auth.CreateUser(t.Context(), usecase.RegisterInput{
		Username: "lockme",
		Email:    "lockme@example.com",
		Password: "right-password-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	acct := usecase.AccountKey(&domain.User{
		ID: user.ID,
	}, "lockme")
	for range 20 {
		env.Auth.LoginGuard().Fail("10.0.0.1", acct)
	}

	res := doJSON(t, env.Server, http.MethodPost, "/api/auth/login",
		`{"login":"lockme","password":"right-password-1"}`, nil)
	if res.StatusCode != http.StatusTooManyRequests || !strings.Contains(res.Body, "too_many_attempts") {
		t.Fatalf("want 429 too_many_attempts, got %d %s", res.StatusCode, res.Body)
	}

	if ra := res.Header.Get("Retry-After"); ra == "" {
		t.Fatal("expected Retry-After header")
	} else if n, err := strconv.Atoi(ra); err != nil || n < 1 {
		t.Fatalf("Retry-After=%q", ra)
	}
}
