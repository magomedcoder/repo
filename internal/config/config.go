package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr      string `yaml:"http_addr"`
	DataDir       string `yaml:"data_dir"`
	DBPath        string `yaml:"db_path"`
	ReposRoot     string `yaml:"repos_root"`
	AvatarsDir    string `yaml:"avatars_dir"`
	CookieSecure  bool   `yaml:"cookie_secure"`
	SessionSecret string `yaml:"-"`
	StaticDir     string `yaml:"static_dir"`
	OwnerUsername string `yaml:"owner_username"`
	OwnerEmail    string `yaml:"owner_email"`
	OwnerPassword string `yaml:"-"`
}

func Default() Config {
	return Config{
		HTTPAddr:   ":8080",
		DataDir:    "data",
		DBPath:     "data/db.sqlite",
		ReposRoot:  "data/repos",
		AvatarsDir: "data/avatars",
		StaticDir:  "web/dist",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		path = envOr("REPO_CONFIG", "config.yaml")
	}

	if _, err := os.Stat(path); err == nil {
		raw, err := os.ReadFile(path)
		if err != nil {
			return cfg, fmt.Errorf("read config: %w", err)
		}

		if err := yaml.Unmarshal(raw, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return cfg, err
	}

	cfg.HTTPAddr = envOr("REPO_HTTP_ADDR", cfg.HTTPAddr)
	cfg.DataDir = envOr("REPO_DATA_DIR", cfg.DataDir)
	cfg.DBPath = envOr("REPO_DB_PATH", cfg.DBPath)
	cfg.ReposRoot = envOr("REPO_REPOS_ROOT", cfg.ReposRoot)
	cfg.AvatarsDir = envOr("REPO_AVATARS_DIR", cfg.AvatarsDir)
	cfg.StaticDir = envOr("REPO_STATIC_DIR", cfg.StaticDir)
	cfg.OwnerUsername = envOr("REPO_OWNER_USERNAME", cfg.OwnerUsername)
	cfg.OwnerEmail = envOr("REPO_OWNER_EMAIL", cfg.OwnerEmail)
	secret, err := secretFromEnvOrFile("REPO_SESSION_SECRET", "REPO_SESSION_SECRET_FILE")
	if err != nil {
		return cfg, err
	}
	cfg.SessionSecret = secret
	password, err := secretFromEnvOrFile("REPO_OWNER_PASSWORD", "REPO_OWNER_PASSWORD_FILE")
	if err != nil {
		return cfg, err
	}
	cfg.OwnerPassword = password
	if v := strings.TrimSpace(os.Getenv("REPO_COOKIE_SECURE")); v == "1" || strings.EqualFold(v, "true") {
		cfg.CookieSecure = true
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	return cfg, nil
}

func secretFromEnvOrFile(envKey, fileKey string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v, nil
	}

	path := strings.TrimSpace(os.Getenv(fileKey))
	if path == "" {
		return "", nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", fileKey, err)
	}

	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("%s: mode must be 0600", fileKey)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", fileKey, err)
	}

	return strings.TrimSpace(string(raw)), nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}
