package spa

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestSPAFallbackAndCache(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<html>ok</html>"),
		},
		"assets/app.js": &fstest.MapFile{
			Data: []byte("console.log(1)"),
		},
		"assets/app.css": &fstest.MapFile{
			Data: []byte("body{}"),
		},
	}
	h := FileServer(fsys, Options{})

	t.Run("spa route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/repos/ada/hello", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}

		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("cache %q", rec.Header().Get("Cache-Control"))
		}

		if got := rec.Body.String(); got != "<html>ok</html>" {
			t.Fatalf("body %q", got)
		}
	})

	t.Run("asset immutable", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}

		if rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
			t.Fatalf("cache %q", rec.Header().Get("Cache-Control"))
		}
	})

	t.Run("gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/assets/app.css", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Header().Get("Content-Encoding") != "gzip" {
			t.Fatalf("encoding %q", rec.Header().Get("Content-Encoding"))
		}
	})
}

func TestSub(t *testing.T) {
	root := fstest.MapFS{
		"dist/index.html": &fstest.MapFile{
			Data: []byte("ui"),
		},
	}

	h, err := Sub(root, "dist", Options{})
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Body.String() != "ui" {
		t.Fatalf("body %q", rec.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := FileServer(fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("x"),
		},
	}, Options{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", rec.Code)
	}
}

var _ fs.FS = fstest.MapFS{}
