package domain

import "context"

type CodeIndexFile struct {
	RepoID uint
	Path   string
	TipSHA string
	Body   string
}

type CodeIndexHit struct {
	RepoID uint
	Path   string
	TipSHA string
	Body   string
}

type CodeIndexMeta struct {
	RepoID    uint
	TipSHA    string
	UpdatedAt int64
}

type CodeIndexStore interface {
	EnsureSchema(ctx context.Context) error

	ReplaceRepoIndex(ctx context.Context, repoID uint, tipSHA string, files []CodeIndexFile) error

	DeleteRepoIndex(ctx context.Context, repoID uint) error

	Search(ctx context.Context, viewerID uint, tokens []string, afterRepoID uint, afterPath string, limit int) ([]CodeIndexHit, error)

	Meta(ctx context.Context, repoID uint) (*CodeIndexMeta, error)

	ListStaleOrMissing(ctx context.Context, limit int) ([]uint, error)
}
