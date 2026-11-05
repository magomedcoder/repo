package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/config"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/client"
	"github.com/go-git/go-git/v6/plumbing/transport"
	githttp "github.com/go-git/go-git/v6/plumbing/transport/http"
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

func (r *Repository) MirrorClone(ctx context.Context, path, url string, auth *domain.MirrorAuth) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("mirror url required")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return "", fmt.Errorf("repository already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}

	opts := &gogit.CloneOptions{
		URL:           url,
		Mirror:        true,
		ClientOptions: mirrorClientOpts(auth),
	}

	repo, err := gogit.PlainCloneContext(ctx, path, opts)
	if err != nil {
		_ = os.RemoveAll(path)
		return "", fmt.Errorf("mirror clone: %w", err)
	}
	_ = repo.Close()

	if err := os.Chmod(path, 0o700); err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}

	branch, _ := detectHEADBranch(path)
	return branch, nil
}

func (r *Repository) MirrorFetch(ctx context.Context, path, url string, auth *domain.MirrorAuth) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("mirror url required")
	}

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open mirror: %w", err)
	}
	defer repo.Close()

	err = repo.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName:    "origin",
		RemoteURL:     url,
		RefSpecs:      []config.RefSpec{"+refs/*:refs/*"},
		Force:         true,
		Prune:         true,
		Tags:          gogit.AllTags,
		ClientOptions: mirrorClientOpts(auth),
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) && !errors.Is(err, transport.ErrEmptyRemoteRepository) {
		return fmt.Errorf("mirror fetch: %w", err)
	}
	return nil
}

func (r *Repository) MirrorPush(ctx context.Context, path, url string, auth *domain.MirrorAuth) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("mirror url required")
	}

	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return fmt.Errorf("open mirror: %w", err)
	}
	defer repo.Close()

	err = repo.PushContext(ctx, &gogit.PushOptions{
		RemoteName:    "origin",
		RemoteURL:     url,
		RefSpecs:      []config.RefSpec{"+refs/*:refs/*"},
		Force:         true,
		Prune:         true,
		ClientOptions: mirrorClientOpts(auth),
	})
	if err != nil && !errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return fmt.Errorf("mirror push: %w", err)
	}
	return nil
}

func mirrorClientOpts(auth *domain.MirrorAuth) []client.Option {
	if auth == nil || (auth.Username == "" && auth.Password == "") {
		return nil
	}
	user := auth.Username
	if user == "" {
		user = "git"
	}
	return []client.Option{
		client.WithHTTPAuth(&githttp.BasicAuth{
			Username: user,
			Password: auth.Password,
		}),
	}
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

	defer r.LockRepo(path)()

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove repository: %w", err)
	}

	if safe, serr := fsutil.SafePath(r.Root, path); serr == nil {
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

	defer r.lockRepos(oldPath, newPath)()

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

	if safe, serr := fsutil.SafePath(r.Root, oldPath); serr == nil {
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
			defaultBranch := receive.DefaultBranch
			req.Hooks.PreReceive = func(ctx context.Context, info *transport.PreReceiveInfo) error {
				return enforceProtectedDefaultBranch(ctx, info, defaultBranch)
			}
		}

		return transport.ReceivePack(ctx, repo.Storer, in, out, req)
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
