package git

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/fsutil"
)

const lfsOutputCap = 64 << 10

const bundleImportStaging = "refs/bundle-import/"

type cappedBuffer struct {
	buf []byte
	cap int
	n   int
}

func newCappedBuffer(cap int) *cappedBuffer {
	return &cappedBuffer{cap: cap}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.n += len(p)
	if c.cap <= 0 {
		return len(p), nil
	}

	if len(p) >= c.cap {
		c.buf = append(c.buf[:0], p[len(p)-c.cap:]...)

		return len(p), nil
	}

	need := len(c.buf) + len(p) - c.cap
	if need > 0 {
		c.buf = c.buf[need:]
	}

	c.buf = append(c.buf, p...)

	return len(p), nil
}

func (c *cappedBuffer) String() string {
	return string(c.buf)
}

var _ io.Writer = (*cappedBuffer)(nil)

func insecureTLS() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func gitAvailable() bool {
	_, err := exec.LookPath("git")

	return err == nil
}

func lfsRemoteURL(raw string, _ *domain.MirrorOptions) string {
	return raw
}

func lfsEnv(o *domain.MirrorOptions, keyFile string) []string {
	env := filterEnv(os.Environ(), "GIT_TERMINAL_PROMPT", "GIT_SSL_NO_VERIFY", "GIT_SSH_COMMAND")
	env = append(env, "GIT_TERMINAL_PROMPT=0")
	if o != nil && o.InsecureTLS {
		env = append(env, "GIT_SSL_NO_VERIFY=true")
	}

	if keyFile != "" {
		ssh := "ssh -i " + strconv.Quote(keyFile) + " -o IdentitiesOnly=yes -o BatchMode=yes"
		env = append(env, "GIT_SSH_COMMAND="+ssh)
	}

	return env
}

func filterEnv(env []string, dropKeys ...string) []string {
	drop := map[string]struct{}{}
	for _, k := range dropKeys {
		drop[k] = struct{}{}
	}

	out := make([]string, 0, len(env))
	for _, e := range env {
		key, _, _ := strings.Cut(e, "=")
		if _, ok := drop[key]; ok {
			continue
		}

		out = append(out, e)
	}
	return out
}

func lfsRun(ctx context.Context, repoPath, rawURL string, o *domain.MirrorOptions, verb string) error {
	_ = ctx
	_ = repoPath
	_ = rawURL
	_ = o
	_ = verb
	return domain.ErrMirrorLFSUnavailable
}

func partialMirrorClone(ctx context.Context, path, rawURL string, o *domain.MirrorOptions) (string, error) {
	_ = ctx
	_ = path
	_ = rawURL
	_ = o
	return "", domain.ErrMirrorPartialUnavailable
}

func (r *Repository) VerifyCommitSignature(ctx context.Context, repoPath, sha string) error {
	if !gitAvailable() {
		return fmt.Errorf("verify-commit: git binary not found")
	}

	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return fmt.Errorf("verify-commit: empty sha")
	}

	out, err := exec.CommandContext(ctx, "git", "-C", safe, "verify-commit", sha).CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify-commit: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (r *Repository) BundleCreate(ctx context.Context, repoPath, bundlePath string) error {
	if !gitAvailable() {
		return fmt.Errorf("bundle: git binary not found")
	}

	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o700); err != nil {
		return fmt.Errorf("bundle: %w", err)
	}

	out, err := exec.CommandContext(ctx, "git", "-C", safe, "bundle", "create", bundlePath, "--all").CombinedOutput()
	if err != nil {
		_ = os.Remove(bundlePath)

		return fmt.Errorf("bundle create: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (r *Repository) BundleImport(ctx context.Context, repoPath, bundlePath string, protect bool) error {
	if !gitAvailable() {
		return fmt.Errorf("bundle: git binary not found")
	}

	safe, err := fsutil.SafePath(r.Root, repoPath)
	if err != nil {
		return err
	}

	defer r.LockRepo(safe)()

	if out, err := exec.CommandContext(ctx, "git", "-C", safe, "bundle", "verify", bundlePath).CombinedOutput(); err != nil {
		return fmt.Errorf("bundle verify: %w: %s", err, strings.TrimSpace(string(out)))
	}

	if !protect {
		out, err := exec.CommandContext(ctx, "git", "-C", safe, "fetch", "--force", bundlePath, "+refs/*:refs/*").CombinedOutput()
		if err != nil {
			return fmt.Errorf("bundle import: %w: %s", err, strings.TrimSpace(string(out)))
		}

		return nil
	}

	return bundleImportProtected(ctx, safe, bundlePath)
}

func cleanupBundleStaging(safe string) {
	out, err := exec.Command("git", "-C", safe, "for-each-ref", "--format=%(refname)", bundleImportStaging).Output()
	if err != nil {
		return
	}

	for ref := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		ref = strings.TrimSpace(ref)
		if ref == "" || !strings.HasPrefix(ref, bundleImportStaging) {
			continue
		}
		_ = exec.Command("git", "-C", safe, "update-ref", "-d", ref).Run()
	}
}

func bundleImportProtected(ctx context.Context, safe, bundlePath string) error {
	defer cleanupBundleStaging(safe)

	out, err := exec.CommandContext(ctx, "git", "-C", safe, "fetch", "--force", bundlePath, "+refs/*:"+bundleImportStaging+"*").CombinedOutput()
	if err != nil {
		return fmt.Errorf("bundle import: %w: %s", err, strings.TrimSpace(string(out)))
	}

	repo, err := gogit.PlainOpen(safe)
	if err != nil {
		return fmt.Errorf("bundle import: open: %w", err)
	}
	defer repo.Close()

	storer := repo.Storer
	iter, err := repo.References()
	if err != nil {
		return fmt.Errorf("bundle import: refs: %w", err)
	}
	defer iter.Close()

	type update struct {
		name plumbing.ReferenceName
		new  plumbing.Hash
	}
	var planned []update

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		if ref == nil || ref.Type() != plumbing.HashReference {
			return nil
		}

		name := ref.Name().String()
		if !strings.HasPrefix(name, bundleImportStaging) {
			return nil
		}

		finalName := plumbing.ReferenceName("refs/" + strings.TrimPrefix(name, bundleImportStaging))
		newHash := ref.Hash()
		oldHash := plumbing.ZeroHash
		if cur, rerr := storer.Reference(finalName); rerr == nil && cur != nil && cur.Type() == plumbing.HashReference {
			oldHash = cur.Hash()
		}

		ancestor := true
		if !oldHash.IsZero() && !newHash.IsZero() && strings.HasPrefix(string(finalName), "refs/heads/") {
			ok, aerr := isAncestorInStorer(ctx, storer, oldHash, newHash)
			if aerr != nil || !ok {
				ancestor = false
			}
		}

		if err := domain.CheckRefUpdate(true, string(finalName), oldHash.String(), newHash.String(), ancestor); err != nil {
			return err
		}

		planned = append(planned, update{
			name: finalName,
			new:  newHash,
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("bundle import: %w", err)
	}

	for _, u := range planned {
		if err := storer.SetReference(plumbing.NewHashReference(u.name, u.new)); err != nil {
			return fmt.Errorf("bundle import: update %s: %w", u.name, err)
		}
	}

	return nil
}
