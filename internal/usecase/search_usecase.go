package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"magomedcoder.ru/repo/internal/domain"
)

const (
	minSearchQueryLen      = 2
	maxSearchQueryLen      = 200
	maxCodeScanRepos       = 40
	maxCodeHitsPerRepo     = 3
	maxCodeHitsTotal       = 50
	codeGrepTimeoutPerRepo = 1500 * time.Millisecond
)

type CodeSearchResult struct {
	RepoName string `json:"repo_name"`
	RepoPath string `json:"repo_path"`
	Owner    string `json:"owner"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Content  string `json:"content"`
}

func normalizeSearchQuery(raw string) (string, error) {
	q := strings.TrimSpace(raw)
	if q == "" {
		return "", nil
	}

	if utf8.RuneCountInString(q) < minSearchQueryLen {
		return "", NewError(KindValidation, "search_query_too_short")
	}

	if len(q) > maxSearchQueryLen {
		return "", NewError(KindValidation, "search_query_too_long")
	}

	return q, nil
}

func (uc *RepositoryUseCase) SearchAccessibleRepos(ctx context.Context, query string, viewerID uint, limit, offset int) ([]RepositoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	q, err := normalizeSearchQuery(query)
	if err != nil {
		return nil, err
	}

	if q == "" {
		return nil, nil
	}

	offset, limit = NormalizePagination(offset, limit)
	repos, err := uc.store.SearchAccessible(ctx, q, viewerID, limit, offset, domain.RepoOrderUpdated)
	if err != nil {
		return nil, fmt.Errorf("search accessible repositories: %w", err)
	}

	return uc.mapReposWithOwners(repos)
}

func (uc *RepositoryUseCase) SearchCode(ctx context.Context, query string, viewerID uint) ([]CodeSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	q, err := normalizeSearchQuery(query)
	if err != nil {
		return nil, err
	}

	if q == "" {
		return nil, nil
	}

	candidates, err := uc.store.ListAccessible(ctx, viewerID, maxCodeScanRepos, 0, domain.RepoOrderUpdated)
	if err != nil {
		return nil, fmt.Errorf("list accessible repositories: %w", err)
	}

	out := make([]CodeSearchResult, 0, maxCodeHitsTotal)
	userCache := map[uint]string{}

	for _, repo := range candidates {
		if len(out) >= maxCodeHitsTotal {
			break
		}

		if err := ctx.Err(); err != nil {
			return out, err
		}

		remain := maxCodeHitsTotal - len(out)
		perRepo := min(remain, maxCodeHitsPerRepo)

		grepCtx, cancel := context.WithTimeout(ctx, codeGrepTimeoutPerRepo)
		hits, gerr := uc.git.Grep(grepCtx, repo.Path, q, perRepo)
		cancel()
		if gerr != nil {
			continue
		}

		if len(hits) == 0 {
			continue
		}

		owner := userCache[repo.OwnerID]
		if owner == "" {
			if u, uerr := uc.users.FindByID(ctx, repo.OwnerID); uerr == nil {
				owner = u.Username
				userCache[repo.OwnerID] = owner
			}
		}

		for _, hit := range hits {
			out = append(out, CodeSearchResult{
				RepoName: repo.Name,
				RepoPath: repo.RepoPath,
				Owner:    owner,
				Path:     hit.Path,
				Line:     hit.Line,
				Content:  hit.Content,
			})
			if len(out) >= maxCodeHitsTotal {
				break
			}
		}
	}

	return out, nil
}
