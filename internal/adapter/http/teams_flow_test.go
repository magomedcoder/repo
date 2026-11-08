package http_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestGuestCannotUseTeams(t *testing.T) {
	env := newTestServer(t)

	for _, path := range []string{"/api/teams", "/api/teams/devs", "/api/teams/devs/members", "/api/teams/devs/repos"} {
		res := doJSON(t, env.Server, http.MethodGet, path, "", nil)
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("guest %s %d %s", path, res.StatusCode, res.Body)
		}
	}
}

func TestOrgGroupAPIRemoved(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "nogroups", "nogroups@example.com", "test-pass-1")

	for _, path := range []string{"/api/groups", "/api/folders", "/api/admin/groups", "/api/admin/folders"} {
		res := doJSON(t, env.Server, http.MethodGet, path, "", cookies)
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("%s should be gone: %d %s", path, res.StatusCode, res.Body)
		}
	}

	res := doJSON(t, env.Server, http.MethodGet, "/api/repos/work/api", "", cookies)
	if res.StatusCode == http.StatusOK {
		t.Fatalf("nested repo path must not resolve: %d %s", res.StatusCode, res.Body)
	}
}

func TestTeamsGlobalFlow(t *testing.T) {
	env := newTestServer(t)
	srv := env.Server
	utest := loginUser(t, env, "utest", "utest@example.com", "test-pass-1")
	utest2 := loginUser(t, env, "utest2", "utest2@example.com", "test-pass-1")
	carol := loginUser(t, env, "carol", "carol@example.com", "test-pass-1")

	work := createRepo(t, srv, "work", true, utest)

	secret2 := createRepo(t, srv, "utest2s-secret", true, utest2)
	public2 := createRepo(t, srv, "utest2s-public", false, utest2)

	created := doJSON(t, srv, http.MethodPost, "/api/teams", `{"name":"Core Devs"}`, utest)
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create team %d %s", created.StatusCode, created.Body)
	}

	var team struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal([]byte(created.Body), &team); err != nil || team.Slug != "core-devs" {
		t.Fatalf("team body %s err=%v", created.Body, err)
	}

	if res := doJSON(t, srv, http.MethodPost, "/api/teams", `{"name":"core devs"}`, utest); res.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate team %d %s", res.StatusCode, res.Body)
	}

	list := doJSON(t, srv, http.MethodGet, "/api/teams", "", utest)
	if list.StatusCode != http.StatusOK {
		t.Fatalf("list teams %d %s", list.StatusCode, list.Body)
	}

	var listed struct {
		Teams []struct {
			Slug string `json:"slug"`
		} `json:"teams"`
	}
	if err := json.Unmarshal([]byte(list.Body), &listed); err != nil || len(listed.Teams) != 1 {
		t.Fatalf("list body %s err=%v", list.Body, err)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/teams", "", utest2); res.StatusCode != http.StatusOK || res.Body == list.Body {
		t.Fatalf("teams are per-owner: %d %s", res.StatusCode, res.Body)
	}

	base := "/api/teams/core-devs"
	if res := doJSON(t, srv, http.MethodPost, base+"/members", `{"username":"utest2"}`, utest); res.StatusCode != http.StatusCreated {
		t.Fatalf("add member %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/members", `{"username":"utest2"}`, utest); res.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate member %d %s", res.StatusCode, res.Body)
	}

	members := doJSON(t, srv, http.MethodGet, base+"/members", "", utest)
	if members.StatusCode != http.StatusOK {
		t.Fatalf("list members %d %s", members.StatusCode, members.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 before grant %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/repos", `{"repo":"`+work+`","role":"write"}`, utest); res.StatusCode != http.StatusOK {
		t.Fatalf("grant repo %d %s", res.StatusCode, res.Body)
	}

	grants := doJSON(t, srv, http.MethodGet, base+"/repos", "", utest)
	if grants.StatusCode != http.StatusOK {
		t.Fatalf("list grants %d %s", grants.StatusCode, grants.Body)
	}

	var gl struct {
		Repos []struct {
			RepoName string `json:"repo_name"`
			Role     string `json:"role"`
		} `json:"repos"`
	}
	if err := json.Unmarshal([]byte(grants.Body), &gl); err != nil || len(gl.Repos) != 1 || gl.Repos[0].RepoName != "work" || gl.Repos[0].Role != "write" {
		t.Fatalf("grants body %s err=%v", grants.Body, err)
	}

	got := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2)
	if got.StatusCode != http.StatusOK {
		t.Fatalf("utest2 after grant %d %s", got.StatusCode, got.Body)
	}

	var item struct {
		CanRead  bool `json:"can_read"`
		CanWrite bool `json:"can_write"`
		CanAdmin bool `json:"can_admin"`
	}
	if err := json.Unmarshal([]byte(got.Body), &item); err != nil || !item.CanRead || !item.CanWrite || item.CanAdmin {
		t.Fatalf("utest2 caps %s err=%v", got.Body, err)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work+"/issues", "", utest2); res.StatusCode != http.StatusOK {
		t.Fatalf("utest2 issues %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", carol); res.StatusCode != http.StatusNotFound {
		t.Fatalf("carol is not on the team %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/members", `{"username":"carol"}`, utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 manage utest team %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodDelete, base, "", utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 delete utest team %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/repos", `{"repo":"`+secret2+`","role":"read"}`, utest); res.StatusCode != http.StatusNotFound {
		t.Fatalf("grant foreign private repo %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodDelete, base+"/repos/"+secret2, "", utest); res.StatusCode != http.StatusNotFound {
		t.Fatalf("remove foreign private repo oracle %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodDelete, base+"/repos/missing-repo-xyz", "", utest); res.StatusCode != http.StatusNotFound {
		t.Fatalf("remove missing repo %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/repos", `{"repo":"`+public2+`","role":"read"}`, utest); res.StatusCode != http.StatusForbidden {
		t.Fatalf("grant foreign public repo %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/repos", `{"repo":"`+work+`","role":"admin"}`, utest); res.StatusCode != http.StatusOK {
		t.Fatalf("set grant %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPatch, base, `{"name":"Platform"}`, utest); res.StatusCode != http.StatusOK {
		t.Fatalf("rename team %d %s", res.StatusCode, res.Body)
	}

	base = "/api/teams/platform"
	if res := doJSON(t, srv, http.MethodDelete, base+"/members/utest2", "", utest); res.StatusCode != http.StatusNoContent {
		t.Fatalf("remove member %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 after removal %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/members", `{"username":"utest2"}`, utest); res.StatusCode != http.StatusCreated {
		t.Fatalf("re-add member %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2); res.StatusCode != http.StatusOK {
		t.Fatalf("utest2 re-added %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodDelete, base+"/repos/"+work, "", utest); res.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke grant %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 after revoke %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodPost, base+"/repos", `{"repo":"`+work+`","role":"read"}`, utest); res.StatusCode != http.StatusOK {
		t.Fatalf("regrant %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodDelete, base, "", utest); res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete team %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/"+work, "", utest2); res.StatusCode != http.StatusNotFound {
		t.Fatalf("utest2 after team delete %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, base, "", utest); res.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted team %d %s", res.StatusCode, res.Body)
	}
}

func TestFlatRepoCreateAndDeepPathRejected(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "flat", "flat@example.com", "test-pass-1")
	srv := env.Server

	if res := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"api","private":false}`, cookies); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("reserved name %d %s", res.StatusCode, res.Body)
	}

	notes := createRepo(t, srv, "notes", false, cookies)
	if notes != "notes" {
		t.Fatalf("path key must be name, got %q", notes)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/notes", "", cookies); res.StatusCode != http.StatusOK {
		t.Fatalf("get by name %d %s", res.StatusCode, res.Body)
	}

	if res := doJSON(t, srv, http.MethodGet, "/api/repos/flat/notes", "", cookies); res.StatusCode == http.StatusOK {
		t.Fatalf("nested path must fail, got %d %s", res.StatusCode, res.Body)
	}
}
