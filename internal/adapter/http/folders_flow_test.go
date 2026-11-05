package http_test

import (
	json "encoding/json/v2"
	"net/http"
	"strings"
	"testing"
)

func TestFoldersCreateRepo(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "utest", "utest@example.com", "password1")
	srv := env.Server

	folderRes := doJSON(t, srv, http.MethodPost, "/api/folders", `{"name":"work"}`, cookies)
	if folderRes.StatusCode != http.StatusCreated {
		t.Fatalf("create folder %d %s", folderRes.StatusCode, folderRes.Body)
	}

	var folder struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(folderRes.Body), &folder); err != nil || folder.Path == "" {
		t.Fatalf("folder body %s", folderRes.Body)
	}

	byPath := doJSON(t, srv, http.MethodGet, "/api/folders/"+folder.Path, "", cookies)
	if byPath.StatusCode != http.StatusOK {
		t.Fatalf("get folder by path %d %s", byPath.StatusCode, byPath.Body)
	}

	inFolder := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"api","folder_path":"work","private":false}`, cookies)
	if inFolder.StatusCode != http.StatusCreated {
		t.Fatalf("create repo in folder %d %s", inFolder.StatusCode, inFolder.Body)
	}

	rootRepo := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"solo","private":false}`, cookies)
	if rootRepo.StatusCode != http.StatusCreated {
		t.Fatalf("create root repo %d %s", rootRepo.StatusCode, rootRepo.Body)
	}

	search := doJSON(t, srv, http.MethodGet, "/api/repos?q=api", "", cookies)
	if search.StatusCode != http.StatusOK || !strings.Contains(search.Body, `"name":"api"`) {
		t.Fatalf("search %d %s", search.StatusCode, search.Body)
	}

	got := doJSON(t, srv, http.MethodGet, "/api/repos/work/api", "", cookies)
	if got.StatusCode != http.StatusOK || !strings.Contains(got.Body, `"name":"api"`) {
		t.Fatalf("get folder repo %d %s", got.StatusCode, got.Body)
	}

	issues := doJSON(t, srv, http.MethodGet, "/api/repos/work/api/issues", "", cookies)
	if issues.StatusCode != http.StatusOK {
		t.Fatalf("folder repo issues %d %s", issues.StatusCode, issues.Body)
	}

	createdIssue := doJSON(t, srv, http.MethodPost, "/api/repos/work/api/issues", `{"title":"bug","body":"x"}`, cookies)
	if createdIssue.StatusCode != http.StatusCreated {
		t.Fatalf("create folder issue %d %s", createdIssue.StatusCode, createdIssue.Body)
	}

	issue := doJSON(t, srv, http.MethodGet, "/api/repos/work/api/issues/1", "", cookies)
	if issue.StatusCode != http.StatusOK || !strings.Contains(issue.Body, `"title":"bug"`) {
		t.Fatalf("folder issue detail %d %s", issue.StatusCode, issue.Body)
	}

	branches := doJSON(t, srv, http.MethodGet, "/api/repos/work/api/branches", "", cookies)
	if branches.StatusCode != http.StatusOK {
		t.Fatalf("folder repo branches %d %s", branches.StatusCode, branches.Body)
	}

	solo := doJSON(t, srv, http.MethodGet, "/api/repos/solo", "", cookies)
	if solo.StatusCode != http.StatusOK || !strings.Contains(solo.Body, `"name":"solo"`) {
		t.Fatalf("get root repo %d %s", solo.StatusCode, solo.Body)
	}

	rootIssue := doJSON(t, srv, http.MethodPost, "/api/repos/solo/issues", `{"title":"root","body":"y"}`, cookies)
	if rootIssue.StatusCode != http.StatusCreated {
		t.Fatalf("create root issue %d %s", rootIssue.StatusCode, rootIssue.Body)
	}

	rootDetail := doJSON(t, srv, http.MethodGet, "/api/repos/solo/issues/1", "", cookies)
	if rootDetail.StatusCode != http.StatusOK || !strings.Contains(rootDetail.Body, `"title":"root"`) {
		t.Fatalf("root issue detail via catch-all %d %s", rootDetail.StatusCode, rootDetail.Body)
	}
}

func TestFolderRenameAndDelete(t *testing.T) {
	env := newTestServer(t)
	cookies := loginUser(t, env, "frename", "frename@example.com", "password1")
	srv := env.Server

	create := doJSON(t, srv, http.MethodPost, "/api/folders", `{"name":"old-name"}`, cookies)
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create %d %s", create.StatusCode, create.Body)
	}

	renamed := doJSON(t, srv, http.MethodPatch, "/api/folders/old-name", `{"name":"new-name"}`, cookies)
	if renamed.StatusCode != http.StatusOK {
		t.Fatalf("rename %d %s", renamed.StatusCode, renamed.Body)
	}
	if !strings.Contains(renamed.Body, `"path":"new-name"`) {
		t.Fatalf("rename body %s", renamed.Body)
	}

	goneOld := doJSON(t, srv, http.MethodGet, "/api/folders/old-name", "", cookies)
	if goneOld.StatusCode != http.StatusNotFound {
		t.Fatalf("old path still exists %d", goneOld.StatusCode)
	}

	okNew := doJSON(t, srv, http.MethodGet, "/api/folders/new-name", "", cookies)
	if okNew.StatusCode != http.StatusOK {
		t.Fatalf("new path %d %s", okNew.StatusCode, okNew.Body)
	}

	del := doJSON(t, srv, http.MethodDelete, "/api/folders/new-name", "", cookies)
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete %d %s", del.StatusCode, del.Body)
	}
}

func TestGuestCannotListFolders(t *testing.T) {
	env := newTestServer(t)
	res := doJSON(t, env.Server, http.MethodGet, "/api/folders", "", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("guest folders %d %s", res.StatusCode, res.Body)
	}
}
