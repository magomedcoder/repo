package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gorm.io/gorm"

	"github.com/magomedcoder/repo"
	"github.com/magomedcoder/repo/internal/adapter/git"
	deliveryhttp "github.com/magomedcoder/repo/internal/adapter/http"
	"github.com/magomedcoder/repo/internal/adapter/http/audit"
	"github.com/magomedcoder/repo/internal/adapter/http/handler"
	"github.com/magomedcoder/repo/internal/adapter/http/middleware"
	"github.com/magomedcoder/repo/internal/adapter/http/seo"
	"github.com/magomedcoder/repo/internal/adapter/sqlite"
	"github.com/magomedcoder/repo/internal/config"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/argon2id"
	"github.com/magomedcoder/repo/pkg/i18n"
	"github.com/magomedcoder/repo/pkg/secretbox"
	"github.com/magomedcoder/repo/pkg/token"
)

func runServe() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := i18n.Preload(); err != nil {
		slog.Error("i18n", "err", err)
		os.Exit(1)
	}

	cfg, err := config.Load("")
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	if err := cfg.ValidateSecrets(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	if err := cfg.ValidateListen(); err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	for _, w := range cfg.WarnInsecure() {
		slog.Warn("config", "detail", w)
	}

	db, err := sqlite.NewDB(cfg.Server.DataDir)
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
	nameHoldStore := sqlite.NewRepoNameHoldRepo(db)
	accessLog := audit.New(accessStore)

	gitRepo := git.NewRepository(cfg.Server.DataDir)
	openpgpPath, err := git.ResolveTrustPath(cfg.Server.DataDir, cfg.Mirror.Verify.OpenPGPKeyring)
	if err != nil {
		slog.Error("mirror trust", "err", err)
		os.Exit(1)
	}

	signersPath, err := git.ResolveTrustPath(cfg.Server.DataDir, cfg.Mirror.Verify.AllowedSigners)
	if err != nil {
		slog.Error("mirror trust", "err", err)
		os.Exit(1)
	}

	trust, err := git.LoadTrustFiles(openpgpPath, signersPath)
	if err != nil {
		slog.Error("mirror trust", "err", err)
		os.Exit(1)
	}

	gitRepo.SetTrust(trust)
	if trust.Empty() {
		slog.Warn("mirror trust empty")
	}

	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	middleware.SetTrustedProxies(cfg.TrustedProxyNets)

	authz := usecase.NewAuthz(teamRepoGrantStore, collabStore)
	authz.SetAllowPublicContribute(cfg.Auth.AllowPublicContribute)
	authz.SetRequireAuthRead(cfg.RequireAuthRead())
	authUC := usecase.NewAuthUseCase(userStore, sessionStore, tokenStore, hasher, tokens)
	authUC.SetSessionSecret(cfg.Secrets.SessionSecret)
	authUC.SetAllowRegistration(cfg.Auth.AllowRegistration)
	authUC.SetSessionTTL(cfg.Auth.SessionAbsoluteDur, cfg.Auth.SessionIdleDur)
	repoUC := usecase.NewRepositoryUseCase(repoStore, userStore, gitRepo, activityStore, authz)
	repoUC.SetPasswordHasher(hasher)
	stepUp := authUC.PasswordGate()
	stepUp.SetOnLock(func(userID uint) {
		if err := sessionStore.DeleteByUserID(userID); err != nil {
			slog.Warn("revoke sessions", "user_id", userID, "err", err)
		}
	})
	repoUC.SetPasswordGate(stepUp)
	repoUC.SetNameHolds(nameHoldStore)
	repoUC.SetCodeIndex(sqlite.NewCodeIndex(db))
	if key, err := secretbox.DeriveKey(cfg.MirrorSecretsMaterial()); err != nil {
		slog.Error("mirror secrets", "err", err)
		os.Exit(1)
	} else {
		repoUC.SetMirrorSecretsKey(key)
	}

	if err := setupMirror(repoUC, cfg, db); err != nil {
		slog.Error("mirror", "err", err)
		os.Exit(1)
	}

	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	tokenUC.SetPepper(cfg.TokenPepperMaterial())
	tokenUC.SetStepUp(userStore, hasher)
	tokenUC.SetPasswordGate(stepUp)
	tokenUC.SetRepoScope(repoStore, authz)
	gitUC := usecase.NewGitUseCase(repoStore, userStore, tokenStore, hasher, gitRepo, activityStore, authz)
	gitUC.SetTokenPepper(cfg.TokenPepperMaterial())
	gitUC.SetLoginGuard(authUC.LoginGuard())
	gitUC.SetPasswordAuth(cfg.Auth.GitPasswordAuth)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, userStore, activityStore, authz)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, userStore, gitRepo, activityStore, authz)
	membershipUC := usecase.NewAccessUseCase(repoStore, userStore, collabStore, authz)
	membershipUC.SetTeamRepos(teamRepoGrantStore)
	teamUC := usecase.NewTeamUseCase(teamStore, teamMemberStore, teamRepoGrantStore, repoStore, userStore, authz)
	profileUC := usecase.NewProfileUseCase(userStore, hasher, cfg.Server.DataDir)
	profileUC.SetSessionStores(sessionStore, tokenStore)
	profileUC.SetPasswordGate(stepUp)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)
	activityUC.SetAuthz(authz)
	adminUC := usecase.NewAdminUseCase(userStore, sessionStore, tokenStore, authUC, hasher)
	adminUC.SetDataStores(repoStore, accessStore)
	adminUC.SetPasswordGate(stepUp)
	adminUC.SetRepoFiles(gitRepo)
	adminUC.SetNameHolds(nameHoldStore)
	adminUC.SetAccessCleanup(collabStore, teamStore, teamMemberStore, teamRepoGrantStore)
	accessUC := usecase.NewAccessLogUseCase(accessStore, userStore)
	adminMirrorUC := usecase.NewAdminMirrorUseCase(repoUC)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCookiePolicy(cfg.HTTP.CookieSecure, http.SameSiteLaxMode)
	authHandler.SetCSRFSecret(cfg.CSRFSecretMaterial())
	authHandler.SetAllowRegistration(cfg.Auth.AllowRegistration)
	authHandler.SetAccessLogger(accessLog)
	membershipHandler := handler.NewMembershipHandler(membershipUC)
	teamHandler := handler.NewTeamHandler(teamUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)
	repoHandler.SetAccessLogger(accessLog)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	tokenHandler.SetAccessLogger(accessLog)
	gitHandler := handler.NewGitHandler(gitUC)
	gitHandler.SetReceiveMaxBytes(cfg.HTTP.ReceiveMaxBytes)
	gitHandler.SetAccessLogger(accessLog)
	gitHandler.SetMirrorScheduler(repoUC)
	pullUC.SetPushMirrorScheduler(repoUC)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	profileHandler.SetCookieSecure(cfg.HTTP.CookieSecure)
	activityHandler := handler.NewActivityHandler(activityUC)
	siteHandler := handler.NewSiteHandler(cfg.Server.SiteName, cfg.Server.PublicURL)
	adminHandler := handler.NewAdminHandler(adminUC, accessUC)
	adminHandler.SetAccessLogger(accessLog)
	adminHandler.SetMirrors(adminMirrorUC)
	accessHandler := handler.NewAccessHandler(accessLog)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, membershipHandler, teamHandler, tokenHandler, gitHandler, issueHandler, pullHandler, profileHandler, activityHandler, siteHandler, adminHandler, accessHandler, authUC, cfg.HTTP.CookieSecure, cfg.CSRFSecretMaterial())

	webFS, err := repo.WebFS()
	if err != nil {
		slog.Error("web", "err", err)
		os.Exit(1)
	}

	seoSvc := newSEOService(cfg, authUC, repoStore, repoUC, webFS)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           withStatic(router, accessLog, authUC, cfg.HTTP.CookieSecure, seoSvc),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Minute,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	repoUC.SetMirrorRootContext(ctx)
	go repoUC.RunPullMirrorLoop(ctx, cfg.Mirror.PullEvery)
	go repoUC.RunCodeIndexBackfill(ctx)
	authUC.StartAttemptsSweep(ctx, attemptsSweepEvery)
	gitUC.StartAttemptsSweep(ctx, attemptsSweepEvery)

	maintenanceDone := make(chan struct{})
	go func() {
		defer close(maintenanceDone)
		runMaintenance(ctx, maintenanceDeps{
			db:                   db,
			sessions:             sessionStore,
			tokens:               tokenStore,
			nameHolds:            nameHoldStore,
			activity:             activityStore,
			repos:                repoUC,
			softDeletedRetention: cfg.Server.SoftDeletedRetentionDur,
		})
	}()

	go func() {
		slog.Info("listening", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown")
	repoUC.StopMirrorDebouncers()
	idle := make(chan struct{})
	go func() {
		repoUC.WaitMirrorIdle()
		close(idle)
	}()
	select {
	case <-idle:
	case <-time.After(30 * time.Second):
		slog.Warn("shutdown mirrors")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}

	select {
	case <-maintenanceDone:
	case <-time.After(5 * time.Second):
		slog.Warn("shutdown maintenance")
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
	nameHolds            domain.RepoNameHoldStore
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
			slog.Warn("maintenance sessions", "err", err)
		} else if n > 0 {
			slog.Info("maintenance sessions", "rows", n)
		}
	}

	if d.tokens != nil {
		if n, err := d.tokens.DeleteExpired(ctx, now); err != nil {
			slog.Warn("maintenance tokens", "err", err)
		} else if n > 0 {
			slog.Info("maintenance tokens", "rows", n)
		}
	}

	if d.nameHolds != nil {
		if n, err := d.nameHolds.DeleteExpired(ctx, now); err != nil {
			slog.Warn("maintenance holds", "err", err)
		} else if n > 0 {
			slog.Info("maintenance holds", "rows", n)
		}
	}

	if d.activity != nil {
		if n, err := d.activity.DeleteOlderThan(ctx, now.Add(-activityRetention)); err != nil {
			slog.Warn("maintenance activity", "err", err)
		} else if n > 0 {
			slog.Info("maintenance activity", "rows", n)
		}
	}

	if d.repos != nil {
		retention := d.softDeletedRetention
		if retention <= 0 {
			retention = config.DefaultSoftDeletedRetention
		}

		if n, err := d.repos.PurgeSoftDeletedRepos(ctx, retention); err != nil {
			slog.Warn("maintenance purge", "err", err)
		} else if n > 0 {
			slog.Info("maintenance purge", "repos", n)
		}
	}

	if d.db != nil {
		if err := sqlite.CheckpointWAL(d.db); err != nil {
			slog.Warn("maintenance wal", "err", err)
		}
	}
}

func withStatic(api http.Handler, access *audit.Logger, auth middleware.Authenticator, cookieSecure bool, seoSvc *seo.Service) http.Handler {
	frontend, err := repo.WebHandler()
	if err != nil {
		slog.Error("web", "err", err)
		os.Exit(1)
	}

	spa := middleware.SecurityHeaders(cookieSecure)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if access != nil && audit.IsPageVisit(r) {
			var user *domain.User
			if auth != nil {
				if token := middleware.SessionTokenFor(r, cookieSecure); token != "" {
					if u, _, err := auth.Authenticate(r.Context(), token); err == nil {
						user = u
					}
				}
			}
			access.Visit(r, user)
		}

		switch r.URL.Path {
		case "/robots.txt":
			seoSvc.Robots(w, r)
			return
		case "/sitemap.xml":
			seoSvc.Sitemap(w, r)
			return
		case "/site.webmanifest":
			seoSvc.Manifest(w, r)
			return
		}

		seoSvc.ServeDocument(w, r, frontend)
	}))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if handler.IsGitHTTPPath(path) || isAPIPath(path) {
			api.ServeHTTP(w, r)
			return
		}

		spa.ServeHTTP(w, r)
	})
}

func isAPIPath(urlPath string) bool {
	return urlPath == "/api" || strings.HasPrefix(urlPath, "/api/")
}

func newSEOService(cfg config.Config, auth *usecase.AuthUseCase, store domain.RepositoryStore, repos *usecase.RepositoryUseCase, webFS fs.FS) *seo.Service {
	return &seo.Service{
		SiteName:            cfg.Server.SiteName,
		PublicURL:           cfg.Server.PublicURL,
		CookieSecure:        cfg.HTTP.CookieSecure,
		AnonymousReadDenied: cfg.RequireAuthRead(),
		Auth:                auth,
		IndexFS:             webFS,
		FindRepo: func(ctx context.Context, logical string) (*seo.RepoInfo, error) {
			repo, err := store.FindByRepoPath(ctx, logical)
			if err != nil {
				return nil, err
			}

			return &seo.RepoInfo{
				Name:           repo.Name,
				Description:    repo.Description,
				RepoPath:       repo.RepoPath,
				IsPrivate:      repo.IsPrivate,
				LastActivityAt: repo.LastActivityAt,
			}, nil
		},
		ListPublic: func(ctx context.Context, limit, offset int) ([]seo.RepoInfo, error) {
			items, err := repos.ListPublic(ctx, limit, offset, domain.RepoOrderUpdated)
			if err != nil {
				return nil, err
			}

			out := make([]seo.RepoInfo, 0, len(items))
			for _, item := range items {
				out = append(out, seo.RepoInfo{
					Name:           item.Name,
					Description:    item.Description,
					RepoPath:       item.RepoPath,
					IsPrivate:      item.IsPrivate,
					LastActivityAt: item.LastActivityAt,
				})
			}

			return out, nil
		},
	}
}
