package usecase

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrNameRequired  = errors.New("repository name required")
	ErrAlreadyExists = errors.New("repository already exists")
	ErrInvalidName   = errors.New("invalid repository name")
)

type CreateRepositoryInput struct {
	Name        string
	Description string
	OwnerID     uint
	BasePath    string
}

type CreateRepositoryOutput struct {
	ID          uint
	Name        string
	Description string
}

type RepositoryItem struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RepositoryUseCase struct {
	store domain.RepositoryStore
	git   domain.GitRepository
}

func NewRepositoryUseCase(store domain.RepositoryStore, git domain.GitRepository) *RepositoryUseCase {
	return &RepositoryUseCase{
		store: store,
		git:   git,
	}
}

func (uc *RepositoryUseCase) Create(in CreateRepositoryInput) (*CreateRepositoryOutput, error) {
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

	return &CreateRepositoryOutput{
		ID:          repo.ID,
		Name:        repo.Name,
		Description: repo.Description,
	}, nil
}

func (uc *RepositoryUseCase) List(ownerID uint) ([]RepositoryItem, error) {
	if ownerID == 0 {
		return nil, ErrUnauthorized
	}

	repos, err := uc.store.ListByOwnerID(ownerID)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}

	items := make([]RepositoryItem, 0, len(repos))
	for _, repo := range repos {
		items = append(items, RepositoryItem{
			ID:          repo.ID,
			Name:        repo.Name,
			Description: repo.Description,
			CreatedAt:   repo.CreatedAt,
			UpdatedAt:   repo.UpdatedAt,
		})
	}

	return items, nil
}
