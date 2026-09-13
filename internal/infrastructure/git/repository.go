package git

import (
	"fmt"
	"os"
	"path/filepath"

	gogit "github.com/go-git/go-git/v6"
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) InitBare(path string) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("repository already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if _, err := gogit.PlainInit(path, true); err != nil {
		return fmt.Errorf("init bare repository: %w", err)
	}

	return nil
}
