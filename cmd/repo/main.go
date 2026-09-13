package main

import (
	"log"
	"net/http"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	"github.com/magomedcoder/repo/internal/infrastructure/git"
	"github.com/magomedcoder/repo/internal/infrastructure/persistence/sqlite"
	"github.com/magomedcoder/repo/internal/usecase"
)

func main() {
	db, err := sqlite.NewDB("data/db.sqlite")
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	store := sqlite.NewRepositoryStore(db)
	gitRepo := git.NewRepository()
	createUC := usecase.NewCreateUseCase(store, gitRepo)
	repoHandler := handler.NewRepositoryHandler(createUC)

	router := deliveryhttp.NewRouter(repoHandler)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("server: %v", err)
	}
}
