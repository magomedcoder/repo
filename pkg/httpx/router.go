package httpx

import (
	"net/http"
	"strings"
)

type Router struct {
	mux    *http.ServeMux
	prefix string
	mws    []Middleware
}

func New() *Router {
	return &Router{mux: http.NewServeMux()}
}

func (r *Router) Group(prefix string, mw ...Middleware) *Router {
	return &Router{
		mux:    r.mux,
		prefix: joinPath(r.prefix, prefix),
		mws:    append(append([]Middleware{}, r.mws...), mw...),
	}
}

func (r *Router) GET(path string, h Handler) {
	r.handle(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h Handler) {
	r.handle(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h Handler) {
	r.handle(http.MethodPut, path, h)
}

func (r *Router) PATCH(path string, h Handler) {
	r.handle(http.MethodPatch, path, h)
}

func (r *Router) DELETE(path string, h Handler) {
	r.handle(http.MethodDelete, path, h)
}

func (r *Router) handle(method, path string, h Handler) {
	pattern := method + " " + joinPath(r.prefix, path)
	r.mux.Handle(pattern, Handle(chain(r.mws, h)))
}

func (r *Router) Handler() http.Handler {
	return r.mux
}

func joinPath(base, elem string) string {
	if elem == "" {
		if base == "" {
			return ""
		}
		return base
	}

	if !strings.HasPrefix(elem, "/") {
		elem = "/" + elem
	}

	if base == "" || base == "/" {
		return elem
	}

	return strings.TrimSuffix(base, "/") + elem
}
