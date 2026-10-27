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
		ID uint `json:"id"`
	}
	if err := json.Unmarshal([]byte(folderRes.Body), &folder); err != nil || folder.ID == 0 {
		t.Fatalf("folder body %s", folderRes.Body)
	}

	inFolder := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"api","folder_id":`+itoa(folder.ID)+`,"private":false}`, cookies)
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

func itoa(v uint) string {
	return strings.TrimSpace(strings.ReplaceAll(jsonNumber(v), " ", ""))
}

func jsonNumber(v uint) string {
	b, _ := json.Marshal(v)
	return string(b)
}
