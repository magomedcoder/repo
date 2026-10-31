package git

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"magomedcoder.ru/repo/internal/domain"
)

type Repository struct {
	Root  string
	opens *openCache
	stats *statsCache
}

func NewRepository(dataDir string) *Repository {
	return &Repository{
		Root:  filepath.Join(dataDir, "repos"),
		opens: newOpenCache(32),
		stats: newStatsCache(),
	}
}

func (r *Repository) GetReposRoot() string {
	return r.Root
}

func (r *Repository) InitBare(path string, opts domain.BareInitOptions) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("repository already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if _, err := gogit.PlainInit(path, true); err != nil {
		return fmt.Errorf("init bare repository: %w", err)
	}

	if err := os.Chmod(path, 0o700); err != nil {
		_ = os.RemoveAll(path)
		return err
	}

	branch := opts.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	if err := os.WriteFile(filepath.Join(path, "HEAD"), []byte("ref: refs/heads/"+branch+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(path)
		return err
	}

	if opts.DenyForcePushDefault {
		if err := r.SetReceiveDenyNonFastForwards(path, true); err != nil {
			_ = os.RemoveAll(path)
			return err
		}
	}

	return nil
}

func (r *Repository) SetReceiveDenyNonFastForwards(path string, deny bool) error {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open repository: %w", err)
	}

	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	val := "false"
	if deny {
		val = "true"
	}

	if cfg.Raw == nil {
		if _, err := cfg.Marshal(); err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
	}

	cfg.Raw.Section("receive").SetOption("denyNonFastForwards", val)
	if err := repo.SetConfig(cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
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

	if safe, serr := SafePath(r.Root, path); serr == nil {
		if r.opens != nil {
			r.opens.evict(safe)
		}
		if r.stats != nil {
			r.stats.evictPrefix(safe)
		}
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

	if err := os.MkdirAll(filepath.Dir(newPath), 0o700); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("move repository: %w", err)
	}

	if safe, serr := SafePath(r.Root, oldPath); serr == nil {
		if r.opens != nil {
			r.opens.evict(safe)
		}
		if r.stats != nil {
			r.stats.evictPrefix(safe)
		}
	}

	return nil
}

func (r *Repository) AdvertiseRefs(ctx context.Context, repoPath, service string, w io.Writer) error {
	repo, err := r.open(repoPath)
	if err != nil {
		return err
	}

	service = normalizeService(service)
	if ctx == nil {
		ctx = context.Background()
	}
	empty := io.NopCloser(strings.NewReader(""))
	out := nopCloser{w}
	if strings.Contains(service, "receive") {
		return transport.ReceivePack(ctx, repo.Storer, empty, out, &transport.ReceivePackRequest{
			AdvertiseRefs: true,
			StatelessRPC:  true,
		})
	}

	return transport.UploadPack(ctx, repo.Storer, empty, out, &transport.UploadPackRequest{
		AdvertiseRefs: true,
		StatelessRPC:  true,
	})
}

func (r *Repository) ServePack(ctx context.Context, repoPath, service string, stdin io.Reader, stdout io.Writer) error {
	repo, err := r.open(repoPath)
	if err != nil {
		return err
	}

	service = normalizeService(service)
	if ctx == nil {
		ctx = context.Background()
	}
	in := io.NopCloser(stdin)
	out := nopCloser{stdout}
	if strings.Contains(service, "receive") {
		return transport.ReceivePack(ctx, repo.Storer, in, out, &transport.ReceivePackRequest{
			StatelessRPC: true,
		})
	}

	return transport.UploadPack(ctx, repo.Storer, in, out, &transport.UploadPackRequest{
		StatelessRPC: true,
	})
}

func (r *Repository) EnsureDefaultBranch(repoPath, branch string) error {
	repo, err := r.open(repoPath)
	if err != nil {
		return err
	}

	if branch == "" {
		branch = "main"
	}

	head, err := repo.Storer.Reference(plumbing.HEAD)
	if err == nil && head.Type() == plumbing.SymbolicReference {
		if _, rerr := repo.Storer.Reference(head.Target()); rerr == nil {
			return nil
		}
	}

	name := plumbing.ReferenceName("refs/heads/" + branch)
	if _, err := repo.Storer.Reference(name); err != nil {
		iter, ierr := repo.Branches()
		if ierr != nil {
			return nil
		}

		defer iter.Close()
		ref, nerr := iter.Next()
		if nerr != nil {
			return nil
		}
		name = ref.Name()
	}

	return repo.Storer.SetReference(plumbing.NewSymbolicReference(plumbing.HEAD, name))
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

func normalizeService(service string) string {
	service = strings.TrimSpace(service)
	switch service {
	case "upload-pack", "git-upload-pack":
		return "git-upload-pack"
	case "receive-pack", "git-receive-pack":
		return "git-receive-pack"
	default:
		return service
	}
}
