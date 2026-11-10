package seo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInjectHTML(t *testing.T) {
	t.Parallel()
	raw := []byte(`<!DOCTYPE html><html lang="en"><head><title>Old</title></head><body></body></html>`)
	out := string(InjectHTML(raw, Meta{
		Title:       `Demo <repo>`,
		Description: `Hello "world"`,
		Canonical:   "https://git.example/demo",
		Robots:      "index,follow",
		SiteName:    "Repo",
		OGLocale:    "ru_RU",
		OGImage:     "https://git.example/og.png",
		JSONLD: []map[string]any{
			{"@type": "WebSite", "name": "Repo"},
		},
	}))

	for _, want := range []string{
		"<title>Demo &lt;repo&gt;</title>",
		`name="description" content="Hello &#34;world&#34;"`,
		`rel="canonical" href="https://git.example/demo"`,
		`property="og:locale" content="ru_RU"`,
		`application/ld+json`,
		`"@type":"WebSite"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRobots(t *testing.T) {
	t.Parallel()
	svc := &Service{
		SiteName:  "Repo",
		PublicURL: "https://git.example",
	}
	rec := httptest.NewRecorder()
	svc.Robots(rec, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
	body := rec.Body.String()
	if !strings.Contains(body, "Sitemap: https://git.example/sitemap.xml") {
		t.Fatalf("robots: %s", body)
	}

	if !strings.Contains(body, "Disallow: /cp") {
		t.Fatalf("robots missing disallow: %s", body)
	}
}

func TestManifestUsesSiteName(t *testing.T) {
	t.Parallel()
	svc := &Service{
		SiteName: "Nest Forge",
	}
	rec := httptest.NewRecorder()
	svc.Manifest(rec, httptest.NewRequest(http.MethodGet, "/site.webmanifest", nil))
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "manifest+json") {
		t.Fatalf("content-type=%q", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"name":"Nest Forge"`) && !strings.Contains(body, `"name": "Nest Forge"`) {
		t.Fatalf("manifest: %s", body)
	}

	if !strings.Contains(body, `"short_name":"Nest Forge"`) && !strings.Contains(body, `"short_name": "Nest Forge"`) {
		t.Fatalf("short_name: %s", body)
	}
}

func TestResolveSoft404(t *testing.T) {
	t.Parallel()
	svc := &Service{
		SiteName:  "Nest",
		PublicURL: "https://git.example",
		FindRepo: func(context.Context, string) (*RepoInfo, error) {
			return nil, errors.New("missing")
		},
	}
	req := httptest.NewRequest(http.MethodGet, "https://git.example/missing-repo", nil)
	meta := svc.Resolve(req)
	if meta.Status != http.StatusNotFound {
		t.Fatalf("status=%d", meta.Status)
	}

	if !strings.Contains(meta.Robots, "noindex") {
		t.Fatalf("robots=%q", meta.Robots)
	}
}

func TestResolvePrivateGuestIs404(t *testing.T) {
	t.Parallel()
	svc := &Service{
		SiteName:  "Nest",
		PublicURL: "https://git.example",
		FindRepo: func(context.Context, string) (*RepoInfo, error) {
			return &RepoInfo{
				Name:      "secret",
				IsPrivate: true,
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "https://git.example/secret", nil)
	meta := svc.Resolve(req)
	if meta.Status != http.StatusNotFound {
		t.Fatalf("status=%d", meta.Status)
	}
}

func TestServeDocument(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html lang="en"><head><title>x</title></head><body></body></html>`),
		},
	}
	svc := &Service{
		SiteName:  "Repo",
		PublicURL: "https://git.example",
		IndexFS:   fsys,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://git.example/", nil)
	svc.ServeDocument(rec, req, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not run for SPA index")
	}))
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "<title>Repo</title>") {
		t.Fatalf("body=%s", rec.Body.String())
	}

	if !strings.Contains(rec.Body.String(), `rel="canonical"`) {
		t.Fatalf("missing canonical: %s", rec.Body.String())
	}
}
