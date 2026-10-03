package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/magomedcoder/repo/internal/domain"
)

func (r *Repository) Compare(repoPath, base, head string) (*domain.CompareResult, error) {
	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	baseCommit, err := r.commitFromRef(repo, base)
	if err != nil {
		return nil, fmt.Errorf("base ref not found: %w", err)
	}

	headCommit, err := r.commitFromRef(repo, head)
	if err != nil {
		return nil, fmt.Errorf("head ref not found: %w", err)
	}

	out := &domain.CompareResult{
		BaseSHA: baseCommit.Hash.String(),
		HeadSHA: headCommit.Hash.String(),
		Commits: []domain.CommitInfo{},
		Files:   []domain.FileDiff{},
	}

	ff, err := r.CanFastForward(repoPath, base, head)
	if err != nil {
		return nil, err
	}

	out.CanFastForward = ff

	commits, err := r.logRange(repoPath, base, head)
	if err != nil {
		return nil, err
	}

	out.Commits = commits

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, err
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, err
	}

	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, err
	}

	for _, change := range changes {
		if fd, ok := fileDiffFromChange(change); ok {
			out.Files = append(out.Files, fd)
		}
	}

	return out, nil
}

func (r *Repository) logRange(repoPath, base, head string) ([]domain.CommitInfo, error) {
	cmd := exec.Command(r.gitBin, "--git-dir", repoPath, "log", "--format=%H", base+".."+head)
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("log range: %w", err)
	}

	repo, err := r.open(repoPath)
	if err != nil {
		return nil, err
	}

	text := strings.TrimSpace(string(raw))
	if text == "" {
		return []domain.CommitInfo{}, nil
	}

	lines := strings.Split(text, "\n")
	out := make([]domain.CommitInfo, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		c, err := repo.CommitObject(plumbing.NewHash(line))
		if err != nil {
			continue
		}
		out = append(out, commitToInfo(c))
	}

	return out, nil
}

func (r *Repository) CanFastForward(repoPath, base, head string) (bool, error) {
	cmd := exec.Command(r.gitBin, "--git-dir", repoPath, "merge-base", "--is-ancestor", base, head)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("merge-base: %w", err)
}

func (r *Repository) Merge(repoPath, base, head, strategy, authorName, authorEmail, message string) (string, error) {
	base = strings.TrimSpace(base)
	head = strings.TrimSpace(head)
	if base == "" || head == "" {
		return "", fmt.Errorf("base and head required")
	}

	if base == head {
		return "", fmt.Errorf("base and head are the same")
	}

	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		strategy = "merge"
	}

	if strategy != "merge" && strategy != "ff-only" {
		return "", fmt.Errorf("unsupported merge strategy")
	}

	if authorName == "" {
		authorName = "Repo"
	}

	if authorEmail == "" {
		authorEmail = "repo@localhost"
	}

	if message == "" {
		message = fmt.Sprintf("Merge branch '%s' into %s", head, base)
	}

	ff, err := r.CanFastForward(repoPath, base, head)
	if err != nil {
		return "", err
	}

	if strategy == "ff-only" {
		if !ff {
			return "", fmt.Errorf("not fast-forward")
		}

		return r.fastForward(repoPath, base, head)
	}

	if ff {
		return r.fastForward(repoPath, base, head)
	}

	tmp, err := os.MkdirTemp("", "repo-merge-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	wt := filepath.Join(tmp, "wt")
	add := exec.Command(r.gitBin, "--git-dir", repoPath, "worktree", "add", "-f", "--detach", wt, base)
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("worktree add: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	defer func() {
		_ = exec.Command(r.gitBin, "--git-dir", repoPath, "worktree", "remove", "--force", wt).Run()
		_ = exec.Command(r.gitBin, "--git-dir", repoPath, "worktree", "prune").Run()
	}()

	merge := exec.Command(r.gitBin, "merge", "--no-ff", "-m", message, head)
	merge.Dir = wt
	merge.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME="+authorName,
		"GIT_AUTHOR_EMAIL="+authorEmail,
		"GIT_COMMITTER_NAME="+authorName,
		"GIT_COMMITTER_EMAIL="+authorEmail,
		"GIT_AUTHOR_DATE="+time.Now().Format(time.RFC3339),
		"GIT_COMMITTER_DATE="+time.Now().Format(time.RFC3339),
	)
	if out, err := merge.CombinedOutput(); err != nil {
		abort := exec.Command(r.gitBin, "merge", "--abort")
		abort.Dir = wt
		_ = abort.Run()
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "conflict") {
			return "", fmt.Errorf("merge conflict")
		}

		return "", fmt.Errorf("merge: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	shaCmd := exec.Command(r.gitBin, "rev-parse", "HEAD")
	shaCmd.Dir = wt
	shaRaw, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse: %w", err)
	}

	sha := strings.TrimSpace(string(shaRaw))
	upd := exec.Command(r.gitBin, "--git-dir", repoPath, "update-ref", "refs/heads/"+base, sha)
	if out, err := upd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("update-ref: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	return sha, nil
}

func (r *Repository) fastForward(repoPath, base, head string) (string, error) {
	shaCmd := exec.Command(r.gitBin, "--git-dir", repoPath, "rev-parse", head)
	shaRaw, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("rev-parse head: %w", err)
	}

	sha := strings.TrimSpace(string(shaRaw))
	upd := exec.Command(r.gitBin, "--git-dir", repoPath, "update-ref", "refs/heads/"+base, sha)
	if out, err := upd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("fast-forward: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	return sha, nil
}

func fileDiffFromChange(change *object.Change) (domain.FileDiff, bool) {
	action, err := change.Action()
	if err != nil {
		return domain.FileDiff{}, false
	}

	fd := domain.FileDiff{}
	switch action.String() {
	case "Insert":
		fd.Status = "added"
		fd.Path = change.To.Name
	case "Delete":
		fd.Status = "deleted"
		fd.Path = change.From.Name
	case "Modify":
		fd.Status = "modified"
		fd.Path = change.To.Name
		if change.From.Name != "" && change.From.Name != change.To.Name {
			fd.Status = "renamed"
			fd.OldPath = change.From.Name
		}
	default:
		fd.Status = strings.ToLower(action.String())
		if change.To.Name != "" {
			fd.Path = change.To.Name
		} else {
			fd.Path = change.From.Name
		}
	}

	if patch, err := change.Patch(); err == nil {
		fd.Patch = patch.String()
	}

	return fd, true
}
