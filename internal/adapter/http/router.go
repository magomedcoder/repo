package http

import (
	"net/http"

	"repo.magomedcoder.ru/repo/internal/adapter/http/handler"
	"repo.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"repo.magomedcoder.ru/repo/pkg/i18n"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	repoHandler *handler.RepositoryHandler,
	folderHandler *handler.FolderHandler,
	tokenHandler *handler.TokenHandler,
	gitHandler *handler.GitHandler,
	issueHandler *handler.IssueHandler,
	pullHandler *handler.PullRequestHandler,
	profileHandler *handler.ProfileHandler,
	activityHandler *handler.ActivityHandler,
	auth middleware.Authenticator,
	cookieSecure bool,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/auth/registration", authHandler.Registration)
	mux.HandleFunc("POST /api/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("GET /api/auth/me", authHandler.Me)

	requireAuth := middleware.RequireAuth(auth)
	optionalAuth := middleware.OptionalAuth(auth)
	mux.Handle("POST /api/auth/password", requireAuth(http.HandlerFunc(authHandler.Password)))

	mux.Handle("GET /api/users/{username}", optionalAuth(http.HandlerFunc(profileHandler.Get)))
	mux.Handle("GET /api/users/{username}/avatar", http.HandlerFunc(profileHandler.ServeAvatar))
	mux.Handle("PATCH /api/users/me", requireAuth(http.HandlerFunc(profileHandler.UpdateMe)))
	mux.Handle("POST /api/users/me/avatar", requireAuth(http.HandlerFunc(profileHandler.UploadAvatar)))
	mux.Handle("DELETE /api/users/me/avatar", requireAuth(http.HandlerFunc(profileHandler.DeleteAvatar)))

	mux.Handle("POST /api/repos", requireAuth(http.HandlerFunc(repoHandler.Create)))
	mux.Handle("GET /api/repos", optionalAuth(http.HandlerFunc(repoHandler.List)))
	repoAPI := &handler.RepoDispatcher{
		Repos:  repoHandler,
		Issues: issueHandler,
		Pulls:  pullHandler,
	}
	mux.Handle("GET /api/repos/{path...}", optionalAuth(http.HandlerFunc(repoAPI.DispatchGet)))
	mux.Handle("PATCH /api/repos/{path...}", requireAuth(http.HandlerFunc(repoAPI.DispatchPatch)))
	mux.Handle("DELETE /api/repos/{path...}", requireAuth(http.HandlerFunc(repoAPI.DispatchDelete)))
	mux.Handle("POST /api/repos/{path...}", requireAuth(http.HandlerFunc(repoAPI.DispatchPost)))

	mux.Handle("GET /api/activity/{owner}", optionalAuth(http.HandlerFunc(activityHandler.ListOwner)))

	mux.Handle("POST /api/folders", requireAuth(http.HandlerFunc(folderHandler.Create)))
	mux.Handle("GET /api/folders", requireAuth(http.HandlerFunc(folderHandler.List)))
	mux.Handle("GET /api/folders/{path...}", requireAuth(http.HandlerFunc(folderHandler.Get)))
	mux.Handle("PATCH /api/folders/{path...}", requireAuth(http.HandlerFunc(folderHandler.Rename)))
	mux.Handle("DELETE /api/folders/{path...}", requireAuth(http.HandlerFunc(folderHandler.Delete)))

	mux.Handle("POST /api/tokens", requireAuth(http.HandlerFunc(tokenHandler.Create)))
	mux.Handle("GET /api/tokens", requireAuth(http.HandlerFunc(tokenHandler.List)))
	mux.Handle("DELETE /api/tokens/{id}", requireAuth(http.HandlerFunc(tokenHandler.Revoke)))

	apiLimiter := middleware.NewRateLimiter(180, 60)
	authLimiter := middleware.NewRateLimiter(30, 10)
	gitLimiter := middleware.NewRateLimiter(60, 20)
	api := middleware.SecurityHeaders(middleware.CSRF(cookieSecure)(middleware.MaxBody(1 << 20)(middleware.RateLimit(apiLimiter, authLimiter)(middleware.Logger(i18n.Middleware(mux))))))
	gitHTTP := middleware.SecurityHeaders(middleware.RateLimitIP(gitLimiter)(middleware.Logger(gitHandler)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.IsGitHTTPPath(r.URL.Path) {
			gitHTTP.ServeHTTP(w, r)
			return
		}

		api.ServeHTTP(w, r)
	})
}
