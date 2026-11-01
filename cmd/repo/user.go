package main

import (
	"context"
	"flag"
	"fmt"

	"magomedcoder.ru/repo/internal/adapter/sqlite"
	"magomedcoder.ru/repo/internal/config"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/internal/usecase"
	"magomedcoder.ru/repo/pkg/argon2id"
	"magomedcoder.ru/repo/pkg/token"
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
	password := fs.String("password", "", "password (min 8)")
	roleFlag := fs.String("role", domain.RoleUser.String(), "role: user|admin")
	configPath := fs.String("config", config.DefaultPath, "path to config.yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" || *email == "" || *password == "" {
		return fmt.Errorf("required: -username -email -password")
	}

	role, ok := domain.ParseRole(*roleFlag)
	if !ok {
		return fmt.Errorf("invalid -role %q (want user|admin)", *roleFlag)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := sqlite.NewDB(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}

	authUC := usecase.NewAuthUseCase(
		sqlite.NewUserStore(db),
		sqlite.NewSessionStore(db),
		argon2id.NewHasher(),
		token.NewGenerator(),
	)
	authUC.SetSessionSecret(cfg.SessionSecret)

	user, err := authUC.CreateUser(context.Background(), usecase.RegisterInput{
		Username: *username,
		Email:    *email,
		Password: *password,
		Role:     role,
	})
	if err != nil {
		return err
	}

	fmt.Printf("created user %s (id=%d role=%s)\n", user.Username, user.ID, user.Role)
	return nil
}
