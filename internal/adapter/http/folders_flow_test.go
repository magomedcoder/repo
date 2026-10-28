package http_test

import (
	"encoding/json"
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
