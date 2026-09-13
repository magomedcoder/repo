package usecase

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrNameRequired  = errors.New("repository name required")
	ErrAlreadyExists = errors.New("repository already exists")
	ErrInvalidName   = errors.New("invalid repository name")
)

type CreateInput struct {
	Name        string
	Description string
	OwnerID     uint
	BasePath    string
}

type CreateOutput struct {
	ID   uint
	Name string
	Path string
}

type CreateUseCase struct {
	store domain.RepositoryStore
	git   domain.GitRepository
}

func NewCreateUseCase(store domain.RepositoryStore, git domain.GitRepository) *CreateUseCase {
	return &CreateUseCase{
		store: store,
		git:   git,
	}
}

func (uc *CreateUseCase) Execute(in CreateInput) (*CreateOutput, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return nil, ErrInvalidName
	}

	exists, err := uc.store.ExistsByName(name)
	if err != nil {
		return nil, fmt.Errorf("check repository existence: %w", err)
	}
	if exists {
		return nil, ErrAlreadyExists
	}

	ownerID := in.OwnerID
	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	basePath := in.BasePath
	if basePath == "" {
		basePath = filepath.Join("data", "repos")
	}

	repoPath := filepath.Join(basePath, fmt.Sprintf("%d", ownerID), name+".git")

	if err := uc.git.InitBare(repoPath); err != nil {
		return nil, fmt.Errorf("init bare repository: %w", err)
	}

	repo := &domain.Repository{
		Name:        name,
		OwnerID:     ownerID,
		Description: in.Description,
		Path:        repoPath,
	}
	if err := uc.store.Create(repo); err != nil {
		return nil, fmt.Errorf("save repository metadata: %w", err)
	}

	return &CreateOutput{
		ID:   repo.ID,
		Name: repo.Name,
		Path: repo.Path,
	}, nil
}
