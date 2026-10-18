package main

import (
	"log"
	"net/http"
	"os"
	"time"

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

func main() {
	cfg, err := config.Load("")
	if err != nil {
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

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	authUC.SetSessionSecret(cfg.SessionSecret)
	if err := authUC.EnsureOwner(cfg.OwnerUsername, cfg.OwnerEmail, cfg.OwnerPassword); err != nil {
		log.Fatalf("owner: %v", err)
	}
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo, activityStore)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo, activityStore)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, activityStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, gitRepo, activityStore)
	profileUC := usecase.NewProfileUseCase(userStore, cfg.AvatarsDir)
	activityUC := usecase.NewActivityUseCase(activityStore, userStore, repoStore)

	authHandler := handler.NewAuthHandler(authUC)
	authHandler.SetCookiePolicy(cfg.CookieSecure, http.SameSiteLaxMode)
	middleware.CookieSecure = cfg.CookieSecure
	folderHandler := handler.NewFolderHandler(folderUC)
	repoHandler := handler.NewRepositoryHandler(repoUC, cfg.ReposRoot)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	gitHandler := handler.NewGitHandler(gitUC)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	activityHandler := handler.NewActivityHandler(activityUC)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, folderHandler, tokenHandler, gitHandler, issueHandler, pullHandler, profileHandler, activityHandler, authUC)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           withStatic(cfg.StaticDir, cfg.DataDir, router),
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

func withStatic(dir, dataDir string, api http.Handler) http.Handler {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return api
	}

	files := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler.IsGitHTTPPath(r.URL.Path) || len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			api.ServeHTTP(w, r)
			return
		}

		if stringsHasData(r.URL.Path, dataDir) {
			http.NotFound(w, r)
			return
		}

		files.ServeHTTP(w, r)
	})
}

func stringsHasData(urlPath, dataDir string) bool {
	return dataDir != "" && (urlPath == "/"+dataDir || len(urlPath) > len(dataDir)+1 && urlPath[:len(dataDir)+2] == "/"+dataDir+"/")
}
