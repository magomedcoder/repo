package http_test

import (
	"net/http"
	"testing"
)

func TestChangePasswordHTTP(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "pwduser", "pwduser@example.com", "test-pass-1")

	res := doJSON(t, env.Server, http.MethodPost, "/api/auth/password", `{"current":"test-pass-1","next":"test-pass-2"}`, cookies)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("change password: %d %s", res.StatusCode, res.Body)
	}

	oldLogin := doJSON(t, env.Server, http.MethodPost, "/api/auth/login", `{"login":"pwduser","password":"test-pass-1"}`, nil)
	if oldLogin.StatusCode == http.StatusOK {
		t.Fatal("old password still accepted")
	}

	newLogin := doJSON(t, env.Server, http.MethodPost, "/api/auth/login", `{"login":"pwduser","password":"test-pass-2"}`, nil)
	if newLogin.StatusCode != http.StatusOK {
		t.Fatalf("new password login: %d %s", newLogin.StatusCode, newLogin.Body)
	}
}
