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
	minSearchQueryLen     = 2
	maxSearchQueryLen     = 200
	codeSearchTimeout     = 5 * time.Second
	codeSearchQueueWait   = 2 * time.Second
	maxConcurrentSearches = 2
)

var codeSearchSem = make(chan struct{}, maxConcurrentSearches)

type CodeSearchResult struct {
	RepoName string `json:"repo_name"`
	RepoPath string `json:"repo_path"`
	Owner    string `json:"owner"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Content  string `json:"content"`
}

type CodeSearchPage struct {
	Items      []CodeSearchResult `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
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

	return uc.mapReposWithOwners(ctx, repos)
}

func (uc *RepositoryUseCase) SearchCode(ctx context.Context, query string, viewerID uint, cursor string) (*CodeSearchPage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	q, err := normalizeSearchQuery(query)
	if err != nil {
		return nil, err
	}
	if q == "" {
		return &CodeSearchPage{Items: []CodeSearchResult{}}, nil
	}

	if uc.codeIndex == nil {
		return &CodeSearchPage{Items: []CodeSearchResult{}}, nil
	}

	tokens := tokenizeCodeQuery(q)
	if len(tokens) == 0 {
		return &CodeSearchPage{Items: []CodeSearchResult{}}, nil
	}

	afterRepoID, afterPath, err := decodeCodeCursor(cursor)
	if err != nil {
		return nil, err
	}

	wait := time.NewTimer(codeSearchQueueWait)
	select {
	case codeSearchSem <- struct{}{}:
		wait.Stop()
		defer func() { <-codeSearchSem }()
	case <-wait.C:
		return nil, ErrBusy
	case <-ctx.Done():
		wait.Stop()
		return nil, ctx.Err()
	}

	ctx, cancelSearch := context.WithTimeout(ctx, codeSearchTimeout)
	defer cancelSearch()

	fileHits, err := uc.codeIndex.Search(ctx, viewerID, tokens, afterRepoID, afterPath, maxCodePageFiles+1)
	if err != nil {
		return nil, fmt.Errorf("code index search: %w", err)
	}

	hasMoreFiles := len(fileHits) > maxCodePageFiles
	if hasMoreFiles {
		fileHits = fileHits[:maxCodePageFiles]
	}

	out := make([]CodeSearchResult, 0, maxCodePageHits)
	userCache := map[uint]string{}
	repoCache := map[uint]*domain.Repository{}
	var lastRepoID uint
	var lastPath string

	for _, fh := range fileHits {
		if len(out) >= maxCodePageHits {
			hasMoreFiles = true
			break
		}

		repo := repoCache[fh.RepoID]
		if repo == nil {
			r, rerr := uc.store.FindByID(ctx, fh.RepoID)
			if rerr != nil || r == nil {
				continue
			}
			repo = r
			repoCache[fh.RepoID] = repo
		}

		owner := userCache[repo.OwnerID]
		if owner == "" {
			if u, uerr := uc.users.FindByID(ctx, repo.OwnerID); uerr == nil {
				owner = u.Username
				userCache[repo.OwnerID] = owner
			}
		}

		remain := maxCodePageHits - len(out)
		perFile := min(remain, maxCodeHitsPerFile)
		lines := extractLineHits(fh.Path, fh.Body, q, perFile)
		if len(lines) == 0 {
			preview := sanitizeIndexedLine(firstNonEmptyLine(fh.Body))
			lines = []domain.CodeSearchHit{
				{
					Path:    fh.Path,
					Line:    1,
					Content: preview,
				},
			}
		}

		for _, hit := range lines {
			out = append(out, CodeSearchResult{
				RepoName: repo.Name,
				RepoPath: repo.RepoPath,
				Owner:    owner,
				Path:     hit.Path,
				Line:     hit.Line,
				Content:  hit.Content,
			})
		}
		lastRepoID = fh.RepoID
		lastPath = fh.Path
	}

	page := &CodeSearchPage{Items: out}
	if hasMoreFiles && lastRepoID != 0 {
		page.NextCursor = encodeCodeCursor(lastRepoID, lastPath)
	}

	return page, nil
}

func firstNonEmptyLine(body string) string {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}

	return ""
}
