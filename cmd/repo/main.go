package main

import (
	"log"
	"net/http"
	"os/exec"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/infrastructure/git"
	"github.com/magomedcoder/repo/internal/infrastructure/persistence/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/bcrypt"
	"github.com/magomedcoder/repo/pkg/token"
)

func main() {
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatalf("git binary required for Smart HTTP: %v", err)
	}

	db, err := sqlite.NewDB("data/db.sqlite")
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	userStore := sqlite.NewUserStore(db)
	sessionStore := sqlite.NewSessionStore(db)
	folderStore := sqlite.NewFolderStore(db)
	repoStore := sqlite.NewRepositoryStore(db)
	tokenStore := sqlite.NewAccessTokenStore(db)
	gitRepo := git.NewRepository()
	hasher := bcrypt.NewHasher()
	tokens := token.NewGenerator()

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, gitRepo)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, tokenStore, hasher, gitRepo)

	authHandler := handler.NewAuthHandler(authUC)
	folderHandler := handler.NewFolderHandler(folderUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	gitHandler := handler.NewGitHandler(gitUC)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, folderHandler, tokenHandler, gitHandler, authUC)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
