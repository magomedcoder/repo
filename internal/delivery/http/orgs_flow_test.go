package http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestOrganizationBasics(t *testing.T) {
	srv, _ := newTestServer(t)

	reg := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"ada","email":"ada@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}
	cookies := reg.Cookies

	created := doJSON(t, srv, http.MethodPost, "/api/orgs", `{"slug":"name","name":"Name","description":"demo"}`, cookies)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create org %d %s", created.StatusCode, created.Body)
	}

	if !strings.Contains(created.Body, `"slug":"name"`) || !strings.Contains(created.Body, `"role":"owner"`) {
		t.Fatalf("create body %s", created.Body)
	}

	dupUser := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"name","email":"name@example.com","password":"password1"}`, nil)
	if dupUser.StatusCode != http.StatusConflict {
		t.Fatalf("username taken by org %d %s", dupUser.StatusCode, dupUser.Body)
	}

	repo := doJSON(t, srv, http.MethodPost, "/api/repos",
		`{"name":"platform","organization":"name","private":false}`, cookies)
	if repo.StatusCode != http.StatusCreated {
		t.Fatalf("create org repo %d %s", repo.StatusCode, repo.Body)
	}

	if !strings.Contains(repo.Body, `"owner":"name"`) {
		t.Fatalf("repo owner %s", repo.Body)
	}

	got := doJSON(t, srv, http.MethodGet, "/api/repos/name/platform", "", nil)
	if got.StatusCode != http.StatusOK {
		t.Fatalf("get org repo %d %s", got.StatusCode, got.Body)
	}

	list := doJSON(t, srv, http.MethodGet, "/api/orgs/name/repos", "", cookies)
	if list.StatusCode != http.StatusOK || !strings.Contains(list.Body, `"name":"platform"`) {
		t.Fatalf("list org repos %d %s", list.StatusCode, list.Body)
	}

	reg2 := doJSON(t, srv, http.MethodPost, "/api/auth/register", `{"username":"bob","email":"bob@example.com","password":"password1"}`, nil)
	if reg2.StatusCode != http.StatusCreated {
		t.Fatalf("register bob %d %s", reg2.StatusCode, reg2.Body)
	}

	member := doJSON(t, srv, http.MethodPost, "/api/orgs/name/members", `{"username":"bob","role":"member"}`, cookies)
	if member.StatusCode != http.StatusCreated {
		t.Fatalf("add member %d %s", member.StatusCode, member.Body)
	}

	members := doJSON(t, srv, http.MethodGet, "/api/orgs/name/members", "", cookies)
	if members.StatusCode != http.StatusOK || !strings.Contains(members.Body, `"username":"bob"`) {
		t.Fatalf("list members %d %s", members.StatusCode, members.Body)
	}

	mine := doJSON(t, srv, http.MethodGet, "/api/orgs", "", cookies)
	var payload struct {
		Organizations []struct {
			Slug string `json:"slug"`
		} `json:"organizations"`
	}
	if err := json.Unmarshal([]byte(mine.Body), &payload); err != nil || len(payload.Organizations) == 0 {
		t.Fatalf("list mine %s", mine.Body)
	}
}
