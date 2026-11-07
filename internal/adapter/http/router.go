package http

import (
	"net/http"

	"magomedcoder.ru/repo/internal/adapter/http/handler"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/pkg/httpx"
	"magomedcoder.ru/repo/pkg/i18n"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	repoHandler *handler.RepositoryHandler,
	membershipHandler *handler.MembershipHandler,
	teamHandler *handler.TeamHandler,
	tokenHandler *handler.TokenHandler,
	gitHandler *handler.GitHandler,
	issueHandler *handler.IssueHandler,
	pullHandler *handler.PullRequestHandler,
	profileHandler *handler.ProfileHandler,
	activityHandler *handler.ActivityHandler,
	siteHandler *handler.SiteHandler,
	adminHandler *handler.AdminHandler,
	accessHandler *handler.AccessHandler,
	auth middleware.Authenticator,
	cookieSecure bool,
	sessionSecret string,
) http.Handler {
	r := httpx.New()
	api := r.Group("/api")

	api.GET("/site", siteHandler.Get)
	api.GET("/auth/registration", authHandler.Registration)
	api.GET("/auth/csrf", authHandler.CSRF)
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/logout", authHandler.Logout)
	api.GET("/auth/me", authHandler.Me)
	api.GET("/users/{username}/avatar", profileHandler.ServeAvatar)

	require := httpx.HTTPMiddleware(middleware.RequireAuth(auth))
	optional := httpx.HTTPMiddleware(middleware.OptionalAuth(auth))
	authed := api.Group("", require)
	pub := api.Group("", optional)

	authed.POST("/auth/password", authHandler.Password)
	if accessHandler != nil {
		authed.POST("/access/visit", accessHandler.Visit)
	}
	authed.GET("/auth/sessions", authHandler.ListSessions)
	authed.DELETE("/auth/sessions/{id}", authHandler.RevokeSession)
	authed.POST("/auth/sessions/revoke-others", authHandler.RevokeOtherSessions)
	authed.POST("/auth/sessions/revoke-all", authHandler.RevokeAllSessions)

	pub.GET("/users/{username}", profileHandler.Get)
	authed.PATCH("/users/me", profileHandler.UpdateMe)
	authed.POST("/users/me/avatar", profileHandler.UploadAvatar)
	authed.DELETE("/users/me/avatar", profileHandler.DeleteAvatar)

	authed.POST("/repos", repoHandler.Create)
	pub.GET("/repos", repoHandler.List)
	pub.GET("/search", repoHandler.Search)

	repoAPI := &handler.RepoDispatcher{
		Repos:      repoHandler,
		Issues:     issueHandler,
		Pulls:      pullHandler,
		Membership: membershipHandler,
		Teams:      teamHandler,
	}
	handler.RegisterRepoRoutes(pub, authed, repoAPI)

	pub.GET("/activity/{owner}", activityHandler.ListOwner)

	authed.GET("/teams", teamHandler.List)
	authed.POST("/teams", teamHandler.Create)
	authed.GET("/teams/{slug}", teamHandler.Get)
	authed.PATCH("/teams/{slug}", teamHandler.Update)
	authed.DELETE("/teams/{slug}", teamHandler.Delete)
	authed.GET("/teams/{slug}/members", teamHandler.ListMembers)
	authed.POST("/teams/{slug}/members", teamHandler.AddMember)
	authed.DELETE("/teams/{slug}/members/{username}", teamHandler.RemoveMember)
	authed.GET("/teams/{slug}/repos", teamHandler.ListRepos)
	authed.POST("/teams/{slug}/repos", teamHandler.SetRepo)
	authed.DELETE("/teams/{slug}/repos/{repo}", teamHandler.RemoveRepo)

	authed.POST("/tokens", tokenHandler.Create)
	authed.GET("/tokens", tokenHandler.List)
	authed.DELETE("/tokens/{id}", tokenHandler.Revoke)

	admin := authed.Group("", httpx.HTTPMiddleware(middleware.RequireAdmin()))
	admin.GET("/admin/overview", adminHandler.Overview)
	admin.GET("/admin/users", adminHandler.ListUsers)
	admin.POST("/admin/users", adminHandler.CreateUser)
	admin.PATCH("/admin/users/{id}", adminHandler.UpdateUser)
	admin.DELETE("/admin/users/{id}", adminHandler.DeleteUser)
	admin.GET("/admin/repos", adminHandler.ListRepos)
	admin.GET("/admin/access-logs", adminHandler.ListAccessLogs)
	admin.GET("/admin/mirrors", adminHandler.ListMirrors)
	admin.GET("/admin/mirrors/health", adminHandler.MirrorHealth)
	admin.GET("/admin/mirrors/metrics", adminHandler.MirrorMetrics)
	admin.POST("/admin/mirrors/verify", adminHandler.VerifyMirrors)
	admin.POST("/admin/mirrors/import", adminHandler.ImportOrgMirrors)
	admin.POST("/admin/mirrors/{action}", adminHandler.BulkMirrors)
	admin.GET("/admin/mirror-groups", adminHandler.ListMirrorGroups)
	admin.POST("/admin/mirror-groups", adminHandler.CreateMirrorGroup)
	admin.DELETE("/admin/mirror-groups/{id}", adminHandler.DeleteMirrorGroup)

	apiLimiter := middleware.NewRateLimiter(180, 60)
	authLimiter := middleware.NewRateLimiter(30, 10)
	contributeLimiter := middleware.NewRateLimiter(40, 15)
	gitLimiter := middleware.NewRateLimiter(40, 15)
	apiHandler := middleware.SecurityHeaders(cookieSecure)(middleware.CSRF(cookieSecure, sessionSecret)(middleware.MaxBody(1 << 20)(middleware.RateLimitWithContribute(apiLimiter, authLimiter, contributeLimiter)(middleware.Logger(i18n.Middleware(r.Handler()))))))
	gitHTTP := middleware.SecurityHeaders(cookieSecure)(middleware.RateLimitIP(gitLimiter)(middleware.Logger(gitHandler)))

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if handler.IsGitHTTPPath(req.URL.Path) {
			gitHTTP.ServeHTTP(w, req)
			return
		}

		apiHandler.ServeHTTP(w, req)
	})
}
