package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsGitHTTPPathSuffixOnly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		ok   bool
	}{
		{"/demo.git/info/refs", true},
		{"/demo.git/git-upload-pack", true},
		{"/demo.git/git-receive-pack", true},
		{"/demo.git/info/refs-extra", false},
		{"/not-git/tree", false},
		{"/api/repos/x", false},
	}
	for _, tc := range cases {
		if got := IsGitHTTPPath(tc.path); got != tc.ok {
			t.Fatalf("%s: got %v want %v", tc.path, got, tc.ok)
		}
	}
}

func TestGitPackBodyGzipClose(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte("pack-data"))
	_ = zw.Close()

	r := httptest.NewRequest(http.MethodPost, "/x.git/git-upload-pack", &buf)
	r.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()

	body, err := gitPackBody(w, r, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	got, err := io.ReadAll(body)
	if err != nil || string(got) != "pack-data" {
		t.Fatalf("read: %q %v", got, err)
	}

	if err := body.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
