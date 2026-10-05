package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	deliveryhttp "github.com/magomedcoder/repo/internal/delivery/http"
	"github.com/magomedcoder/repo/internal/delivery/http/handler"
	deliveryssh "github.com/magomedcoder/repo/internal/delivery/ssh"
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
	sshKeyStore := sqlite.NewSSHKeyStore(db)
	issueStore := sqlite.NewIssueStore(db)
	pullStore := sqlite.NewPullRequestStore(db)
	orgStore := sqlite.NewOrganizationStore(db)
	gitRepo := git.NewRepository()
	hasher := bcrypt.NewHasher()
	tokens := token.NewGenerator()

	authUC := usecase.NewAuthUseCase(userStore, sessionStore, orgStore, hasher, tokens)
	folderUC := usecase.NewFolderUseCase(folderStore, repoStore)
	repoUC := usecase.NewRepositoryUseCase(repoStore, folderStore, userStore, orgStore, gitRepo)
	tokenUC := usecase.NewTokenUseCase(tokenStore, tokens)
	sshKeyUC := usecase.NewSSHKeyUseCase(sshKeyStore, userStore)
	gitUC := usecase.NewGitUseCase(repoStore, folderStore, userStore, orgStore, tokenStore, hasher, gitRepo)
	issueUC := usecase.NewIssueUseCase(issueStore, repoStore, folderStore, userStore, orgStore)
	pullUC := usecase.NewPullRequestUseCase(pullStore, repoStore, folderStore, userStore, orgStore, gitRepo)
	profileUC := usecase.NewProfileUseCase(userStore, "data/avatars")
	orgUC := usecase.NewOrganizationUseCase(orgStore, userStore, repoStore)

	authHandler := handler.NewAuthHandler(authUC)
	folderHandler := handler.NewFolderHandler(folderUC)
	repoHandler := handler.NewRepositoryHandler(repoUC)
	tokenHandler := handler.NewTokenHandler(tokenUC)
	sshKeyHandler := handler.NewSSHKeyHandler(sshKeyUC)
	gitHandler := handler.NewGitHandler(gitUC)
	issueHandler := handler.NewIssueHandler(issueUC)
	pullHandler := handler.NewPullRequestHandler(pullUC)
	profileHandler := handler.NewProfileHandler(profileUC)
	orgHandler := handler.NewOrganizationHandler(orgUC)

	router := deliveryhttp.NewRouter(authHandler, repoHandler, folderHandler, tokenHandler, sshKeyHandler, gitHandler, issueHandler, pullHandler, profileHandler, orgHandler, authUC)

	sshAddr := envOr("REPO_SSH_ADDR", ":2222")
	sshServer := deliveryssh.NewServer(sshKeyUC, gitUC, "data/ssh", sshAddr)
	go func() {
		if err := sshServer.ListenAndServe(); err != nil {
			log.Fatalf("ssh: %v", err)
		}
	}()

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
