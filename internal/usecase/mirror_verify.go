package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

type VerifyMirrorInput struct {
	ResolveRepositoryInput
}

type AdminMirrorVerifyItem struct {
	RepoID    uint   `json:"repo_id"`
	RepoPath  string `json:"repo_path"`
	OK        bool   `json:"ok"`
	Checked   int    `json:"checked"`
	DiffCount int    `json:"diff_count"`
	Error     string `json:"error,omitzero"`
}

func (uc *RepositoryUseCase) VerifyMirror(ctx context.Context, in VerifyMirrorInput) (*domain.MirrorVerifyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, _, err := uc.resolve(in.ResolveRepositoryInput)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanAdminRepo(repo, in.ViewerID) {
		return nil, ErrRepoNotFound
	}

	if !uc.mirrorLimiter().allow(repo.ID|checkLimiterBit, time.Now()) {
		return nil, ErrMirrorRateLimited
	}

	return uc.verifyMirrorRepo(ctx, repo)
}

func (uc *RepositoryUseCase) AdminVerifyMirrors(ctx context.Context, viewerID uint, filter domain.MirrorListFilter) ([]AdminMirrorVerifyItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := uc.requireSiteAdmin(ctx, viewerID); err != nil {
		return nil, err
	}

	if filter.Limit <= 0 || filter.Limit > maxAdminMirrorVerify {
		filter.Limit = maxAdminMirrorVerify
	}

	repos, err := uc.store.ListMirrors(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list mirrors: %w", err)
	}

	out := make([]AdminMirrorVerifyItem, 0, len(repos))
	for i := range repos {
		r := &repos[i]
		item := AdminMirrorVerifyItem{
			RepoID:   r.ID,
			RepoPath: r.RepoPath,
		}
		res, verr := uc.verifyMirrorRepo(ctx, r)
		if verr != nil {
			item.Error = truncateErr(uc.sanitizeMirrorErr(r, r.MirrorURL, verr), 500)
			out = append(out, item)
			continue
		}

		item.OK = res.OK
		item.Checked = res.Checked
		if res.TotalDiffs > 0 {
			item.DiffCount = res.TotalDiffs
		} else {
			item.DiffCount = len(res.Diffs)
		}

		if !res.OK {
			item.Error = formatVerifyDiffs(res)
		}

		out = append(out, item)
	}

	return out, nil
}

func (uc *RepositoryUseCase) verifyMirrorRepo(ctx context.Context, repo *domain.Repository) (*domain.MirrorVerifyResult, error) {
	if repo.MirrorDirection == domain.MirrorNone || strings.TrimSpace(repo.MirrorURL) == "" {
		return nil, ErrMirrorNotConfigured
	}

	opts, err := uc.mirrorOptions(repo)
	if err != nil {
		return nil, err
	}

	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	res, err := uc.git.MirrorVerify(cctx, repo.Path, repo.MirrorURL, repo.MirrorDirection, opts)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return &domain.MirrorVerifyResult{OK: true}, nil
	}

	return res, nil
}

func formatVerifyDiffs(res *domain.MirrorVerifyResult) string {
	if res == nil || len(res.Diffs) == 0 {
		return ""
	}

	total := res.TotalDiffs
	if total == 0 {
		total = len(res.Diffs)
	}

	parts := make([]string, 0, len(res.Diffs))
	for i, d := range res.Diffs {
		if i >= 5 {
			parts = append(parts, fmt.Sprintf("...+%d more", total-5))
			break
		}

		parts = append(parts, d.Name+":"+d.Reason)
	}

	return "ref mismatch: " + strings.Join(parts, ", ")
}
