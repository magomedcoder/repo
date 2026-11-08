package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFCookieNameFollowsSecureFlag(t *testing.T) {
	t.Parallel()

	insecure := csrfConfig(false, "secret")
	if insecure.CookieName != CSRFCookie || insecure.Secure {
		t.Fatalf("insecure: name=%q secure=%v", insecure.CookieName, insecure.Secure)
	}

	secure := csrfConfig(true, "secret")
	if secure.CookieName != HostCSRFCookie || !secure.Secure {
		t.Fatalf("secure: name=%q secure=%v", secure.CookieName, secure.Secure)
	}
}

func TestCSRFCookieIssuedMatchesSecureFlag(t *testing.T) {
	t.Parallel()

	t.Run("insecure_never_host_prefix", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/csrf", nil)
		token := EnsureCSRFCookie(rec, req, false, "secret")
		if token == "" {
			t.Fatal("empty token")
		}

		got := rec.Result().Cookies()
		var found bool
		for _, c := range got {
			if strings.HasPrefix(c.Name, "__Host-") {
				t.Fatalf("must not issue __Host- cookie when cookie_secure=false: %+v", c)
			}
			if c.Name == CSRFCookie {
				found = true
				if c.Secure {
					t.Fatal("repo_csrf must not be Secure when cookie_secure=false")
				}
			}
		}
		if !found {
			t.Fatalf("expected %s cookie, got %#v", CSRFCookie, got)
		}
	})

	t.Run("secure_uses_host_prefix", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://example.com/api/auth/csrf", nil)
		token := EnsureCSRFCookie(rec, req, true, "secret")
		if token == "" {
			t.Fatal("empty token")
		}

		got := rec.Result().Cookies()
		var found bool
		for _, c := range got {
			if c.Name == HostCSRFCookie {
				found = true
				if !c.Secure {
					t.Fatal("__Host-repo_csrf must be Secure")
				}
			}

			if c.Name == CSRFCookie {
				t.Fatal("must not also set plain repo_csrf when secure")
			}
		}
		if !found {
			t.Fatalf("expected %s cookie, got %#v", HostCSRFCookie, got)
		}
	})
}
