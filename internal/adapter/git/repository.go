package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/transport"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/fsutil"
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

	return nil
}

func (r *Repository) DetectHEADBranch(repoPath string) (string, error) {
	return detectHEADBranch(repoPath)
}

func detectHEADBranch(path string) (string, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return "", err
	}
	defer repo.Close()

	head, err := repo.Storer.Reference(plumbing.HEAD)
	if err == nil && head.Type() == plumbing.SymbolicReference {
		name := head.Target().String()
		return strings.TrimPrefix(name, "refs/heads/"), nil
	}

	iter, err := repo.Branches()
	if err != nil {
		return "", err
	}
	defer iter.Close()

	ref, err := iter.Next()
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(ref.Name().String(), "refs/heads/"), nil
}

func (r *Repository) Remove(path string) error {
	if path == "" {
		return nil
	}

	safe, err := fsutil.SafePath(r.Root, path)
	if err != nil {
		return err
	}

	defer r.LockRepo(safe)()

	if err := os.RemoveAll(safe); err != nil {
		return fmt.Errorf("remove repository: %w", err)
	}

	if r.opens != nil {
		r.opens.evict(safe)
	}
	if r.stats != nil {
		r.stats.evictPrefix(safe)
	}

	return nil
}

func (r *Repository) Move(oldPath, newPath string) error {
	if oldPath == newPath {
		return nil
	}

	oldSafe, err := fsutil.SafePath(r.Root, oldPath)
	if err != nil {
		return err
	}

	newSafe, err := fsutil.SafePath(r.Root, newPath)
	if err != nil {
		return err
	}

	defer r.lockRepos(oldSafe, newSafe)()

	if _, err := os.Stat(oldSafe); err != nil {
		return fmt.Errorf("source repository: %w", err)
	}

	if _, err := os.Stat(newSafe); !os.IsNotExist(err) {
		return fmt.Errorf("destination already exists: %s", newSafe)
	}

	if err := os.MkdirAll(filepath.Dir(newSafe), 0o700); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(oldSafe, newSafe); err != nil {
		return fmt.Errorf("move repository: %w", err)
	}

	if r.opens != nil {
		r.opens.evict(oldSafe)
	}
	if r.stats != nil {
		r.stats.evictPrefix(oldSafe)
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

const maxConcurrentServePacks = 32

var servePackSem = make(chan struct{}, maxConcurrentServePacks)

func (r *Repository) ServePack(ctx context.Context, repoPath, service string, stdin io.Reader, stdout io.Writer, receive *domain.ReceivePackOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case servePackSem <- struct{}{}:
		defer func() { <-servePackSem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return err
	}

	service = normalizeService(service)

	in := io.NopCloser(stdin)
	out := nopCloser{stdout}
	if strings.Contains(service, "receive") {
		req := &transport.ReceivePackRequest{StatelessRPC: true}
		if receive != nil && receive.ProtectDefaultBranch {
			req.Hooks.PreReceive = func(ctx context.Context, info *transport.PreReceiveInfo) error {
				return enforceProtectedBranches(ctx, info)
			}
		}

		err := transport.ReceivePack(ctx, repo.Storer, in, out, req)
		if err != nil && (errors.Is(err, domain.ErrBranchProtected) || strings.Contains(err.Error(), domain.ErrBranchProtected.Error())) {
			_ = r.pruneUnreachable(ctx, repoPath)
		}

		return err
	}

	return transport.UploadPack(ctx, repo.Storer, in, out, &transport.UploadPackRequest{
		StatelessRPC: true,
	})
}

func (r *Repository) pruneUnreachable(ctx context.Context, repoPath string) error {
	if !gitAvailable() {
		return nil
	}

	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return err
	}

	out, err := exec.CommandContext(ctx, "git", "-C", safe, "prune", "--expire=now").CombinedOutput()
	if err != nil {
		return fmt.Errorf("prune: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
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
