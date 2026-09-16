package git

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v6"
	"github.com/magomedcoder/repo/internal/domain"
)

type Repository struct {
	gitBin string
}

func NewRepository() *Repository {
	bin, err := exec.LookPath("git")
	if err != nil {
		bin = "git"
	}

	return &Repository{gitBin: bin}
}

func (r *Repository) InitBare(path string, opts domain.BareInitOptions) error {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("repository already exists: %s", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	if _, err := gogit.PlainInit(path, true); err != nil {
		return fmt.Errorf("init bare repository: %w", err)
	}

	branch := opts.DefaultBranch
	if branch == "" {
		branch = "main"
	}
	_ = r.setConfig(path, "init.defaultBranch", branch)
	_ = r.setConfig(path, "http.receivepack", "true")
	_ = r.setConfig(path, "core.bare", "true")

	if opts.DenyForcePushDefault {
		if err := r.installForcePushHook(path, branch); err != nil {
			_ = os.RemoveAll(path)
			return err
		}
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

	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("move repository: %w", err)
	}
	return nil
}

func (r *Repository) AdvertiseRefs(repoPath, service string, w io.Writer) error {
	service = normalizeService(service)
	cmdName := strings.TrimPrefix(service, "git-")
	cmd := exec.Command(r.gitBin, cmdName, "--stateless-rpc", "--advertise-refs", repoPath)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("advertise refs: %w", err)
	}

	if _, err := w.Write(pktLine("# service=" + service + "\n")); err != nil {
		return err
	}

	if _, err := w.Write([]byte("0000")); err != nil {
		return err
	}

	_, err = w.Write(out)
	return err
}

func (r *Repository) ServePack(repoPath, service string, stdin io.Reader, stdout io.Writer) error {
	service = normalizeService(service)
	cmdName := strings.TrimPrefix(service, "git-")
	cmd := exec.Command(r.gitBin, cmdName, "--stateless-rpc", repoPath)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w (%s)", cmdName, err, stderr.String())
	}

	return nil
}

func (r *Repository) setConfig(repoPath, key, value string) error {
	cmd := exec.Command(r.gitBin, "config", "--file", filepath.Join(repoPath, "config"), key, value)
	return cmd.Run()
}

func (r *Repository) installForcePushHook(repoPath, defaultBranch string) error {
	hooksDir := filepath.Join(repoPath, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks dir: %w", err)
	}

	script := fmt.Sprintf(`#!/bin/sh
# Deny non-fast-forward pushes to the default branch.
DEFAULT_BRANCH=%q
zero=$(git hash-object --stdin </dev/null | tr '[0-9a-f]' '0')
while read oldrev newrev refname; do
  case "$refname" in
    refs/heads/"$DEFAULT_BRANCH")
      if [ "$oldrev" != "$zero" ] && [ "$newrev" != "$zero" ]; then
        if ! git merge-base --is-ancestor "$oldrev" "$newrev"; then
          echo "force-push to $DEFAULT_BRANCH is not allowed" >&2
          exit 1
        fi
      fi
      ;;
  esac
done
exit 0
`, defaultBranch)

	hookPath := filepath.Join(hooksDir, "pre-receive")
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write pre-receive hook: %w", err)
	}

	return nil
}

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

func pktLine(s string) []byte {
	return []byte(fmt.Sprintf("%04x%s", len(s)+4, s))
}
