package usecase

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	gitadapter "magomedcoder.ru/repo/internal/adapter/git"
	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrPullTitleRequired    = NewError(KindValidation, "pull_title_required")
	ErrPullNotFound         = NewError(KindNotFound, "pull_not_found")
	ErrPullForbidden        = NewError(KindForbidden, "pull_forbidden")
	ErrInvalidPullState     = NewError(KindValidation, "invalid_pull_state")
	ErrBranchNotFound       = NewError(KindNotFound, "branch_not_found")
	ErrSameBranch           = NewError(KindValidation, "same_branch")
	ErrMergeConflict        = NewError(KindConflict, "merge_conflict")
	ErrNotFastForward       = NewError(KindConflict, "not_fast_forward")
	ErrInvalidMergeStrategy = NewError(KindValidation, "invalid_merge_strategy")
	ErrPullNotOpen          = NewError(KindValidation, "pull_not_open")
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
	MergedAt     *time.Time `json:"merged_at,omitzero"`
	MergedBy     string     `json:"merged_by,omitzero"`
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

type pushMirrorScheduler interface {
	SchedulePushMirror(repoID uint)
}

type PullRequestUseCase struct {
	pulls       domain.PullRequestStore
	repos       domain.RepositoryStore
	folders     domain.FolderStore
	users       domain.UserStore
	git         domain.GitRepository
	activity    domain.ActivityStore
	authz       *Authz
	pushMirrors pushMirrorScheduler
}

func (uc *PullRequestUseCase) SetPushMirrorScheduler(s pushMirrorScheduler) {
	uc.pushMirrors = s
}

func NewPullRequestUseCase(
	pulls domain.PullRequestStore,
	repos domain.RepositoryStore,
	folders domain.FolderStore,
	users domain.UserStore,
	git domain.GitRepository,
	activity domain.ActivityStore,
	authz *Authz,
) *PullRequestUseCase {
	return &PullRequestUseCase{
		pulls:    pulls,
		repos:    repos,
		folders:  folders,
		users:    users,
		git:      git,
		activity: activity,
		authz:    authz,
	}
}

func (uc *PullRequestUseCase) List(ctx context.Context, in ResolveRepositoryInput, state string, offset, limit int) ([]PullItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	offset, limit = NormalizePagination(offset, limit)

	items, err := uc.pulls.ListByRepo(repo.ID, state, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list pulls: %w", err)
	}

	return uc.mapPulls(items)
}

func (uc *PullRequestUseCase) Get(ctx context.Context, in ResolveRepositoryInput, number int) (*PullDetail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	item, err := uc.mapPull(pr, nil)
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

	return &PullDetail{
		PullItem: *item,
		Comments: mapped,
	}, nil
}

func (uc *PullRequestUseCase) Create(ctx context.Context, in ResolveRepositoryInput, title, body, base, head string) (*PullItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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

	recordRepoActivity(uc.activity, repo, in.ViewerID, domain.ActivityPROpen, activitySummary("opened pull request #%d: %s", pr.Number, pr.Title))
	return uc.mapPull(pr, nil)
}

func (uc *PullRequestUseCase) Update(ctx context.Context, in ResolveRepositoryInput, number int, title, body, state, base, head *string) (*PullItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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

	if pr.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return nil, ErrPullForbidden
	}

	prevState := pr.State

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
		if !uc.authz.CanWriteRepo(repo, in.ViewerID) {
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

	if state != nil && *state != prevState && *state == "closed" {
		recordRepoActivity(uc.activity, repo, in.ViewerID, domain.ActivityPRClose, activitySummary("closed pull request #%d", pr.Number))
	}
	return uc.mapPull(pr, nil)
}

func (uc *PullRequestUseCase) Delete(ctx context.Context, in ResolveRepositoryInput, number int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	if pr.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return ErrPullForbidden
	}

	return uc.pulls.Delete(pr.ID)
}

func (uc *PullRequestUseCase) Diff(ctx context.Context, in ResolveRepositoryInput, number int) (*PullCompare, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	pr, err := uc.findPull(repo.ID, number)
	if err != nil {
		return nil, err
	}

	cmp, err := uc.git.Compare(ctx, repo.Path, pr.BaseBranch, pr.HeadBranch)
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

func (uc *PullRequestUseCase) Commits(ctx context.Context, in ResolveRepositoryInput, number int) ([]domain.CommitInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cmp, err := uc.Diff(ctx, in, number)
	if err != nil {
		return nil, err
	}

	return cmp.Commits, nil
}

func (uc *PullRequestUseCase) Merge(ctx context.Context, in ResolveRepositoryInput, number int, strategy string) (*PullItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return nil, ErrPullForbidden
	}

	if repo.MirrorDirection == domain.MirrorPull {
		return nil, ErrMirrorReadOnly
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

	user, err := uc.users.FindByID(ctx, in.ViewerID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	msg := fmt.Sprintf("Merge pull request #%d from %s", pr.Number, pr.HeadBranch)
	noreply := fmt.Sprintf("%d+%s@users.noreply.local", user.ID, user.Username)
	_, err = uc.git.Merge(ctx, repo.Path, pr.BaseBranch, pr.HeadBranch, strategy, user.Username, noreply, msg)
	if err != nil {
		return nil, mapGitMergeErr(err)
	}

	now := time.Now()
	pr.State = "merged"
	pr.MergedAt = &now
	viewer := in.ViewerID
	pr.MergedBy = &viewer
	var updateErr error
	for attempt := range 3 {
		updateErr = uc.pulls.Update(pr)
		if updateErr == nil {
			break
		}

		time.Sleep(time.Duration(attempt+1) * 25 * time.Millisecond)
	}

	if updateErr != nil {
		return nil, fmt.Errorf("merge succeeded but failed to update pull state: %w", updateErr)
	}

	_ = uc.repos.TouchLastActivity(repo.ID, now)
	recordRepoActivity(uc.activity, repo, in.ViewerID, domain.ActivityPRMerge, activitySummary("merged pull request #%d", pr.Number))
	if uc.pushMirrors != nil && repo.MirrorDirection == domain.MirrorPush {
		uc.pushMirrors.SchedulePushMirror(repo.ID)
	}

	return uc.mapPull(pr, nil)
}

func (uc *PullRequestUseCase) AddComment(ctx context.Context, in ResolveRepositoryInput, number int, body string) (*CommentItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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

	return uc.mapComment(c, nil)
}

func (uc *PullRequestUseCase) UpdateComment(ctx context.Context, in ResolveRepositoryInput, number int, commentID uint, body string) (*CommentItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

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

	if c.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
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

	return uc.mapComment(c, nil)
}

func (uc *PullRequestUseCase) DeleteComment(ctx context.Context, in ResolveRepositoryInput, number int, commentID uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	if c.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return ErrPullForbidden
	}

	return uc.pulls.DeleteComment(commentID)
}

func (uc *PullRequestUseCase) readable(in ResolveRepositoryInput) (*domain.Repository, error) {
	return ResolveReadable(uc.repos, uc.authz, in)
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

	if slices.ContainsFunc(branches, func(b domain.RefInfo) bool { return b.Name == name }) {
		return nil
	}

	return ErrBranchNotFound
}

func (uc *PullRequestUseCase) mapPulls(items []domain.PullRequest) ([]PullItem, error) {
	idSet := make(map[uint]struct{})
	for i := range items {
		if items[i].AuthorID != 0 {
			idSet[items[i].AuthorID] = struct{}{}
		}

		if items[i].MergedBy != nil && *items[i].MergedBy != 0 {
			idSet[*items[i].MergedBy] = struct{}{}
		}
	}

	ids := slices.Collect(maps.Keys(idSet))

	usersByID, err := uc.users.FindByIDs(context.Background(), ids)
	if err != nil {
		return nil, fmt.Errorf("load pull users: %w", err)
	}

	out := make([]PullItem, 0, len(items))
	for i := range items {
		item, err := uc.mapPull(&items[i], usersByID)
		if err != nil {
			return nil, err
		}

		out = append(out, *item)
	}

	return out, nil
}

func (uc *PullRequestUseCase) mapPull(pr *domain.PullRequest, usersByID map[uint]*domain.User) (*PullItem, error) {
	if usersByID == nil {
		ids := []uint{pr.AuthorID}
		if pr.MergedBy != nil && *pr.MergedBy != 0 && *pr.MergedBy != pr.AuthorID {
			ids = append(ids, *pr.MergedBy)
		}
		var err error
		usersByID, err = uc.users.FindByIDs(context.Background(), ids)
		if err != nil {
			return nil, fmt.Errorf("find pull users: %w", err)
		}
	}

	var authorName string
	if author := usersByID[pr.AuthorID]; author != nil {
		authorName = author.Username
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
		Author:       authorName,
		BaseBranch:   pr.BaseBranch,
		HeadBranch:   pr.HeadBranch,
		CommentCount: count,
		MergedAt:     pr.MergedAt,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
	}
	if pr.MergedBy != nil {
		if u := usersByID[*pr.MergedBy]; u != nil {
			item.MergedBy = u.Username
		}
	}

	return item, nil
}

func (uc *PullRequestUseCase) mapComments(comments []domain.PullRequestComment) ([]CommentItem, error) {
	idSet := make(map[uint]struct{})
	for i := range comments {
		if comments[i].AuthorID != 0 {
			idSet[comments[i].AuthorID] = struct{}{}
		}
	}

	ids := slices.Collect(maps.Keys(idSet))

	usersByID, err := uc.users.FindByIDs(context.Background(), ids)
	if err != nil {
		return nil, fmt.Errorf("load comment authors: %w", err)
	}

	out := make([]CommentItem, 0, len(comments))
	for i := range comments {
		item, err := uc.mapComment(&comments[i], usersByID)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}

	return out, nil
}

func (uc *PullRequestUseCase) mapComment(c *domain.PullRequestComment, usersByID map[uint]*domain.User) (*CommentItem, error) {
	if usersByID == nil {
		var err error
		usersByID, err = uc.users.FindByIDs(context.Background(), []uint{c.AuthorID})
		if err != nil {
			return nil, fmt.Errorf("find author: %w", err)
		}
	}

	var authorName string
	if user := usersByID[c.AuthorID]; user != nil {
		authorName = user.Username
	}

	return &CommentItem{
		ID:        c.ID,
		Body:      c.Body,
		Author:    authorName,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func mapGitCompareErr(err error) error {
	if errors.Is(err, gitadapter.ErrRefNotFound) {
		return ErrBranchNotFound
	}
	if errors.Is(err, gitadapter.ErrHistoryTooLarge) {
		return ErrHistoryTooLarge
	}
	return err
}

func mapGitMergeErr(err error) error {
	switch {
	case errors.Is(err, gitadapter.ErrMergeConflict):
		return ErrMergeConflict
	case errors.Is(err, gitadapter.ErrNotFastForward):
		return ErrNotFastForward
	case errors.Is(err, gitadapter.ErrRefNotFound):
		return ErrBranchNotFound
	default:
		return err
	}
}
