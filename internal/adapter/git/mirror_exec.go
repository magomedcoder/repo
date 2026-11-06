package git

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"magomedcoder.ru/repo/internal/domain"
)

const lfsOutputCap = 64 << 10

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

func lfsAvailable(ctx context.Context) bool {
	if !gitAvailable() {
		return false
	}

	return exec.CommandContext(ctx, "git", "lfs", "version").Run() == nil
}

func lfsRemoteURL(raw string, o *domain.MirrorOptions) string {
	if isSSHURL(raw) || o.Auth == nil || o.Auth.Password == "" {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	user := o.Auth.Username
	if user == "" {
		user = "git"
	}

	u.User = url.UserPassword(user, o.Auth.Password)

	return u.String()
}

func lfsEnv(o *domain.MirrorOptions, keyFile string) []string {
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if o.InsecureTLS {
		env = append(env, "GIT_SSL_NO_VERIFY=true")
	}

	if keyFile != "" {
		ssh := "ssh -i " + keyFile + " -o IdentitiesOnly=yes -o BatchMode=yes"
		if o.InsecureTLS {
			ssh += " -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
		}

		env = append(env, "GIT_SSH_COMMAND="+ssh)
	}

	return env
}

func lfsRun(ctx context.Context, repoPath, rawURL string, o *domain.MirrorOptions, verb string) error {
	if !lfsAvailable(ctx) {
		return domain.ErrMirrorLFSMissing
	}

	keyFile := ""
	if isSSHURL(rawURL) && o.SSHPrivateKey != "" {
		if o.SSHPassphrase != "" {
			return fmt.Errorf("mirror lfs: passphrase-protected ssh keys are not supported for LFS")
		}

		f, err := os.CreateTemp("", "mirror-lfs-key-*")
		if err != nil {
			return fmt.Errorf("mirror lfs: %w", err)
		}

		defer os.Remove(f.Name())

		key := o.SSHPrivateKey
		if !strings.HasSuffix(key, "\n") {
			key += "\n"
		}

		if _, err := f.WriteString(key); err != nil {
			_ = f.Close()

			return fmt.Errorf("mirror lfs: %w", err)
		}

		_ = f.Close()
		keyFile = f.Name()
	}

	remote := lfsRemoteURL(rawURL, o)
	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "lfs", verb, "--all", remote)
	cmd.Env = lfsEnv(o, keyFile)

	out := newCappedBuffer(lfsOutputCap)
	cmd.Stdout = out
	cmd.Stderr = out

	if err := cmd.Run(); err != nil {
		msg := out.String()
		msg = strings.ReplaceAll(msg, remote, domain.RedactURL(rawURL))

		if o.Auth != nil && o.Auth.Password != "" {
			msg = strings.ReplaceAll(msg, o.Auth.Password, "***")
		}

		return fmt.Errorf("mirror lfs %s: %w: %s", verb, err, strings.TrimSpace(msg))
	}

	return nil
}

func partialMirrorClone(ctx context.Context, path, rawURL string, o *domain.MirrorOptions) (string, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}

	remote := lfsRemoteURL(rawURL, o)
	args := []string{"clone", "--mirror", "--filter=blob:none", remote, path}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = lfsEnv(o, "")

	out := newCappedBuffer(lfsOutputCap)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		msg := strings.ReplaceAll(out.String(), remote, domain.RedactURL(rawURL))
		return "", fmt.Errorf("partial mirror clone: %w: %s", err, strings.TrimSpace(msg))
	}

	branch, _ := detectHEADBranch(path)

	return branch, nil
}

func (r *Repository) VerifyCommitSignature(ctx context.Context, repoPath, sha string) error {
	if !gitAvailable() {
		return fmt.Errorf("verify-commit: git binary not found")
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		return fmt.Errorf("verify-commit: empty sha")
	}

	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "verify-commit", sha).CombinedOutput()
	if err != nil {
		return fmt.Errorf("verify-commit: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (r *Repository) BundleCreate(ctx context.Context, repoPath, bundlePath string) error {
	if !gitAvailable() {
		return fmt.Errorf("bundle: git binary not found")
	}

	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o700); err != nil {
		return fmt.Errorf("bundle: %w", err)
	}

	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "bundle", "create", bundlePath, "--all").CombinedOutput()
	if err != nil {
		_ = os.Remove(bundlePath)

		return fmt.Errorf("bundle create: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

func (r *Repository) BundleImport(ctx context.Context, repoPath, bundlePath string) error {
	if !gitAvailable() {
		return fmt.Errorf("bundle: git binary not found")
	}

	if out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "bundle", "verify", bundlePath).CombinedOutput(); err != nil {
		return fmt.Errorf("bundle verify: %w: %s", err, strings.TrimSpace(string(out)))
	}

	out, err := exec.CommandContext(ctx, "git", "-C", repoPath, "fetch", "--force", bundlePath, "+refs/*:refs/*").CombinedOutput()
	if err != nil {
		return fmt.Errorf("bundle import: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return nil
}
