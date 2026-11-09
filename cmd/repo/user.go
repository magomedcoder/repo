package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/magomedcoder/repo/internal/adapter/sqlite"
	"github.com/magomedcoder/repo/internal/config"
	"github.com/magomedcoder/repo/internal/domain"
	"github.com/magomedcoder/repo/internal/usecase"
	"github.com/magomedcoder/repo/pkg/argon2id"
	"github.com/magomedcoder/repo/pkg/token"
)

func runUser(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: repo user create ...")
	}

	switch args[0] {
	case "create":
		return runUserCreate(args[1:])
	default:
		return fmt.Errorf("unknown user command %q (want create)", args[0])
	}
}

func runUserCreate(args []string) error {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	username := fs.String("username", "", "username")
	email := fs.String("email", "", "email")
	password := fs.String("password", "", "password (min 8); discouraged - visible in process list. Prefer REPO_USER_PASSWORD env or stdin")
	roleFlag := fs.String("role", domain.RoleUser.String(), "role: user|admin")
	configPath := fs.String("config", config.DefaultPath, "path to config.yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" || *email == "" {
		return fmt.Errorf("required: -username -email (password via REPO_USER_PASSWORD or stdin)")
	}

	pw, err := resolveUserPassword(*password, os.Getenv, os.Stdin)
	if err != nil {
		return err
	}

	role, ok := domain.ParseRole(*roleFlag)
	if !ok {
		return fmt.Errorf("invalid -role %q (want user|admin)", *roleFlag)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := sqlite.NewDB(cfg.Server.DataDir)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}

	authUC := usecase.NewAuthUseCase(
		sqlite.NewUserRepo(db),
		sqlite.NewSessionRepo(db),
		sqlite.NewAccessTokenRepo(db),
		argon2id.NewHasher(),
		token.NewGenerator(),
	)
	authUC.SetSessionSecret(cfg.Secrets.SessionSecret)

	user, err := authUC.CreateUser(context.Background(), usecase.RegisterInput{
		Username: *username,
		Email:    *email,
		Password: pw,
		Role:     role,
	})
	if err != nil {
		return err
	}

	fmt.Printf("created user %s (id=%d role=%s)\n", user.Username, user.ID, user.Role)
	return nil
}

func resolveUserPassword(flagValue string, getenv func(string) string, stdin *os.File) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if v := getenv("REPO_USER_PASSWORD"); v != "" {
		return v, nil
	}

	if stdin == nil {
		return "", fmt.Errorf("password required: set REPO_USER_PASSWORD or pipe it on stdin")
	}

	if fi, err := stdin.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprint(os.Stderr, "Password: ")
	}

	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("read password from stdin: %w", err)
	}

	pw := strings.TrimRight(line, "\r\n")
	if pw == "" {
		return "", fmt.Errorf("password required: set REPO_USER_PASSWORD or pipe it on stdin")
	}

	return pw, nil
}
