package http

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	repoHandler *handler.RepositoryHandler,
	folderHandler *handler.FolderHandler,
	auth middleware.Authenticator,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/me", authHandler.Me)

	requireAuth := middleware.RequireAuth(auth)
	optionalAuth := middleware.OptionalAuth(auth)

	mux.Handle("POST /api/repos", requireAuth(http.HandlerFunc(repoHandler.Create)))
	mux.Handle("GET /api/repos", optionalAuth(http.HandlerFunc(repoHandler.List)))
	mux.Handle("GET /api/repos/{owner}/{path...}", optionalAuth(http.HandlerFunc(repoHandler.Get)))
	mux.Handle("PATCH /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(repoHandler.Update)))
	mux.Handle("DELETE /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(repoHandler.Delete)))
	mux.Handle("POST /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(repoHandler.Move)))

	mux.Handle("POST /api/folders", requireAuth(http.HandlerFunc(folderHandler.Create)))
	mux.Handle("GET /api/folders", requireAuth(http.HandlerFunc(folderHandler.List)))
	mux.Handle("GET /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Get)))
	mux.Handle("PATCH /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Rename)))
	mux.Handle("DELETE /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Delete)))
	mux.Handle("POST /api/folders/{id}/move", requireAuth(http.HandlerFunc(folderHandler.Move)))

	return middleware.Logger(mux)
}
