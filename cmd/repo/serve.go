package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"git.magomedcoder.ru/repo"
	"git.magomedcoder.ru/repo/internal/adapter/git"
	deliveryhttp "git.magomedcoder.ru/repo/internal/adapter/http"
	"git.magomedcoder.ru/repo/internal/adapter/http/handler"
	"git.magomedcoder.ru/repo/internal/adapter/http/middleware"
	"git.magomedcoder.ru/repo/internal/adapter/sqlite"
	"git.magomedcoder.ru/repo/internal/config"
	"git.magomedcoder.ru/repo/internal/usecase"
	"git.magomedcoder.ru/repo/pkg/argon2id"
	"git.magomedcoder.ru/repo/pkg/token"
)

func runServe() {
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := cfg.ValidateSecrets(); err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := sqlite.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	userStore := sqlite.NewUserStore(db)
	sessionStore := sqlite.NewSessionStore(db)
	folderStore := sqlite.NewFolderStore(db)
	repoStore := sqlite.NewRepositoryStore(db)
	tokenStore := sqlite.NewAccessTokenStore(db)
	issueStore := sqlite.NewIssueStore(db)
	pullStore := sqlite.NewPullRequestStore(db)
	activityStore := sqlite.NewActivityStore(db)
	gitRepo := git.NewRepository()
	gitRepo.Root = cfg.ReposRoot
	hasher := argon2id.NewHasher()
	tokens := token.NewGenerator()

	middleware.SetTrustedProxies(cfg.TrustedProxyNets)

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	authUC.SetSessionSecret(cfg.SessionSecret)
	authUC.SetAllowRegistration(cfg.AllowRegistration)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore, gitRepo, cfg.ReposRoot)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo, activityStore)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore)
	profileUC := usecase.NewProfileUseCase(userStore, cfg.AvatarsDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCookiePolicy(cfg.CookieSecure, http.SameSiteLaxMode)
	authHandler.SetAllowRegistration(cfg.AllowRegistration)
	folderHandler := handler.NewFolderHandler(folderUC)
	repoHandler := handler.NewRepositoryHandler(repoUC, cfg.ReposRoot)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	gitHandler := handler.NewGitHandler(gitUC)
	gitHandler.SetReceiveMaxBytes(cfg.ReceiveMaxBytes)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	activityHandler := handler.NewActivityHandler(activityUC)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, folderHandler, tokenHandler, gitHandler, issueHandler, pullHandler, profileHandler, activityHandler, authUC, cfg.CookieSecure)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           withStatic(cfg.DataDir, router),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Minute,
		WriteTimeout:      15 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func withStatic(dataDir string, api http.Handler) http.Handler {
	frontend, err := repo.WebHandler()
	if err != nil {
		log.Fatalf("web: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.IsGitHTTPPath(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/api") {
			api.ServeHTTP(w, r)
			return
		}

		if stringsHasData(r.URL.Path, dataDir) {
			http.NotFound(w, r)
			return
		}

		frontend.ServeHTTP(w, r)
	})
}

func stringsHasData(urlPath, dataDir string) bool {
	return dataDir != "" && (urlPath == "/"+dataDir || len(urlPath) > len(dataDir)+1 && urlPath[:len(dataDir)+2] == "/"+dataDir+"/")
}
