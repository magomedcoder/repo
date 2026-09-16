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

func (r *Repository) Remove(path string) error {
	if path == "" {
		return nil
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove repository: %w", err)
	}

	return nil
}

func (r *Repository) Move(oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}

	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("source repository: %w", err)
	}

	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		return fmt.Errorf("destination already exists: %s", newPath)
	}

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("move repository: %w", err)
	}

	return nil
}
