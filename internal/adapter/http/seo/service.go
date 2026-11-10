package seo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/domain"
)

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*domain.User, *domain.Session, error)
}

type RepoInfo struct {
	Name           string
	Description    string
	RepoPath       string
	IsPrivate      bool
	LastActivityAt *time.Time
}

type Service struct {
	SiteName            string
	PublicURL           string
	CookieSecure        bool
	AnonymousReadDenied bool
	Auth                Authenticator
	FindRepo            func(ctx context.Context, logical string) (*RepoInfo, error)
	ListPublic          func(ctx context.Context, limit, offset int) ([]RepoInfo, error)
	IndexFS             fs.FS
	IndexName           string
}

func (s *Service) indexFile() string {
	if s.IndexName != "" {
		return s.IndexName
	}

	return "index.html"
}

func (s *Service) BaseURL(r *http.Request) string {
	if u := strings.TrimRight(strings.TrimSpace(s.PublicURL), "/"); u != "" {
		return u
	}

	scheme := "http"
	if s.CookieSecure || middleware.IsHTTPS(r) {
		scheme = "https"
	}

	return scheme + "://" + r.Host
}

func (s *Service) Absolute(r *http.Request, p string) string {
	if p == "" {
		p = "/"
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	return s.BaseURL(r) + p
}

func (s *Service) locale(r *http.Request) (htmlLang, ogLocale string) {
	if c, err := r.Cookie("repo.locale"); err == nil {
		switch strings.ToLower(c.Value) {
		case "ru":
			return "ru", "ru_RU"
		case "en":
			return "en", "en_US"
		}
	}

	al := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(al, "ru") {
		return "ru", "ru_RU"
	}

	return "en", "en_US"
}

func (s *Service) Resolve(r *http.Request) Meta {
	site := s.SiteName
	if site == "" {
		site = "Repo"
	}

	_, ogLocale := s.locale(r)
	pathName := path.Clean("/" + r.URL.Path)
	if pathName == "." {
		pathName = "/"
	}

	canonicalPath := pathName
	if r.URL.RawQuery != "" && pathName == "/search" {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q != "" {
			canonicalPath = "/search?q=" + q
		}
	}

	meta := Meta{
		Status:      http.StatusOK,
		SiteName:    site,
		OGLocale:    ogLocale,
		OGImage:     s.Absolute(r, "/og.png"),
		Canonical:   s.Absolute(r, canonicalPath),
		Description: site,
		Title:       site,
		OGType:      "website",
		Robots:      "index,follow",
	}

	if isNoIndexPath(pathName) {
		meta.Robots = "noindex,nofollow"
		meta.Title = pageTitle(pathLabel(pathName), site)
		return meta
	}

	guestBlocked := s.AnonymousReadDenied && s.viewer(r) == nil

	switch {
	case pathName == "/":
		if guestBlocked {
			meta.Robots = "noindex,follow"
		}
		meta.Title = site
		meta.Description = fmt.Sprintf("%s - public Git repositories", site)
		meta.JSONLD = []map[string]any{
			{
				"@context": "https://schema.org",
				"@type":    "WebSite",
				"name":     site,
				"url":      s.Absolute(r, "/"),
				"potentialAction": map[string]any{
					"@type":       "SearchAction",
					"target":      s.Absolute(r, "/search") + "?q={search_term_string}",
					"query-input": "required name=search_term_string",
				},
			},
		}
		return meta

	case pathName == "/search":
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" || guestBlocked {
			meta.Robots = "noindex,follow"
			meta.Title = pageTitle("Search", site)
			return meta
		}

		meta.Title = pageTitle(q, site)
		meta.Description = fmt.Sprintf("Search results for %q on %s", q, site)
		meta.Robots = "index,follow"
		return meta
	}

	repoSlug, rest, ok := splitRepoPath(pathName)
	if !ok {
		meta.Robots = "noindex,follow"
		meta.Title = pageTitle("Not found", site)
		meta.Status = http.StatusNotFound
		return meta
	}

	if guestBlocked {
		meta.Status = http.StatusNotFound
		meta.Robots = "noindex,follow"
		meta.Title = pageTitle("Not found", site)
		return meta
	}

	info, err := s.FindRepo(r.Context(), repoSlug+".git")
	if err != nil || info == nil {
		meta.Status = http.StatusNotFound
		meta.Robots = "noindex,follow"
		meta.Title = pageTitle("Not found", site)
		return meta
	}

	viewer := s.viewer(r)
	if info.IsPrivate && viewer == nil {
		meta.Status = http.StatusNotFound
		meta.Robots = "noindex,follow"
		meta.Title = pageTitle("Not found", site)
		return meta
	}

	if info.IsPrivate {
		meta.Robots = "noindex,nofollow"
	}

	desc := strings.TrimSpace(info.Description)
	if desc == "" {
		desc = fmt.Sprintf("%s on %s", info.Name, site)
	}

	page := repoPageLabel(rest)
	if page == "" {
		meta.Title = pageTitle(info.Name, site)
	} else {
		meta.Title = pageTitle(page+" | "+info.Name, site)
	}
	meta.Description = desc
	meta.OGType = "article"
	meta.Canonical = s.Absolute(r, "/"+info.Name+rest)
	meta.JSONLD = []map[string]any{
		{
			"@context":       "https://schema.org",
			"@type":          "SoftwareSourceCode",
			"name":           info.Name,
			"description":    desc,
			"codeRepository": s.Absolute(r, "/"+info.Name+".git"),
			"url":            s.Absolute(r, "/"+info.Name),
		},
		{
			"@context": "https://schema.org",
			"@type":    "BreadcrumbList",
			"itemListElement": []map[string]any{
				{"@type": "ListItem", "position": 1, "name": site, "item": s.Absolute(r, "/")},
				{"@type": "ListItem", "position": 2, "name": info.Name, "item": s.Absolute(r, "/"+info.Name)},
			},
		},
	}

	return meta
}

func (s *Service) viewer(r *http.Request) *domain.User {
	if s.Auth == nil {
		return nil
	}

	token := middleware.SessionTokenFor(r, s.CookieSecure)
	if token == "" {
		return nil
	}

	u, _, err := s.Auth.Authenticate(r.Context(), token)
	if err != nil {
		return nil
	}

	return u
}

func (s *Service) ServeDocument(w http.ResponseWriter, r *http.Request, next http.Handler) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		next.ServeHTTP(w, r)
		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name != "" && name != "." {
		if f, err := s.IndexFS.Open(name); err == nil {
			stat, serr := f.Stat()
			_ = f.Close()
			if serr == nil && !stat.IsDir() {
				next.ServeHTTP(w, r)
				return
			}
		}
	}

	meta := s.Resolve(r)
	raw, err := fs.ReadFile(s.IndexFS, s.indexFile())
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	lang, _ := s.locale(r)
	body := InjectHTML(raw, meta)
	body = setHTMLLang(body, lang)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if meta.Robots != "" {
		w.Header().Set("X-Robots-Tag", meta.Robots)
	}

	status := meta.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if r.Method == http.MethodHead {
		return
	}

	_, _ = w.Write(body)
}

func setHTMLLang(raw []byte, lang string) []byte {
	if lang == "" {
		return raw
	}

	s := string(raw)
	if strings.Contains(s, `lang="`) {
		const prefix = `<html lang="`
		if i := strings.Index(s, prefix); i >= 0 {
			j := i + len(prefix)
			k := strings.Index(s[j:], `"`)
			if k >= 0 {
				s = s[:j] + htmlEscapeAttr(lang) + s[j+k:]
				return []byte(s)
			}
		}
	}

	return []byte(strings.Replace(s, "<html", `<html lang="`+htmlEscapeAttr(lang)+`"`, 1))
}

func htmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, `&`, "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func (s *Service) Robots(w http.ResponseWriter, r *http.Request) {
	base := s.BaseURL(r)
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	if s.AnonymousReadDenied {
		b.WriteString("Disallow: /\n")
	} else {
		b.WriteString("Allow: /\n")
		for _, p := range []string{"/login", "/register", "/profile", "/settings", "/cp", "/api/"} {
			b.WriteString("Disallow: ")
			b.WriteString(p)
			b.WriteByte('\n')
		}
		b.WriteString("Sitemap: ")
		b.WriteString(base)
		b.WriteString("/sitemap.xml\n")
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.WriteString(w, b.String())
}

func (s *Service) Manifest(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(s.SiteName)
	if name == "" {
		name = "Repo"
	}

	short := name
	if n := utf8.RuneCountInString(short); n > 12 {
		short = string([]rune(short)[:12])
	}

	payload := map[string]any{
		"name":             name,
		"short_name":       short,
		"start_url":        "/",
		"display":          "standalone",
		"background_color": "#fafaf9",
		"theme_color":      "#1c1917",
		"icons": []map[string]string{
			{"src": "/favicon-16x16.png", "sizes": "16x16", "type": "image/png"},
			{"src": "/favicon-32x32.png", "sizes": "32x32", "type": "image/png"},
			{"src": "/favicon.ico", "sizes": "48x48", "type": "image/x-icon"},
		},
	}

	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(payload)
}

func (s *Service) Sitemap(w http.ResponseWriter, r *http.Request) {
	base := s.BaseURL(r)
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=600")
	_, _ = io.WriteString(w, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	_, _ = io.WriteString(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`+"\n")
	writeURL(w, base+"/", time.Now().UTC(), "1.0")

	if s.ListPublic != nil {
		const page = 200
		for offset := 0; offset < 10000; offset += page {
			repos, err := s.ListPublic(r.Context(), page, offset)
			if err != nil || len(repos) == 0 {
				break
			}

			for _, repo := range repos {
				if repo.IsPrivate || repo.Name == "" {
					continue
				}

				last := time.Now().UTC()
				if repo.LastActivityAt != nil {
					last = repo.LastActivityAt.UTC()
				}

				writeURL(w, base+"/"+repo.Name, last, "0.8")
			}

			if len(repos) < page {
				break
			}
		}
	}

	_, _ = io.WriteString(w, `</urlset>`+"\n")
}

func writeURL(w http.ResponseWriter, loc string, lastMod time.Time, pri string) {
	_, _ = fmt.Fprintf(w, "  <url><loc>%s</loc><lastmod>%s</lastmod><priority>%s</priority></url>\n",
		xmlEscape(loc), lastMod.Format("2006-01-02"), pri)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func pageTitle(page, site string) string {
	page = strings.TrimSpace(page)
	site = strings.TrimSpace(site)
	if page == "" {
		return site
	}

	if site == "" {
		return page
	}

	full := page + " | " + site
	if len(full) <= 60 {
		return full
	}

	return page
}

func pathLabel(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}

	parts := strings.Split(p, "/")
	label := parts[0]
	switch label {
	case "login":
		return "Sign in"
	case "register":
		return "Register"
	case "profile":
		return "Profile"
	case "settings":
		return "Settings"
	case "cp":
		return "Control panel"
	default:
		return label
	}
}

func repoPageLabel(rest string) string {
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return ""
	}

	seg := strings.Split(rest, "/")[0]
	switch seg {
	case "issues":
		return "Issues"
	case "pulls":
		return "Pull requests"
	case "commits":
		return "Commits"
	case "blob":
		return "File"
	case "refs":
		return "Refs"
	case "settings":
		return "Settings"
	default:
		return seg
	}
}

func isNoIndexPath(p string) bool {
	p = path.Clean(p)
	prefixes := []string{
		"/login", "/register", "/profile", "/settings", "/cp",
	}
	for _, pre := range prefixes {
		if p == pre || strings.HasPrefix(p, pre+"/") {
			return true
		}
	}

	if strings.HasSuffix(p, "/settings") || strings.Contains(p, "/settings/") {
		return true
	}

	return false
}

func splitRepoPath(p string) (slug, rest string, ok bool) {
	p = strings.TrimPrefix(path.Clean(p), "/")
	if p == "" || p == "." {
		return "", "", false
	}

	parts := strings.SplitN(p, "/", 2)
	slug = parts[0]
	if domain.IsReservedSlug(slug) {
		return "", "", false
	}

	if strings.Contains(slug, ".") && !strings.HasSuffix(strings.ToLower(slug), ".git") {

	}

	rest = ""
	if len(parts) == 2 {
		rest = "/" + parts[1]
	}

	return slug, rest, true
}
