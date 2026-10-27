package spa

import (
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

type Options struct {
	Index        string
	AssetsPrefix string
	Gzip         *bool
}

func FileServer(fsys fs.FS, opts Options) http.Handler {
	if opts.Index == "" {
		opts.Index = "index.html"
	}

	if opts.AssetsPrefix == "" {
		opts.AssetsPrefix = "assets/"
	}

	gzipOn := true
	if opts.Gzip != nil {
		gzipOn = *opts.Gzip
	}

	return &server{
		fsys:         fsys,
		index:        opts.Index,
		assetsPrefix: opts.AssetsPrefix,
		gzip:         gzipOn,
	}
}

func Sub(fsys fs.FS, dir string, opts Options) (http.Handler, error) {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		return nil, err
	}

	return FileServer(sub, opts), nil
}

type server struct {
	fsys         fs.FS
	index        string
	assetsPrefix string
	gzip         bool
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := resolveName(s.fsys, r.URL.Path, s.index)
	setCacheHeaders(w, name, s.index, s.assetsPrefix)
	s.serveFile(w, r, name)
}

func resolveName(fsys fs.FS, urlPath, index string) string {
	name := path.Clean("/" + urlPath)
	name = strings.TrimPrefix(name, "/")
	if name == "" || name == "." {
		return index
	}

	f, err := fsys.Open(name)
	if err != nil {
		return index
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return index
	}
	if info.IsDir() {
		dirIndex := path.Join(name, index)
		if _, err := fsys.Open(dirIndex); err == nil {
			return dirIndex
		}

		return index
	}

	return name
}

func setCacheHeaders(w http.ResponseWriter, name, index, assetsPrefix string) {
	if name == index || strings.HasSuffix(name, "/"+index) {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}

	if assetsPrefix != "" && strings.HasPrefix(name, assetsPrefix) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
}

func (s *server) serveFile(w http.ResponseWriter, r *http.Request, name string) {
	if !s.gzip || !wantsGzip(r) || !compressible(name) {
		http.ServeFileFS(w, r, s.fsys, name)
		return
	}

	f, err := s.fsys.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	if ctype := contentType(name); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	gz := gzip.NewWriter(w)
	defer gz.Close()
	_, _ = io.Copy(gz, f)
}

func wantsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}

func compressible(name string) bool {
	switch path.Ext(name) {
	case ".html", ".js", ".mjs", ".css", ".svg", ".json", ".txt", ".xml", ".map", ".wasm":
		return true
	default:
		return false
	}
}

func contentType(name string) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json", ".map":
		return "application/json"
	case ".wasm":
		return "application/wasm"
	default:
		return ""
	}
}
