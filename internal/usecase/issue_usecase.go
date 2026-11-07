package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"magomedcoder.ru/repo/internal/domain"
)

var (
	ErrIssueTitleRequired  = NewError(KindValidation, "issue_title_required")
	ErrIssueNotFound       = NewError(KindNotFound, "issue_not_found")
	ErrIssueForbidden      = NewError(KindForbidden, "issue_forbidden")
	ErrInvalidIssueState   = NewError(KindValidation, "invalid_issue_state")
	ErrCommentBodyRequired = NewError(KindValidation, "comment_body_required")
	ErrCommentNotFound     = NewError(KindNotFound, "comment_not_found")
	ErrLabelNameRequired   = NewError(KindValidation, "label_name_required")
	ErrLabelNotFound       = NewError(KindNotFound, "label_not_found")
	ErrLabelExists         = NewError(KindConflict, "label_already_exists")
	ErrInvalidLabel        = NewError(KindValidation, "invalid_label")
	ErrIssuesDisabled      = NewError(KindNotFound, "issues_disabled")
)

type LabelItem struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type IssueItem struct {
	Number       int         `json:"number"`
	Title        string      `json:"title"`
	Body         string      `json:"body"`
	State        string      `json:"state"`
	Author       string      `json:"author"`
	Labels       []LabelItem `json:"labels"`
	CommentCount int         `json:"comment_count"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type CommentItem struct {
	ID        uint      `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type IssueDetail struct {
	IssueItem
	Comments []CommentItem `json:"comments"`
}

type IssueUseCase struct {
	issues   domain.IssueStore
	repos    domain.RepositoryStore
	users    domain.UserStore
	activity domain.ActivityStore
	authz    *Authz
}

func NewIssueUseCase(
	issues domain.IssueStore,
	repos domain.RepositoryStore,
	users domain.UserStore,
	activity domain.ActivityStore,
	authz *Authz,
) *IssueUseCase {
	return &IssueUseCase{
		issues:   issues,
		repos:    repos,
		users:    users,
		activity: activity,
		authz:    authz,
	}
}

func (uc *IssueUseCase) List(ctx context.Context, in ResolveRepositoryInput, state string, offset, limit int) ([]IssueItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	offset, limit = NormalizePagination(offset, limit)

	issues, err := uc.issues.ListByRepo(repo.ID, state, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list issues: %w", err)
	}

	return uc.mapIssues(issues)
}

func (uc *IssueUseCase) Get(ctx context.Context, in ResolveRepositoryInput, number int) (*IssueDetail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return nil, err
	}

	item, err := uc.mapIssue(issue)
	if err != nil {
		return nil, err
	}

	comments, err := uc.issues.ListComments(issue.ID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}

	mapped, err := uc.mapComments(comments)
	if err != nil {
		return nil, err
	}

	return &IssueDetail{
		IssueItem: *item,
		Comments:  mapped,
	}, nil
}

func (uc *IssueUseCase) Create(ctx context.Context, in ResolveRepositoryInput, title, body string, labelIDs []uint) (*IssueItem, error) {
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

	if !uc.authz.CanContribute(repo, in.ViewerID) {
		return nil, ErrIssueForbidden
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrIssueTitleRequired
	}

	if err := uc.validateLabels(repo.ID, labelIDs); err != nil {
		return nil, err
	}

	issue := &domain.Issue{
		RepositoryID: repo.ID,
		Title:        title,
		Body:         strings.TrimSpace(body),
		State:        "open",
		AuthorID:     in.ViewerID,
	}

	if err := uc.issues.Create(issue); err != nil {
		return nil, fmt.Errorf("create issue: %w", err)
	}

	if len(labelIDs) > 0 {
		if err := uc.issues.SetIssueLabels(issue.ID, labelIDs); err != nil {
			return nil, fmt.Errorf("set labels: %w", err)
		}
	}

	recordRepoActivity(uc.activity, repo, in.ViewerID, domain.ActivityIssueOpen, activitySummary("opened issue #%d: %s", issue.Number, issue.Title))
	return uc.mapIssue(issue)
}

func (uc *IssueUseCase) Update(ctx context.Context, in ResolveRepositoryInput, number int, title *string, body *string, state *string, labelIDs *[]uint) (*IssueItem, error) {
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

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return nil, err
	}

	if issue.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return nil, ErrIssueForbidden
	}

	prevState := issue.State

	if labelIDs != nil {
		if !uc.authz.CanWriteRepo(repo, in.ViewerID) {
			return nil, ErrIssueForbidden
		}

		if err := uc.validateLabels(repo.ID, *labelIDs); err != nil {
			return nil, err
		}
	}

	if title != nil {
		next := strings.TrimSpace(*title)
		if next == "" {
			return nil, ErrIssueTitleRequired
		}
		issue.Title = next
	}

	if body != nil {
		issue.Body = strings.TrimSpace(*body)
	}

	if state != nil {
		if *state != "open" && *state != "closed" {
			return nil, ErrInvalidIssueState
		}
		issue.State = *state
	}

	if err := uc.issues.Update(issue); err != nil {
		return nil, fmt.Errorf("update issue: %w", err)
	}

	if labelIDs != nil {
		if err := uc.issues.SetIssueLabels(issue.ID, *labelIDs); err != nil {
			return nil, fmt.Errorf("set labels: %w", err)
		}
	}

	if state != nil && *state != prevState && *state == "closed" {
		recordRepoActivity(uc.activity, repo, in.ViewerID, domain.ActivityIssueClose, activitySummary("closed issue #%d", issue.Number))
	}
	return uc.mapIssue(issue)
}

func (uc *IssueUseCase) Delete(ctx context.Context, in ResolveRepositoryInput, number int) error {
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

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return err
	}

	if issue.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return ErrIssueForbidden
	}

	if err := uc.issues.Delete(issue.ID); err != nil {
		return fmt.Errorf("delete issue: %w", err)
	}

	return nil
}

func (uc *IssueUseCase) AddComment(ctx context.Context, in ResolveRepositoryInput, number int, body string) (*CommentItem, error) {
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

	if !uc.authz.CanContribute(repo, in.ViewerID) {
		return nil, ErrIssueForbidden
	}

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return nil, err
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrCommentBodyRequired
	}

	comment := &domain.IssueComment{IssueID: issue.ID, AuthorID: in.ViewerID, Body: body}
	if err := uc.issues.CreateComment(comment); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	return uc.mapComment(comment)
}

func (uc *IssueUseCase) UpdateComment(ctx context.Context, in ResolveRepositoryInput, number, commentID int, body string) (*CommentItem, error) {
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

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return nil, err
	}

	comment, err := uc.findComment(issue.ID, uint(commentID))
	if err != nil {
		return nil, err
	}

	if comment.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return nil, ErrIssueForbidden
	}

	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrCommentBodyRequired
	}

	comment.Body = body
	if err := uc.issues.UpdateComment(comment); err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}

	return uc.mapComment(comment)
}

func (uc *IssueUseCase) DeleteComment(ctx context.Context, in ResolveRepositoryInput, number, commentID int) error {
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

	issue, err := uc.findIssue(repo.ID, number)
	if err != nil {
		return err
	}

	comment, err := uc.findComment(issue.ID, uint(commentID))
	if err != nil {
		return err
	}

	if comment.AuthorID != in.ViewerID && !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return ErrIssueForbidden
	}

	if err := uc.issues.DeleteComment(comment.ID); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}

	return nil
}

func (uc *IssueUseCase) ListLabels(ctx context.Context, in ResolveRepositoryInput) ([]LabelItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	labels, err := uc.issues.ListLabels(repo.ID)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}

	return mapLabels(labels), nil
}

func (uc *IssueUseCase) CreateLabel(ctx context.Context, in ResolveRepositoryInput, name, color string) (*LabelItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.ownerRepo(in)
	if err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return nil, ErrLabelNameRequired
	}

	color = normalizeColor(color)
	if _, err := uc.issues.FindLabelByRepoName(repo.ID, name); err == nil {
		return nil, ErrLabelExists
	} else if !isNotFound(err) {
		return nil, fmt.Errorf("check label: %w", err)
	}

	label := &domain.Label{
		RepositoryID: repo.ID,
		Name:         name,
		Color:        color,
	}
	if err := uc.issues.CreateLabel(label); err != nil {
		return nil, fmt.Errorf("create label: %w", err)
	}

	item := toLabelItem(*label)
	return &item, nil
}

func (uc *IssueUseCase) UpdateLabel(ctx context.Context, in ResolveRepositoryInput, id uint, name, color *string) (*LabelItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repo, err := uc.ownerRepo(in)
	if err != nil {
		return nil, err
	}

	label, err := uc.issues.FindLabel(id)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrLabelNotFound
		}
		return nil, err
	}

	if label.RepositoryID != repo.ID {
		return nil, ErrLabelNotFound
	}

	if name != nil {
		next := strings.TrimSpace(*name)
		if next == "" {
			return nil, ErrLabelNameRequired
		}
		label.Name = next
	}

	if color != nil {
		label.Color = normalizeColor(*color)
	}

	if err := uc.issues.UpdateLabel(label); err != nil {
		return nil, fmt.Errorf("update label: %w", err)
	}

	item := toLabelItem(*label)
	return &item, nil
}

func (uc *IssueUseCase) DeleteLabel(ctx context.Context, in ResolveRepositoryInput, id uint) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repo, err := uc.ownerRepo(in)
	if err != nil {
		return err
	}

	label, err := uc.issues.FindLabel(id)
	if err != nil {
		if isNotFound(err) {
			return ErrLabelNotFound
		}
		return err
	}

	if label.RepositoryID != repo.ID {
		return ErrLabelNotFound
	}

	if err := uc.issues.DeleteLabel(id); err != nil {
		return fmt.Errorf("delete label: %w", err)
	}

	return nil
}

func (uc *IssueUseCase) readable(in ResolveRepositoryInput) (*domain.Repository, error) {
	repo, err := ResolveReadable(uc.repos, uc.authz, in)
	if err != nil {
		return nil, err
	}

	if !repo.IssuesEnabled {
		return nil, ErrIssuesDisabled
	}

	return repo, nil
}

func (uc *IssueUseCase) ownerRepo(in ResolveRepositoryInput) (*domain.Repository, error) {
	if in.ViewerID == 0 {
		return nil, ErrUnauthorized
	}

	repo, err := uc.readable(in)
	if err != nil {
		return nil, err
	}

	if !uc.authz.CanWriteRepo(repo, in.ViewerID) {
		return nil, ErrIssueForbidden
	}

	return repo, nil
}

func (uc *IssueUseCase) findIssue(repoID uint, number int) (*domain.Issue, error) {
	issue, err := uc.issues.FindByRepoNumber(repoID, number)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrIssueNotFound
		}

		return nil, err
	}

	return issue, nil
}

func (uc *IssueUseCase) findComment(issueID, commentID uint) (*domain.IssueComment, error) {
	comment, err := uc.issues.FindComment(commentID)
	if err != nil {
		if isNotFound(err) {
			return nil, ErrCommentNotFound
		}

		return nil, err
	}

	if comment.IssueID != issueID {
		return nil, ErrCommentNotFound
	}

	return comment, nil
}

func (uc *IssueUseCase) validateLabels(repoID uint, ids []uint) error {
	for _, id := range ids {
		label, err := uc.issues.FindLabel(id)
		if err != nil || label.RepositoryID != repoID {
			return ErrInvalidLabel
		}
	}

	return nil
}

func (uc *IssueUseCase) mapIssues(issues []domain.Issue) ([]IssueItem, error) {
	if len(issues) == 0 {
		return []IssueItem{}, nil
	}

	ids := make([]uint, 0, len(issues))
	for _, issue := range issues {
		ids = append(ids, issue.ID)
	}

	byIssue, err := uc.issues.ListLabelsByIssues(ids)
	if err != nil {
		return nil, err
	}

	names := map[uint]string{}
	out := make([]IssueItem, 0, len(issues))
	for _, issue := range issues {
		author, err := uc.username(issue.AuthorID, names)
		if err != nil {
			return nil, err
		}

		count, err := uc.issues.CountComments(issue.ID)
		if err != nil {
			return nil, err
		}

		out = append(out, IssueItem{
			Number:       issue.Number,
			Title:        issue.Title,
			Body:         issue.Body,
			State:        issue.State,
			Author:       author,
			Labels:       mapLabels(byIssue[issue.ID]),
			CommentCount: count,
			CreatedAt:    issue.CreatedAt,
			UpdatedAt:    issue.UpdatedAt,
		})
	}

	return out, nil
}

func (uc *IssueUseCase) mapIssue(issue *domain.Issue) (*IssueItem, error) {
	items, err := uc.mapIssues([]domain.Issue{*issue})
	if err != nil {
		return nil, err
	}

	return &items[0], nil
}

func (uc *IssueUseCase) mapComments(comments []domain.IssueComment) ([]CommentItem, error) {
	names := map[uint]string{}
	out := make([]CommentItem, 0, len(comments))
	for _, c := range comments {
		item, err := uc.mapCommentCached(&c, names)
		if err != nil {
			return nil, err
		}

		out = append(out, *item)
	}

	return out, nil
}

func (uc *IssueUseCase) mapComment(c *domain.IssueComment) (*CommentItem, error) {
	return uc.mapCommentCached(c, map[uint]string{})
}

func (uc *IssueUseCase) mapCommentCached(c *domain.IssueComment, names map[uint]string) (*CommentItem, error) {
	author, err := uc.username(c.AuthorID, names)
	if err != nil {
		return nil, err
	}

	return &CommentItem{
		ID:        c.ID,
		Body:      c.Body,
		Author:    author,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}, nil
}

func (uc *IssueUseCase) username(id uint, cache map[uint]string) (string, error) {
	if name, ok := cache[id]; ok {
		return name, nil
	}

	user, err := uc.users.FindByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			cache[id] = ""
			return "", nil
		}

		return "", fmt.Errorf("find author: %w", err)
	}

	cache[id] = user.Username
	return user.Username, nil
}

func mapLabels(labels []domain.Label) []LabelItem {
	if labels == nil {
		return []LabelItem{}
	}

	out := make([]LabelItem, 0, len(labels))
	for _, l := range labels {
		out = append(out, toLabelItem(l))
	}

	return out
}

func toLabelItem(l domain.Label) LabelItem {
	return LabelItem{
		ID:    l.ID,
		Name:  l.Name,
		Color: l.Color,
	}
}

var labelColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func normalizeColor(raw string) string {
	color := strings.TrimSpace(raw)
	if labelColorPattern.MatchString(color) {
		return color
	}

	return "#1f6b4f"
}

func isNotFound(err error) bool {
	return errors.Is(err, domain.ErrNotFound)
}
