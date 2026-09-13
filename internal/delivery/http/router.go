package http

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
)

func NewRouter(repoHandler *handler.RepositoryHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/user/repos", repoHandler.Create)

	return middleware.Logger(mux)
}
