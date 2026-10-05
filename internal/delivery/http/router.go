package http

import (
	"net/http"

	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/delivery/http/middleware"
	"github.com/magomedcoder/repo/pkg/i18n"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	repoHandler *handler.RepositoryHandler,
	folderHandler *handler.FolderHandler,
	tokenHandler *handler.TokenHandler,
	sshKeyHandler *handler.SSHKeyHandler,
	gitHandler *handler.GitHandler,
	issueHandler *handler.IssueHandler,
	pullHandler *handler.PullRequestHandler,
	profileHandler *handler.ProfileHandler,
	orgHandler *handler.OrganizationHandler,
	auth middleware.Authenticator,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/me", authHandler.Me)

	requireAuth := middleware.RequireAuth(auth)
	optionalAuth := middleware.OptionalAuth(auth)

	mux.Handle("GET /api/users/{username}", optionalAuth(http.HandlerFunc(profileHandler.Get)))
	mux.Handle("GET /api/users/{username}/avatar", http.HandlerFunc(profileHandler.ServeAvatar))
	mux.Handle("PATCH /api/users/me", requireAuth(http.HandlerFunc(profileHandler.UpdateMe)))
	mux.Handle("POST /api/users/me/avatar", requireAuth(http.HandlerFunc(profileHandler.UploadAvatar)))
	mux.Handle("DELETE /api/users/me/avatar", requireAuth(http.HandlerFunc(profileHandler.DeleteAvatar)))

	mux.Handle("POST /api/orgs", requireAuth(http.HandlerFunc(orgHandler.Create)))
	mux.Handle("GET /api/orgs", requireAuth(http.HandlerFunc(orgHandler.ListMine)))
	mux.Handle("GET /api/orgs/{slug}", optionalAuth(http.HandlerFunc(orgHandler.Get)))
	mux.Handle("PATCH /api/orgs/{slug}", requireAuth(http.HandlerFunc(orgHandler.Update)))
	mux.Handle("DELETE /api/orgs/{slug}", requireAuth(http.HandlerFunc(orgHandler.Delete)))
	mux.Handle("GET /api/orgs/{slug}/repos", optionalAuth(http.HandlerFunc(orgHandler.ListRepos)))
	mux.Handle("GET /api/orgs/{slug}/members", requireAuth(http.HandlerFunc(orgHandler.ListMembers)))
	mux.Handle("POST /api/orgs/{slug}/members", requireAuth(http.HandlerFunc(orgHandler.AddMember)))
	mux.Handle("PATCH /api/orgs/{slug}/members/{userID}", requireAuth(http.HandlerFunc(orgHandler.UpdateMember)))
	mux.Handle("DELETE /api/orgs/{slug}/members/{userID}", requireAuth(http.HandlerFunc(orgHandler.RemoveMember)))
	mux.Handle("POST /api/orgs/{slug}/leave", requireAuth(http.HandlerFunc(orgHandler.Leave)))

	mux.Handle("POST /api/repos", requireAuth(http.HandlerFunc(repoHandler.Create)))
	mux.Handle("GET /api/repos", optionalAuth(http.HandlerFunc(repoHandler.List)))
	mux.Handle("GET /api/repos/{owner}/{path...}", optionalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pullHandler.Handle(w, r) {
			return
		}

		if issueHandler.Handle(w, r) {
			return
		}
		repoHandler.DispatchGet(w, r)
	})))
	mux.Handle("PATCH /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pullHandler.Handle(w, r) {
			return
		}

		if issueHandler.Handle(w, r) {
			return
		}
		repoHandler.Update(w, r)
	})))
	mux.Handle("DELETE /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pullHandler.Handle(w, r) {
			return
		}

		if issueHandler.Handle(w, r) {
			return
		}
		repoHandler.Delete(w, r)
	})))
	mux.Handle("POST /api/repos/{owner}/{path...}", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if pullHandler.Handle(w, r) {
			return
		}

		if issueHandler.Handle(w, r) {
			return
		}
		repoHandler.Move(w, r)
	})))

	mux.Handle("POST /api/folders", requireAuth(http.HandlerFunc(folderHandler.Create)))
	mux.Handle("GET /api/folders", requireAuth(http.HandlerFunc(folderHandler.List)))
	mux.Handle("GET /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Get)))
	mux.Handle("PATCH /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Rename)))
	mux.Handle("DELETE /api/folders/{id}", requireAuth(http.HandlerFunc(folderHandler.Delete)))
	mux.Handle("POST /api/folders/{id}/move", requireAuth(http.HandlerFunc(folderHandler.Move)))

	mux.Handle("POST /api/tokens", requireAuth(http.HandlerFunc(tokenHandler.Create)))
	mux.Handle("GET /api/tokens", requireAuth(http.HandlerFunc(tokenHandler.List)))
	mux.Handle("DELETE /api/tokens/{id}", requireAuth(http.HandlerFunc(tokenHandler.Revoke)))

	mux.Handle("POST /api/ssh-keys", requireAuth(http.HandlerFunc(sshKeyHandler.Create)))
	mux.Handle("GET /api/ssh-keys", requireAuth(http.HandlerFunc(sshKeyHandler.List)))
	mux.Handle("DELETE /api/ssh-keys/{id}", requireAuth(http.HandlerFunc(sshKeyHandler.Delete)))

	apiLimiter := middleware.NewRateLimiter(180, 60)
	authLimiter := middleware.NewRateLimiter(30, 10)
	api := middleware.SecurityHeaders(middleware.RateLimit(apiLimiter, authLimiter)(middleware.Logger(i18n.Middleware(mux))))
	gitHTTP := middleware.SecurityHeaders(middleware.Logger(gitHandler))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.IsGitHTTPPath(r.URL.Path) {
			gitHTTP.ServeHTTP(w, r)
			return
		}
		
		api.ServeHTTP(w, r)
	})
}
