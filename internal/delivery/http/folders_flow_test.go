package http_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFoldersTreeMoveAndCreateRepo(t *testing.T) {
	srv, _ := newTestServer(t)

	reg := doJSON(t, srv, http.MethodPost, "/api/auth/register",`{"username":"utest","email":"utest@example.com","password":"password1"}`, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register %d %s", reg.StatusCode, reg.Body)
	}
	cookies := reg.Cookies

	parent := doJSON(t, srv, http.MethodPost, "/api/folders", `{"name":"work"}`, cookies)
	if parent.StatusCode != http.StatusCreated {
		t.Fatalf("create parent %d %s", parent.StatusCode, parent.Body)
	}

	var parentFolder struct {
		ID uint `json:"id"`
	}

	if err := json.Unmarshal([]byte(parent.Body), &parentFolder); err != nil || parentFolder.ID == 0 {
		t.Fatalf("parent body %s", parent.Body)
	}

	child := doJSON(t, srv, http.MethodPost, "/api/folders", `{"name":"backend","parent_id":`+itoa(parentFolder.ID)+`}`, cookies)
	if child.StatusCode != http.StatusCreated {
		t.Fatalf("create child %d %s", child.StatusCode, child.Body)
	}

	var childFolder struct {
		ID   uint   `json:"id"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(child.Body), &childFolder); err != nil {
		t.Fatal(err)
	}

	tree := doJSON(t, srv, http.MethodGet, "/api/folders?tree=1", "", cookies)
	if tree.StatusCode != http.StatusOK || !strings.Contains(tree.Body, `"name":"work"`) || !strings.Contains(tree.Body, `"name":"backend"`) {
		t.Fatalf("tree %d %s", tree.StatusCode, tree.Body)
	}

	repo := doJSON(t, srv, http.MethodPost, "/api/repos", `{"name":"api","folder_id":`+itoa(childFolder.ID)+`,"private":false}`, cookies)
	if repo.StatusCode != http.StatusCreated {
		t.Fatalf("create repo %d %s", repo.StatusCode, repo.Body)
	}

	if !strings.Contains(repo.Body, `"name":"api"`) {
		t.Fatalf("repo body %s", repo.Body)
	}

	moved := doJSON(t, srv, http.MethodPost, "/api/folders/"+itoa(childFolder.ID)+"/move", `{"parent_id":null}`, cookies)
	if moved.StatusCode != http.StatusOK {
		t.Fatalf("move folder %d %s", moved.StatusCode, moved.Body)
	}

	var movedFolder struct {
		Path string `json:"path"`
	}
	
	if err := json.Unmarshal([]byte(moved.Body), &movedFolder); err != nil || movedFolder.Path != "backend" {
		t.Fatalf("moved path %s", moved.Body)
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
