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

	"gorm.io/gorm"

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
	"magomedcoder.ru/repo/pkg/secretbox"
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

	userStore := sqlite.NewUserRepo(db)
	sessionStore := sqlite.NewSessionRepo(db)
	repoStore := sqlite.NewRepositoryRepo(db)
	tokenStore := sqlite.NewAccessTokenRepo(db)
	issueStore := sqlite.NewIssueRepo(db)
	pullStore := sqlite.NewPullRequestRepo(db)
	activityStore := sqlite.NewActivityRepo(db)
	accessStore := sqlite.NewAccessLogRepo(db)
	teamStore := sqlite.NewTeamRepo(db)
	teamMemberStore := sqlite.NewTeamMemberRepo(db)
	teamRepoGrantStore := sqlite.NewTeamRepoGrantRepo(db)
	collabStore := sqlite.NewRepoCollaboratorRepo(db)
	accessLog := audit.New(accessStore)

	gitRepo := git.NewRepository(cfg.DataDir)

	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	middleware.SetTrustedProxies(cfg.TrustedProxyNets)

	authz := usecase.NewAuthz(teamRepoGrantStore, collabStore)
	authUC := usecase.NewAuthUseCase(userStore, sessionStore, tokenStore, hasher, tokens)
	authUC.SetSessionSecret(cfg.SessionSecret)
	authUC.SetAllowRegistration(cfg.AllowRegistration)
	repoUC := usecase.NewRepositoryUseCase(repoStore, userStore, gitRepo, activityStore, authz)
	repoUC.SetPasswordHasher(hasher)
	if key, err := secretbox.DeriveKey(cfg.MirrorSecretsMaterial()); err != nil {
		slog.Error("mirror secrets key", "err", err)
		os.Exit(1)
	} else {
		repoUC.SetMirrorSecretsKey(key)
	}

	if err := setupMirror(repoUC, cfg, db); err != nil {
		slog.Error("mirror", "err", err)
		os.Exit(1)
	}

	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	tokenUC.SetPepper(cfg.SessionSecret)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore, authz)
	gitUC.SetTokenPepper(cfg.SessionSecret)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, userStore, activityStore, authz)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, userStore, gitRepo, activityStore, authz)
	membershipUC := usecase.NewAccessUseCase(repoStore, userStore, collabStore, authz)
	teamUC := usecase.NewTeamUseCase(teamStore, teamMemberStore, teamRepoGrantStore, repoStore, userStore, authz)
	profileUC := usecase.NewProfileUseCase(userStore, hasher, cfg.DataDir)
	profileUC.SetSessionStores(sessionStore, tokenStore)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)
	adminUC := usecase.NewAdminUseCase(userStore, sessionStore, tokenStore, authUC, hasher)
	adminUC.SetDataStores(repoStore, accessStore)
	accessUC := usecase.NewAccessLogUseCase(accessStore)
	adminMirrorUC := usecase.NewAdminMirrorUseCase(repoUC)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCookiePolicy(cfg.CookieSecure, http.SameSiteLaxMode)
	authHandler.SetCSRFSecret(cfg.SessionSecret)
	authHandler.SetAllowRegistration(cfg.AllowRegistration)
	authHandler.SetAccessLogger(accessLog)
	membershipHandler := handler.NewMembershipHandler(membershipUC)
	teamHandler := handler.NewTeamHandler(teamUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)
	repoHandler.SetAccessLogger(accessLog)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	tokenHandler.SetAccessLogger(accessLog)
	gitHandler := handler.NewGitHandler(gitUC)
	gitHandler.SetReceiveMaxBytes(cfg.ReceiveMaxBytes)
	gitHandler.SetAccessLogger(accessLog)
	gitHandler.SetMirrorScheduler(repoUC)
	pullUC.SetPushMirrorScheduler(repoUC)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	activityHandler := handler.NewActivityHandler(activityUC)
	siteHandler := handler.NewSiteHandler(cfg.SiteName)
	adminHandler := handler.NewAdminHandler(adminUC, accessUC)
	adminHandler.SetAccessLogger(accessLog)
	adminHandler.SetMirrors(adminMirrorUC)
	accessHandler := handler.NewAccessHandler(accessLog)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, membershipHandler, teamHandler, tokenHandler, gitHandler, issueHandler, pullHandler, profileHandler, activityHandler, siteHandler, adminHandler, accessHandler, authUC, cfg.CookieSecure, cfg.SessionSecret)

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

	repoUC.SetMirrorRootContext(ctx)
	go repoUC.RunPullMirrorLoop(ctx, cfg.MirrorPullEvery)
	authUC.StartAttemptsSweep(ctx, attemptsSweepEvery)
	gitUC.StartAttemptsSweep(ctx, attemptsSweepEvery)

	maintenanceDone := make(chan struct{})
	go func() {
		defer close(maintenanceDone)
		runMaintenance(ctx, maintenanceDeps{
			db:                   db,
			sessions:             sessionStore,
			tokens:               tokenStore,
			activity:             activityStore,
			repos:                repoUC,
			softDeletedRetention: cfg.SoftDeletedRetentionDur,
		})
	}()

	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	repoUC.StopMirrorDebouncers()
	idle := make(chan struct{})
	go func() {
		repoUC.WaitMirrorIdle()
		close(idle)
	}()
	select {
	case <-idle:
	case <-time.After(30 * time.Second):
		slog.Warn("shutdown: mirror work still running")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}

	select {
	case <-maintenanceDone:
	case <-time.After(5 * time.Second):
		slog.Warn("shutdown: maintenance still running")
	}

	accessLog.Close()
}

const (
	maintenanceEvery    = time.Hour
	maintenanceFirstRun = 15 * time.Second
	attemptsSweepEvery  = 5 * time.Minute
	activityRetention   = 90 * 24 * time.Hour
)

type repoPurger interface {
	PurgeSoftDeletedRepos(ctx context.Context, olderThan time.Duration) (int, error)
}

type maintenanceDeps struct {
	db                   *gorm.DB
	sessions             domain.SessionStore
	tokens               domain.AccessTokenStore
	activity             domain.ActivityStore
	repos                repoPurger
	softDeletedRetention time.Duration
}

func runMaintenance(ctx context.Context, d maintenanceDeps) {
	timer := time.NewTimer(maintenanceFirstRun)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		maintenanceOnce(ctx, d, time.Now().UTC())
		timer.Reset(maintenanceEvery)
	}
}

func maintenanceOnce(ctx context.Context, d maintenanceDeps, now time.Time) {
	if d.sessions != nil {
		if n, err := d.sessions.DeleteExpired(ctx, now); err != nil {
			slog.Warn("maintenance: sessions", "err", err)
		} else if n > 0 {
			slog.Info("maintenance: expired sessions removed", "rows", n)
		}
	}

	if d.tokens != nil {
		if n, err := d.tokens.DeleteExpired(ctx, now); err != nil {
			slog.Warn("maintenance: access tokens", "err", err)
		} else if n > 0 {
			slog.Info("maintenance: expired access tokens removed", "rows", n)
		}
	}

	if d.activity != nil {
		if n, err := d.activity.DeleteOlderThan(ctx, now.Add(-activityRetention)); err != nil {
			slog.Warn("maintenance: activity", "err", err)
		} else if n > 0 {
			slog.Info("maintenance: old activity events removed", "rows", n)
		}
	}

	if d.repos != nil {
		retention := d.softDeletedRetention
		if retention <= 0 {
			retention = config.DefaultSoftDeletedRetention
		}

		slog.Debug("maintenance: soft-deleted purge", "retention", retention.String())
		if n, err := d.repos.PurgeSoftDeletedRepos(ctx, retention); err != nil {
			slog.Warn("maintenance: soft-deleted repositories", "err", err)
		} else if n > 0 {
			slog.Info("maintenance: soft-deleted repositories purged", "repos", n, "retention", retention.String())
		}
	}

	if d.db != nil {
		if err := sqlite.CheckpointWAL(d.db); err != nil {
			slog.Warn("maintenance: wal checkpoint", "err", err)
		}
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
