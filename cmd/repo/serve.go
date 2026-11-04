package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"magomedcoder.ru/repo"
	"magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "magomedcoder.ru/repo/internal/adapter/http"
	"magomedcoder.ru/repo/internal/adapter/http/audit"
	"magomedcoder.ru/repo/internal/adapter/http/handler"
	"magomedcoder.ru/repo/internal/adapter/http/middleware"
	"magomedcoder.ru/repo/internal/adapter/sqlite"
	"magomedcoder.ru/repo/internal/config"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/argon2id"
	"magomedcoder.ru/repo/pkg/token"
)

func runServe() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load("")
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	if err := cfg.ValidateSecrets(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	db, err := sqlite.NewDB(cfg.DataDir)
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("database", "err", err)
		os.Exit(1)
	}
	defer func() { _ = sqlDB.Close() }()

	userStore := sqlite.NewUserStore(db)
	sessionStore := sqlite.NewSessionStore(db)
	folderStore := sqlite.NewFolderStore(db)
	repoStore := sqlite.NewRepositoryStore(db)
	tokenStore := sqlite.NewAccessTokenStore(db)
	issueStore := sqlite.NewIssueStore(db)
	pullStore := sqlite.NewPullRequestStore(db)
	activityStore := sqlite.NewActivityStore(db)
	accessStore := sqlite.NewAccessLogStore(db)
	folderMemberStore := sqlite.NewFolderMemberStore(db)
	teamStore := sqlite.NewTeamStore(db)
	teamMemberStore := sqlite.NewTeamMemberStore(db)
	teamRepoGrantStore := sqlite.NewTeamRepoGrantStore(db)
	collabStore := sqlite.NewRepoCollaboratorStore(db)
	accessLog := audit.New(accessStore)

	gitRepo := git.NewRepository(cfg.DataDir)

	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	middleware.SetTrustedProxies(cfg.TrustedProxyNets)

	authz := usecase.NewAuthz(folderMemberStore, teamStore, teamMemberStore, teamRepoGrantStore, collabStore)
	authUC := usecase.NewAuthUseCase(userStore, sessionStore, tokenStore, hasher, tokens)
	authUC.SetSessionSecret(cfg.SessionSecret)
	authUC.SetAllowRegistration(cfg.AllowRegistration)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore, folderMemberStore, gitRepo, authz)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore, authz)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore, authz)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore, authz)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore, authz)
	membershipUC := usecase.NewAccessUseCase(folderStore, repoStore, userStore, folderMemberStore, teamStore, teamMemberStore, teamRepoGrantStore, collabStore, authz)
	profileUC := usecase.NewProfileUseCase(userStore, hasher, cfg.DataDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)
	adminUC := usecase.NewAdminUseCase(userStore, sessionStore, tokenStore, authUC, hasher)
	accessUC := usecase.NewAccessLogUseCase(accessStore)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCookiePolicy(cfg.CookieSecure, http.SameSiteLaxMode)
	authHandler.SetCSRFSecret(cfg.SessionSecret)
	authHandler.SetAllowRegistration(cfg.AllowRegistration)
	authHandler.SetAccessLogger(accessLog)
	folderHandler := handler.NewFolderHandler(folderUC)
	membershipHandler := handler.NewMembershipHandler(membershipUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)
	repoHandler.SetAccessLogger(accessLog)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	tokenHandler.SetAccessLogger(accessLog)
	gitHandler := handler.NewGitHandler(gitUC)
	gitHandler.SetReceiveMaxBytes(cfg.ReceiveMaxBytes)
	gitHandler.SetAccessLogger(accessLog)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	activityHandler := handler.NewActivityHandler(activityUC)
	siteHandler := handler.NewSiteHandler(cfg.SiteName)
	adminHandler := handler.NewAdminHandler(adminUC, accessUC)
	adminHandler.SetAccessLogger(accessLog)
	accessHandler := handler.NewAccessHandler(accessLog)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, folderHandler, membershipHandler, tokenHandler, gitHandler, issueHandler, pullHandler, profileHandler, activityHandler, siteHandler, adminHandler, accessHandler, authUC, cfg.CookieSecure, cfg.SessionSecret)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           withStatic(cfg.DataDir, router, accessLog, authUC, cfg.CookieSecure),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Minute,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

func withStatic(dataDir string, api http.Handler, access *audit.Logger, auth middleware.Authenticator, cookieSecure bool) http.Handler {
	frontend, err := repo.WebHandler()
	if err != nil {
		slog.Error("web", "err", err)
		os.Exit(1)
	}

	spa := middleware.SecurityHeaders(cookieSecure)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if access != nil && audit.IsPageVisit(r) {
			var user *domain.User
			if auth != nil {
				if token := middleware.SessionToken(r); token != "" {
					if u, _, err := auth.Authenticate(r.Context(), token); err == nil {
						user = u
					}
				}
			}
			access.Visit(r, user)
		}
		frontend.ServeHTTP(w, r)
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if handler.IsGitHTTPPath(path) || isAPIPath(path) {
			api.ServeHTTP(w, r)
			return
		}

		_ = dataDir
		spa.ServeHTTP(w, r)
	})
}

func isAPIPath(urlPath string) bool {
	return urlPath == "/api" || strings.HasPrefix(urlPath, "/api/")
}
