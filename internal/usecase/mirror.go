package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrInvalidMirrorDirection = NewError(KindValidation, "invalid_mirror_direction")
	ErrMirrorNotConfigured    = NewError(KindValidation, "mirror_not_configured")
	ErrMirrorReadOnly         = NewError(KindForbidden, "mirror_read_only")
)

type ConfigureMirrorInput struct {
	ViewerID      uint
	FolderPath    string
	Name          string
	Direction     string
	URL           string
	Username      *string
	Password      *string
	ClearPassword bool
	Enabled       bool
}

func (uc *RepositoryUseCase) ConfigureMirror(ctx context.Context, in ConfigureMirrorInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, folderPath, err := uc.resolve(ResolveRepositoryInput{
		FolderPath: in.FolderPath,
		Name:       in.Name,
		ViewerID:   in.ViewerID,
	})
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if !in.Enabled {
		repo.MirrorDirection = domain.MirrorNone
		repo.MirrorURL = ""
		repo.MirrorUsername = ""
		repo.MirrorPassword = ""
		repo.MirrorLastSyncAt = nil
		repo.MirrorLastError = ""
		if err := uc.store.Update(repo); err != nil {
			return nil, fmt.Errorf("clear mirror: %w", err)
		}

		item := uc.toRepositoryItem(repo, ownerSlug, folderPath, in.ViewerID)

		return &item, nil
	}

	direction := strings.ToLower(strings.TrimSpace(in.Direction))
	if direction != domain.MirrorPull && direction != domain.MirrorPush {
		return nil, ErrInvalidMirrorDirection
	}

	mirrorURL := strings.TrimSpace(in.URL)
	if err := validateMirrorURL(mirrorURL); err != nil {
		return nil, err
	}

	repo.MirrorDirection = direction
	repo.MirrorURL = mirrorURL
	if in.Username != nil {
		repo.MirrorUsername = strings.TrimSpace(*in.Username)
	}

	if in.ClearPassword {
		repo.MirrorPassword = ""
	} else if in.Password != nil {
		repo.MirrorPassword = *in.Password
	}

	repo.MirrorLastError = ""

	if err := uc.store.Update(repo); err != nil {
		return nil, fmt.Errorf("save mirror: %w", err)
	}

	item := uc.toRepositoryItem(repo, ownerSlug, folderPath, in.ViewerID)

	return &item, nil
}

func (uc *RepositoryUseCase) SyncMirror(ctx context.Context, in ResolveRepositoryInput) (*RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, ownerSlug, folderPath, err := uc.resolve(in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if repo.MirrorDirection == domain.MirrorNone || strings.TrimSpace(repo.MirrorURL) == "" {
		return nil, ErrMirrorNotConfigured
	}

	lock := uc.mirrorLock(repo.ID)
	lock.Lock()
	defer lock.Unlock()

	fresh, ferr := uc.store.FindByID(ctx, repo.ID)
	if ferr != nil || fresh == nil {
		return nil, ErrRepoNotFound
	}

	if err := uc.runMirrorSync(ctx, fresh); err != nil {
		if errors.Is(err, ErrMirrorFailed) {
			return nil, ErrMirrorFailed
		}
		return nil, err
	}

	item := uc.toRepositoryItem(fresh, ownerSlug, folderPath, in.ViewerID)

	return &item, nil
}

func mirrorAuthFromRepo(repo *domain.Repository) *domain.MirrorAuth {
	if strings.TrimSpace(repo.MirrorUsername) == "" && repo.MirrorPassword == "" {
		return nil
	}

	return &domain.MirrorAuth{
		Username: strings.TrimSpace(repo.MirrorUsername),
		Password: repo.MirrorPassword,
	}
}

func truncateErr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}

	return s[:n]
}

func normalizeMirrorDirection(raw string) (string, error) {
	d := strings.ToLower(strings.TrimSpace(raw))
	switch d {
	case "":
		return domain.MirrorNone, nil
	case domain.MirrorPull:
		return domain.MirrorPull, nil
	case domain.MirrorPush:
		return domain.MirrorPush, nil
	default:
		return "", ErrInvalidMirrorDirection
	}
}
