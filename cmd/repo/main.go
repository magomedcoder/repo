package main

import (
	"log"
	"net/http"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/infrastructure/git"
	"github.com/magomedcoder/repo/internal/infrastructure/persistence/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/bcrypt"
	"github.com/magomedcoder/repo/pkg/token"
)

func main() {
	db, err := sqlite.NewDB("data/db.sqlite")
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	userStore := sqlite.NewUserStore(db)
	sessionStore := sqlite.NewSessionStore(db)
	repoStore := sqlite.NewRepositoryStore(db)
	gitRepo := git.NewRepository()
	hasher := bcrypt.NewHasher()
	tokens := token.NewGenerator()

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, hasher, tokens)
	repoUC := usecase.NewRepositoryUseCase(repoStore, gitRepo)

	authHandler := handler.NewAuthHandler(authUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, authUC)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
