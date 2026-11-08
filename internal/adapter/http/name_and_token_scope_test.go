package http_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestRenameMovesPathAndHoldsOldName(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "renamer", "renamer@example.com", "test-pass-1")

	before := createRepo(t, env.Server, "before", false, cookies)

	res := doJSON(t, env.Server, http.MethodPatch, "/api/repos/"+before, `{"name":"after","current_password":"test-pass-1"}`, cookies)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename %d %s", res.StatusCode, res.Body)
	}

	if got := repoNameOf(t, res); got != "after" {
		t.Fatalf("rename must move path to new name: got %s", got)
	}

	if get := doJSON(t, env.Server, http.MethodGet, "/api/repos/after", "", cookies); get.StatusCode != http.StatusOK {
		t.Fatalf("new path must resolve: %d %s", get.StatusCode, get.Body)
	}

	if old := doJSON(t, env.Server, http.MethodGet, "/api/repos/before", "", cookies); old.StatusCode != http.StatusNotFound {
		t.Fatalf("old path must 404 after rename: %d %s", old.StatusCode, old.Body)
	}

	again := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"before","private":false}`, cookies)
	if again.StatusCode != http.StatusConflict || !strings.Contains(again.Body, "repository_already_exists") {
		t.Fatalf("old name must be held after rename: %d %s", again.StatusCode, again.Body)
	}

	del := doJSON(t, env.Server, http.MethodDelete, "/api/repos/after", `{"confirm":"after","current_password":"test-pass-1"}`, cookies)
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %s", del.StatusCode, del.Body)
	}

	other := loginUser(t, env, "renamer2", "renamer2@example.com", "test-pass-1")
	for _, name := range []string{"after", "before"} {
		for _, who := range []struct {
			label   string
			cookies []*http.Cookie
		}{
			{"owner", cookies},
			{"other", other},
		} {
			res := doJSON(t, env.Server, http.MethodPost, "/api/repos", fmt.Sprintf(`{"name":%q,"private":false}`, name), who.cookies)
			if res.StatusCode != http.StatusConflict {
				t.Fatalf("%s %s must stay held globally: %d %s", who.label, name, res.StatusCode, res.Body)
			}
		}
	}
}

func TestTokenRepoScope(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "scopeuser", "scopeuser@example.com", "test-pass-1")
	other := loginUser(t, env, "scopeother", "scopeother@example.com", "test-pass-1")

	allowed := createRepo(t, env.Server, "allowed", true, cookies)
	denied := createRepo(t, env.Server, "denied", true, cookies)
	foreign := doJSON(t, env.Server, http.MethodPost, "/api/repos", `{"name":"foreign","private":true}`, other)
	if foreign.StatusCode != http.StatusCreated {
		t.Fatalf("create foreign %d %s", foreign.StatusCode, foreign.Body)
	}

	var allowedItem, foreignItem struct {
		ID uint `json:"id"`
	}
	list := doJSON(t, env.Server, http.MethodGet, "/api/repos/"+allowed, "", cookies)
	if err := json.Unmarshal([]byte(list.Body), &allowedItem); err != nil || allowedItem.ID == 0 {
		t.Fatalf("allowed body %s err=%v", list.Body, err)
	}

	if err := json.Unmarshal([]byte(foreign.Body), &foreignItem); err != nil {
		t.Fatal(err)
	}

	bad := doJSON(t, env.Server, http.MethodPost, "/api/tokens", fmt.Sprintf(`{"name":"bad","scope":"git:write","repo_ids":[%d],"current_password":"test-pass-1"}`, foreignItem.ID), cookies)
	if bad.StatusCode != http.StatusBadRequest || !strings.Contains(bad.Body, "token_repos_invalid") {
		t.Fatalf("foreign repo id must be rejected: %d %s", bad.StatusCode, bad.Body)
	}

	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = fmt.Sprint(i + 1)
	}
	capRes := doJSON(t, env.Server, http.MethodPost, "/api/tokens", fmt.Sprintf(`{"name":"cap","repo_ids":[%s],"current_password":"test-pass-1"}`, strings.Join(tooMany, ",")), cookies)
	if capRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("repo_ids cap: %d %s", capRes.StatusCode, capRes.Body)
	}

	scoped := doJSON(t, env.Server, http.MethodPost, "/api/tokens", fmt.Sprintf(`{"name":"scoped","scope":"git:write","repo_ids":[%d],"current_password":"test-pass-1"}`, allowedItem.ID), cookies)
	if scoped.StatusCode != http.StatusCreated {
		t.Fatalf("create scoped token %d %s", scoped.StatusCode, scoped.Body)
	}

	var scopedOut struct {
		Token   string `json:"token"`
		RepoIDs []uint `json:"repo_ids"`
	}
	if err := json.Unmarshal([]byte(scoped.Body), &scopedOut); err != nil || scopedOut.Token == "" || len(scopedOut.RepoIDs) != 1 || scopedOut.RepoIDs[0] != allowedItem.ID {
		t.Fatalf("scoped body %s err=%v", scoped.Body, err)
	}

	open := doJSON(t, env.Server, http.MethodPost, "/api/tokens", `{"name":"open","scope":"git:write","current_password":"test-pass-1"}`, cookies)
	if open.StatusCode != http.StatusCreated || !strings.Contains(open.Body, `"repo_ids":[]`) {
		t.Fatalf("create open token %d %s", open.StatusCode, open.Body)
	}

	var openOut struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(open.Body), &openOut); err != nil {
		t.Fatal(err)
	}

	if st, _, body := gitInfoRefs(t, env, "/"+allowed+".git", "scopeuser", scopedOut.Token); st != http.StatusOK {
		t.Fatalf("scoped token on allowed repo: %d %s", st, body)
	}

	deniedStatus, _, deniedBody := gitInfoRefs(t, env, "/"+denied+".git", "scopeuser", scopedOut.Token)
	missStatus, _, missBody := gitInfoRefs(t, env, "/nosuchrepo1.git", "scopeuser", scopedOut.Token)
	if deniedStatus != http.StatusNotFound || deniedStatus != missStatus || deniedBody != missBody {
		t.Fatalf("scoped token outside allow-list must look like missing: %d %q vs %d %q", deniedStatus, deniedBody, missStatus, missBody)
	}

	if st, _, body := gitInfoRefs(t, env, "/"+denied+".git", "scopeuser", openOut.Token); st != http.StatusOK {
		t.Fatalf("unrestricted token: %d %s", st, body)
	}

	if st, _, body := gitInfoRefs(t, env, "/"+denied+".git", "scopeuser", "test-pass-1"); st != http.StatusOK {
		t.Fatalf("password auth must stay unrestricted: %d %s", st, body)
	}
}
