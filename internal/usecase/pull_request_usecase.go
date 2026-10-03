package usecase

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/magomedcoder/repo/internal/domain"
)

var (
	ErrPullTitleRequired    = errors.New("pull_title_required")
	ErrPullNotFound         = errors.New("pull_not_found")
	ErrPullForbidden        = errors.New("pull_forbidden")
	ErrInvalidPullState     = errors.New("invalid_pull_state")
	ErrBranchNotFound       = errors.New("branch_not_found")
	ErrSameBranch           = errors.New("same_branch")
	ErrMergeConflict        = errors.New("merge_conflict")
	ErrNotFastForward       = errors.New("not_fast_forward")
	ErrInvalidMergeStrategy = errors.New("invalid_merge_strategy")
	ErrPullNotOpen          = errors.New("pull_not_open")
)

type PullItem struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	Author       string     `json:"author"`
	BaseBranch   string     `json:"base_branch"`
	HeadBranch   string     `json:"head_branch"`
	CommentCount int        `json:"comment_count"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	MergedBy     string     `json:"merged_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type PullDetail struct {
	PullItem
	Comments       []CommentItem `json:"comments"`
	CanFastForward bool          `json:"can_fast_forward"`
	Commits        int           `json:"commits"`
	FilesChanged   int           `json:"files_changed"`
}

type PullCompare struct {
	BaseSHA        string              `json:"base_sha"`
	HeadSHA        string              `json:"head_sha"`
	CanFastForward bool                `json:"can_fast_forward"`
	Commits        []domain.CommitInfo `json:"commits"`
	Files          []domain.FileDiff   `json:"files"`
}

type PullRequestUseCase struct {
	pulls   domain.PullRequestStore
	repos   domain.RepositoryStore
	folders domain.FolderStore
	users   domain.UserStore
	git     domain.GitRepository
}

func NewPullRequestUseCase(
	pulls domain.PullRequestStore,
	repos domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	git domain.GitRepository,
) *PullRequestUseCase {
	return &PullRequestUseCase{
		pulls:   pulls,
		repos:   repos,
		folders: folders,
		users:   users,
		git:     git,
	}
}

func (uc *PullRequestUseCase) List(in ResolveRepositoryInput, state string, offset, limit int) ([]PullItem, error) {
	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 30
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		offset = 0
	}

	items, err := uc.pulls.ListByRepo(repo.ID, state, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list pulls: %w", err)
	}

	return uc.mapPulls(items)
}

func (uc *PullRequestUseCase) Get(in ResolveRepositoryInput, number int) (*PullDetail, error) {
	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	item, err := uc.mapPull(pr)
	if err != nil {
		return nil, err
	}

	comments, err := uc.pulls.ListComments(pr.ID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	mapped, err := uc.mapComments(comments)
	if err != nil {
		return nil, err
	}

	detail := &PullDetail{
		PullItem: *item,
		Comments: mapped,
	}
	if pr.State == "open" {
		cmp, cerr := uc.git.Compare(repo.Path, pr.BaseBranch, pr.HeadBranch)
		if cerr == nil {
			detail.CanFastForward = cmp.CanFastForward
			detail.Commits = len(cmp.Commits)
			detail.FilesChanged = len(cmp.Files)
		}
	}

	return detail, nil
}

func (uc *PullRequestUseCase) Create(in ResolveRepositoryInput, title, body, base, head string) (*PullItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrPullTitleRequired
	}

	base = strings.TrimSpace(base)
	head = strings.TrimSpace(head)
	if base == "" {
		base = repo.DefaultBranch
	}

	if head == "" {
		return nil, ErrBranchNotFound
	}

	if base == head {
		return nil, ErrSameBranch
	}

	if err := uc.ensureBranch(repo.Path, base); err != nil {
		return nil, err
	}

	if err := uc.ensureBranch(repo.Path, head); err != nil {
		return nil, err
	}

	pr := &domain.PullRequest{
		RepositoryID: repo.ID,
		Title:        title,
		Body:         strings.TrimSpace(body),
		State:        "open",
		AuthorID:     in.ViewerID,
		BaseBranch:   base,
		HeadBranch:   head,
	}
	if err := uc.pulls.Create(pr); err != nil {
		return nil, fmt.Errorf("create pull: %w", err)
	}

	return uc.mapPull(pr)
}

func (uc *PullRequestUseCase) Update(in ResolveRepositoryInput, number int, title, body, state, base, head *string) (*PullItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	if pr.AuthorID != in.ViewerID && repo.OwnerID != in.ViewerID {
		return nil, ErrPullForbidden
	}

	if title != nil {
		next := strings.TrimSpace(*title)
		if next == "" {
			return nil, ErrPullTitleRequired
		}
		pr.Title = next
	}

	if body != nil {
		pr.Body = strings.TrimSpace(*body)
	}

	if state != nil {
		switch *state {
		case "open":
			if pr.State == "merged" {
				return nil, ErrInvalidPullState
			}

			pr.State = "open"
		case "closed":
			if pr.State == "merged" {
				return nil, ErrInvalidPullState
			}

			pr.State = "closed"
		default:
			return nil, ErrInvalidPullState
		}
	}

	if base != nil || head != nil {
		if repo.OwnerID != in.ViewerID {
			return nil, ErrPullForbidden
		}

		if pr.State != "open" {
			return nil, ErrPullNotOpen
		}

		nextBase := pr.BaseBranch
		nextHead := pr.HeadBranch
		if base != nil {
			nextBase = strings.TrimSpace(*base)
		}

		if head != nil {
			nextHead = strings.TrimSpace(*head)
		}

		if nextBase == "" || nextHead == "" {
			return nil, ErrBranchNotFound
		}

		if nextBase == nextHead {
			return nil, ErrSameBranch
		}

		if err := uc.ensureBranch(repo.Path, nextBase); err != nil {
			return nil, err
		}

		if err := uc.ensureBranch(repo.Path, nextHead); err != nil {
			return nil, err
		}

		pr.BaseBranch = nextBase
		pr.HeadBranch = nextHead
	}
	if err := uc.pulls.Update(pr); err != nil {
		return nil, fmt.Errorf("update pull: %w", err)
	}

	return uc.mapPull(pr)
}

func (uc *PullRequestUseCase) Delete(in ResolveRepositoryInput, number int) error {
	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return err
	}

	if pr.AuthorID != in.ViewerID && repo.OwnerID != in.ViewerID {
		return ErrPullForbidden
	}

	return uc.pulls.Delete(pr.ID)
}

func (uc *PullRequestUseCase) Diff(in ResolveRepositoryInput, number int) (*PullCompare, error) {
	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	cmp, err := uc.git.Compare(repo.Path, pr.BaseBranch, pr.HeadBranch)
	if err != nil {
		return nil, mapGitCompareErr(err)
	}

	return &PullCompare{
		BaseSHA:        cmp.BaseSHA,
		HeadSHA:        cmp.HeadSHA,
		CanFastForward: cmp.CanFastForward,
		Commits:        cmp.Commits,
		Files:          cmp.Files,
	}, nil
}

func (uc *PullRequestUseCase) Commits(in ResolveRepositoryInput, number int) ([]domain.CommitInfo, error) {
	cmp, err := uc.Diff(in, number)
	if err != nil {
		return nil, err
	}

	return cmp.Commits, nil
}

func (uc *PullRequestUseCase) Merge(in ResolveRepositoryInput, number int, strategy string) (*PullItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	if repo.OwnerID != in.ViewerID {
		return nil, ErrPullForbidden
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	if pr.State != "open" {
		return nil, ErrPullNotOpen
	}

	strategy = strings.TrimSpace(strategy)
	if strategy == "" {
		strategy = "merge"
	}

	if strategy != "merge" && strategy != "ff-only" {
		return nil, ErrInvalidMergeStrategy
	}

	user, err := uc.users.FindByID(in.ViewerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	msg := fmt.Sprintf("Merge pull request #%d from %s", pr.Number, pr.HeadBranch)
	_, err = uc.git.Merge(repo.Path, pr.BaseBranch, pr.HeadBranch, strategy, user.Username, user.Email, msg)
	if err != nil {
		return nil, mapGitMergeErr(err)
	}

	now := time.Now()
	pr.State = "merged"
	pr.MergedAt = &now
	viewer := in.ViewerID
	pr.MergedBy = &viewer
	if err := uc.pulls.Update(pr); err != nil {
		return nil, fmt.Errorf("update pull: %w", err)
	}

	_ = uc.repos.TouchLastActivity(repo.ID, now)
	return uc.mapPull(pr)
}

func (uc *PullRequestUseCase) AddComment(in ResolveRepositoryInput, number int, body string) (*CommentItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrCommentBodyRequired
	}

	c := &domain.PullRequestComment{PullID: pr.ID, AuthorID: in.ViewerID, Body: body}
	if err := uc.pulls.CreateComment(c); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	return uc.mapComment(c)
}

func (uc *PullRequestUseCase) UpdateComment(in ResolveRepositoryInput, number int, commentID uint, body string) (*CommentItem, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	c, err := uc.pulls.FindComment(commentID)
	if err != nil || c.PullID != pr.ID {
		return nil, ErrCommentNotFound
	}

	if c.AuthorID != in.ViewerID && repo.OwnerID != in.ViewerID {
		return nil, ErrPullForbidden
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrCommentBodyRequired
	}

	c.Body = body
	if err := uc.pulls.UpdateComment(c); err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}

	return uc.mapComment(c)
}

func (uc *PullRequestUseCase) DeleteComment(in ResolveRepositoryInput, number int, commentID uint) error {
	if in.ViewerID == 0 {
		return ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return err
	}

	c, err := uc.pulls.FindComment(commentID)
	if err != nil || c.PullID != pr.ID {
		return ErrCommentNotFound
	}

	if c.AuthorID != in.ViewerID && repo.OwnerID != in.ViewerID {
		return ErrPullForbidden
	}

	return uc.pulls.DeleteComment(commentID)
}

func (uc *PullRequestUseCase) readable(in ResolveRepositoryInput) (*domain.Repository, error) {
	repoUC := &RepositoryUseCase{store: uc.repos, folders: uc.folders, users: uc.users}
	repo, _, _, err := repoUC.resolve(in)
	if err != nil {
		return nil, err
	}

	if err := repoUC.authorizeRead(repo, in.ViewerID); err != nil {
		return nil, err
	}

	return repo, nil
}

func (uc *PullRequestUseCase) findPull(repoID uint, number int) (*domain.PullRequest, error) {
	pr, err := uc.pulls.FindByRepoNumber(repoID, number)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrPullNotFound
		}

		return nil, err
	}

	return pr, nil
}

func (uc *PullRequestUseCase) ensureBranch(repoPath, name string) error {
	branches, err := uc.git.ListBranches(repoPath)
	if err != nil {
		return fmt.Errorf("list branches: %w", err)
	}

	for _, b := range branches {
		if b.Name == name {
			return nil
		}
	}

	return ErrBranchNotFound
}

func (uc *PullRequestUseCase) mapPulls(items []domain.PullRequest) ([]PullItem, error) {
	out := make([]PullItem, 0, len(items))
	for i := range items {
		item, err := uc.mapPull(&items[i])
		if err != nil {
			return nil, err
		}

		out = append(out, *item)
	}

	return out, nil
}

func (uc *PullRequestUseCase) mapPull(pr *domain.PullRequest) (*PullItem, error) {
	author, err := uc.users.FindByID(pr.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("find author: %w", err)
	}

	count, err := uc.pulls.CountComments(pr.ID)
	if err != nil {
		return nil, err
	}

	item := &PullItem{
		Number:       pr.Number,
		Title:        pr.Title,
		Body:         pr.Body,
		State:        pr.State,
		Author:       author.Username,
		BaseBranch:   pr.BaseBranch,
		HeadBranch:   pr.HeadBranch,
		CommentCount: count,
		MergedAt:     pr.MergedAt,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
	}
	if pr.MergedBy != nil {
		if u, err := uc.users.FindByID(*pr.MergedBy); err == nil {
			item.MergedBy = u.Username
		}
	}

	return item, nil
}

func (uc *PullRequestUseCase) mapComments(comments []domain.PullRequestComment) ([]CommentItem, error) {
	out := make([]CommentItem, 0, len(comments))
	for i := range comments {
		item, err := uc.mapComment(&comments[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}

	return out, nil
}

func (uc *PullRequestUseCase) mapComment(c *domain.PullRequestComment) (*CommentItem, error) {
	user, err := uc.users.FindByID(c.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("find author: %w", err)
	}

	return &CommentItem{
		ID:        c.ID,
		Body:      c.Body,
		Author:    user.Username,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func mapGitCompareErr(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "ref not found") || strings.Contains(msg, "not found") {
		return ErrBranchNotFound
	}

	return err
}

func mapGitMergeErr(err error) error {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "merge conflict"):
		return ErrMergeConflict
	case strings.Contains(msg, "not fast-forward"):
		return ErrNotFastForward
	case strings.Contains(msg, "ref not found"), strings.Contains(msg, "unknown revision"):
		return ErrBranchNotFound
	default:
		return err
	}
}
