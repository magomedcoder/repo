package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"magomedcoder.ru/repo/internal/domain"
)

const (
	codeIndexDebounce      = 2 * time.Second
	codeIndexJobTimeout    = 2 * time.Minute
	codeIndexBackfillEvery = time.Minute
	codeIndexBackfillBatch = 20
	maxCodePageFiles       = 40
	maxCodePageHits        = 50
	maxCodeHitsPerFile     = 3
	maxIndexedLineLen      = 240
)

type codeIndexRuntime struct {
	mu       sync.Mutex
	pending  map[uint]*time.Timer
	inflight map[uint]bool
}

func (uc *RepositoryUseCase) ensureCodeIndexRT() *codeIndexRuntime {
	if uc == nil {
		return nil
	}

	if uc.codeIdxRT == nil {
		uc.codeIdxRT = &codeIndexRuntime{
			pending:  make(map[uint]*time.Timer),
			inflight: make(map[uint]bool),
		}
	}

	return uc.codeIdxRT
}

func (uc *RepositoryUseCase) SetCodeIndex(store domain.CodeIndexStore) {
	if uc == nil {
		return
	}
	uc.codeIndex = store
	_ = uc.ensureCodeIndexRT()
}

func (uc *RepositoryUseCase) ScheduleCodeIndex(repoID uint) {
	if uc == nil || uc.codeIndex == nil || repoID == 0 {
		return
	}
	rt := uc.ensureCodeIndexRT()

	rt.mu.Lock()
	defer rt.mu.Unlock()

	if t, ok := rt.pending[repoID]; ok {
		t.Stop()
	}

	rt.pending[repoID] = time.AfterFunc(codeIndexDebounce, func() {
		rt.mu.Lock()
		delete(rt.pending, repoID)
		rt.mu.Unlock()
		uc.runCodeIndex(context.Background(), repoID)
	})
}

func (uc *RepositoryUseCase) DropCodeIndex(ctx context.Context, repoID uint) {
	if uc == nil || uc.codeIndex == nil || repoID == 0 {
		return
	}

	if err := uc.codeIndex.DeleteRepoIndex(ctx, repoID); err != nil {
		slog.Warn("code index delete", "repo_id", repoID, "err", err)
	}
}

func (uc *RepositoryUseCase) RunCodeIndexBackfill(ctx context.Context) {
	if uc == nil || uc.codeIndex == nil {
		return
	}
	ticker := time.NewTicker(codeIndexBackfillEvery)
	defer ticker.Stop()

	uc.backfillCodeIndexOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			uc.backfillCodeIndexOnce(ctx)
		}
	}
}

func (uc *RepositoryUseCase) backfillCodeIndexOnce(ctx context.Context) {
	ids, err := uc.codeIndex.ListStaleOrMissing(ctx, codeIndexBackfillBatch)
	if err != nil {
		slog.Warn("code index backfill list", "err", err)
		return
	}

	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		uc.ScheduleCodeIndex(id)
	}
}

func (uc *RepositoryUseCase) runCodeIndex(parent context.Context, repoID uint) {
	if uc.codeIndex == nil {
		return
	}
	rt := uc.ensureCodeIndexRT()

	rt.mu.Lock()
	if rt.inflight[repoID] {
		rt.mu.Unlock()
		uc.ScheduleCodeIndex(repoID)
		return
	}

	rt.inflight[repoID] = true
	rt.mu.Unlock()

	defer func() {
		rt.mu.Lock()
		delete(rt.inflight, repoID)
		rt.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(parent, codeIndexJobTimeout)
	defer cancel()

	repo, err := uc.store.FindByID(ctx, repoID)
	if err != nil || repo == nil {
		uc.DropCodeIndex(ctx, repoID)
		return
	}

	tipSHA, walkFiles, err := uc.git.WalkTextFiles(ctx, repo.Path)
	if err != nil {
		slog.Warn("code index walk", "repo_id", repoID, "err", err)
		return
	}

	if tipSHA != "" {
		if meta, merr := uc.codeIndex.Meta(ctx, repoID); merr == nil && meta != nil && meta.TipSHA == tipSHA {
			return
		}
	}

	files := make([]domain.CodeIndexFile, 0, len(walkFiles))
	for _, f := range walkFiles {
		files = append(files, domain.CodeIndexFile{
			RepoID: repoID,
			Path:   f.Path,
			TipSHA: tipSHA,
			Body:   f.Body,
		})
	}

	if err := uc.codeIndex.ReplaceRepoIndex(ctx, repoID, tipSHA, files); err != nil {
		slog.Warn("code index replace", "repo_id", repoID, "err", err)
		return
	}

	slog.Debug("code index updated", "repo_id", repoID, "tip", tipSHA, "files", len(files))
}

func tokenizeCodeQuery(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}

		tok := b.String()
		b.Reset()
		if utf8.RuneCountInString(tok) < 1 {
			return
		}

		tokens = append(tokens, tok)
	}

	for _, r := range raw {
		if unicode.IsSpace(r) || r == '"' || r == '*' || r == '(' || r == ')' || r == ':' || r == '^' {
			flush()
			continue
		}
		b.WriteRune(r)
	}
	flush()

	return tokens
}

func encodeCodeCursor(repoID uint, path string) string {
	raw := fmt.Sprintf("%d\n%s", repoID, path)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCodeCursor(cursor string) (repoID uint, path string, err error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, "", nil
	}

	raw, decErr := base64.RawURLEncoding.DecodeString(cursor)
	if decErr != nil {
		return 0, "", NewError(KindValidation, "invalid_search_cursor")
	}

	parts := strings.SplitN(string(raw), "\n", 2)
	if len(parts) != 2 {
		return 0, "", NewError(KindValidation, "invalid_search_cursor")
	}

	var id uint
	if _, scanErr := fmt.Sscanf(parts[0], "%d", &id); scanErr != nil || id == 0 {
		return 0, "", NewError(KindValidation, "invalid_search_cursor")
	}

	return id, parts[1], nil
}

func extractLineHits(path, body, query string, limit int) []domain.CodeSearchHit {
	if limit <= 0 {
		return nil
	}

	lowerQ := strings.ToLower(query)
	lines := strings.Split(body, "\n")
	out := make([]domain.CodeSearchHit, 0, limit)
	for i, line := range lines {
		if !strings.Contains(line, query) && !strings.Contains(strings.ToLower(line), lowerQ) {
			continue
		}

		out = append(out, domain.CodeSearchHit{
			Path:    path,
			Line:    i + 1,
			Content: sanitizeIndexedLine(line),
		})
		if len(out) >= limit {
			break
		}
	}

	return out
}

func sanitizeIndexedLine(s string) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "\uFFFD")
	}

	s = strings.ReplaceAll(s, "\t", " ")
	if len(s) > maxIndexedLineLen {
		s = s[:maxIndexedLineLen] + "..."
	}

	return s
}
