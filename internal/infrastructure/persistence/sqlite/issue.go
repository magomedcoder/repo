package sqlite

import (
	"time"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

type issueModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_issue_number"`
	Number       int    `gorm:"not null;uniqueIndex:idx_issue_number"`
	Title        string `gorm:"size:200;not null"`
	Body         string
	State        string `gorm:"size:16;not null;index"`
	AuthorID     uint   `gorm:"not null;index"`
}

func (issueModel) TableName() string {
	return "issues"
}

type issueCommentModel struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	IssueID   uint   `gorm:"not null;index"`
	AuthorID  uint   `gorm:"not null"`
	Body      string `gorm:"not null"`
}

func (issueCommentModel) TableName() string {
	return "issue_comments"
}

type labelModel struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RepositoryID uint   `gorm:"not null;uniqueIndex:idx_label_name"`
	Name         string `gorm:"size:50;not null;uniqueIndex:idx_label_name"`
	Color        string `gorm:"size:16;not null"`
}

func (labelModel) TableName() string {
	return "labels"
}

type issueLabelModel struct {
	IssueID uint `gorm:"primaryKey"`
	LabelID uint `gorm:"primaryKey"`
}

func (issueLabelModel) TableName() string {
	return "issue_labels"
}

type IssueStore struct {
	db *gorm.DB
}

func NewIssueStore(db *gorm.DB) *IssueStore {
	return &IssueStore{db: db}
}

func (s *IssueStore) Create(issue *domain.Issue) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var maxNum int
		if err := tx.Model(&issueModel{}).
			Where("repository_id = ?", issue.RepositoryID).
			Select("COALESCE(MAX(number), 0)").
			Scan(&maxNum).Error; err != nil {
			return err
		}

		issue.Number = maxNum + 1
		if issue.State == "" {
			issue.State = "open"
		}

		model := issueFromDomain(issue)
		if err := tx.Create(model).Error; err != nil {
			return err
		}

		*issue = model.toDomain()
		return nil
	})
}

func (s *IssueStore) Update(issue *domain.Issue) error {
	model := issueFromDomain(issue)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*issue = model.toDomain()
	return nil
}

func (s *IssueStore) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", id).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		if err := tx.Where("issue_id = ?", id).Delete(&issueCommentModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&issueModel{}, id).Error
	})
}

func (s *IssueStore) FindByRepoNumber(repoID uint, number int) (*domain.Issue, error) {
	var model issueModel
	if err := s.db.Where("repository_id = ? AND number = ?", repoID, number).First(&model).Error; err != nil {
		return nil, err
	}

	issue := model.toDomain()
	return &issue, nil
}

func (s *IssueStore) ListByRepo(repoID uint, state string, offset, limit int) ([]domain.Issue, error) {
	q := s.db.Where("repository_id = ?", repoID)
	if state == "open" || state == "closed" {
		q = q.Where("state = ?", state)
	}

	if limit <= 0 {
		limit = 30
	}

	var models []issueModel
	if err := q.Order("number desc").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Issue, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}

	return out, nil
}

func (s *IssueStore) CountComments(issueID uint) (int, error) {
	var n int64
	if err := s.db.Model(&issueCommentModel{}).Where("issue_id = ?", issueID).Count(&n).Error; err != nil {
		return 0, err
	}

	return int(n), nil
}

func (s *IssueStore) CreateComment(comment *domain.IssueComment) error {
	model := commentFromDomain(comment)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()
	return nil
}

func (s *IssueStore) UpdateComment(comment *domain.IssueComment) error {
	model := commentFromDomain(comment)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*comment = model.toDomain()
	return nil
}

func (s *IssueStore) DeleteComment(id uint) error {
	return s.db.Delete(&issueCommentModel{}, id).Error
}

func (s *IssueStore) FindComment(id uint) (*domain.IssueComment, error) {
	var model issueCommentModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	c := model.toDomain()
	return &c, nil
}

func (s *IssueStore) ListComments(issueID uint) ([]domain.IssueComment, error) {
	var models []issueCommentModel
	if err := s.db.Where("issue_id = ?", issueID).Order("created_at asc").Find(&models).Error; err != nil {
		return nil, err
	}

	out := make([]domain.IssueComment, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}
	
	return out, nil
}

func (s *IssueStore) CreateLabel(label *domain.Label) error {
	model := labelFromDomain(label)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	*label = model.toDomain()
	return nil
}

func (s *IssueStore) UpdateLabel(label *domain.Label) error {
	model := labelFromDomain(label)
	if err := s.db.Save(model).Error; err != nil {
		return err
	}

	*label = model.toDomain()
	return nil
}

func (s *IssueStore) DeleteLabel(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("label_id = ?", id).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		return tx.Delete(&labelModel{}, id).Error
	})
}

func (s *IssueStore) FindLabel(id uint) (*domain.Label, error) {
	var model labelModel
	if err := s.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	l := model.toDomain()
	return &l, nil
}

func (s *IssueStore) ListLabels(repoID uint) ([]domain.Label, error) {
	var models []labelModel
	if err := s.db.Where("repository_id = ?", repoID).Order("name asc").Find(&models).Error; err != nil {
		return nil, err
	}

	return labelsToDomain(models), nil
}

func (s *IssueStore) FindLabelByRepoName(repoID uint, name string) (*domain.Label, error) {
	var model labelModel
	if err := s.db.Where("repository_id = ? AND name = ?", repoID, name).First(&model).Error; err != nil {
		return nil, err
	}

	l := model.toDomain()
	return &l, nil
}

func (s *IssueStore) SetIssueLabels(issueID uint, labelIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("issue_id = ?", issueID).Delete(&issueLabelModel{}).Error; err != nil {
			return err
		}

		for _, id := range labelIDs {
			row := issueLabelModel{IssueID: issueID, LabelID: id}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *IssueStore) ListIssueLabels(issueID uint) ([]domain.Label, error) {
	var models []labelModel
	err := s.db.Model(&labelModel{}).
		Joins("JOIN issue_labels ON issue_labels.label_id = labels.id").
		Where("issue_labels.issue_id = ?", issueID).
		Order("labels.name asc").
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	return labelsToDomain(models), nil
}

func (s *IssueStore) ListLabelsByIssues(issueIDs []uint) (map[uint][]domain.Label, error) {
	out := map[uint][]domain.Label{}
	if len(issueIDs) == 0 {
		return out, nil
	}

	var links []issueLabelModel
	if err := s.db.Where("issue_id IN ?", issueIDs).Find(&links).Error; err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return out, nil
	}

	ids := make([]uint, 0, len(links))
	seen := map[uint]struct{}{}
	for _, link := range links {
		if _, ok := seen[link.LabelID]; ok {
			continue
		}

		seen[link.LabelID] = struct{}{}
		ids = append(ids, link.LabelID)
	}

	var models []labelModel
	if err := s.db.Where("id IN ?", ids).Find(&models).Error; err != nil {
		return nil, err
	}

	byID := map[uint]domain.Label{}
	for _, m := range models {
		byID[m.ID] = m.toDomain()
	}

	for _, link := range links {
		if label, ok := byID[link.LabelID]; ok {
			out[link.IssueID] = append(out[link.IssueID], label)
		}
	}

	return out, nil
}

func issueFromDomain(issue *domain.Issue) *issueModel {
	return &issueModel{
		ID:           issue.ID,
		RepositoryID: issue.RepositoryID,
		Number:       issue.Number,
		Title:        issue.Title,
		Body:         issue.Body,
		State:        issue.State,
		AuthorID:     issue.AuthorID,
		CreatedAt:    issue.CreatedAt,
		UpdatedAt:    issue.UpdatedAt,
	}
}

func (m issueModel) toDomain() domain.Issue {
	return domain.Issue{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Number:       m.Number,
		Title:        m.Title,
		Body:         m.Body,
		State:        m.State,
		AuthorID:     m.AuthorID,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func commentFromDomain(c *domain.IssueComment) *issueCommentModel {
	return &issueCommentModel{
		ID:        c.ID,
		IssueID:   c.IssueID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (m issueCommentModel) toDomain() domain.IssueComment {
	return domain.IssueComment{
		ID:        m.ID,
		IssueID:   m.IssueID,
		AuthorID:  m.AuthorID,
		Body:      m.Body,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func labelFromDomain(l *domain.Label) *labelModel {
	return &labelModel{
		ID:           l.ID,
		RepositoryID: l.RepositoryID,
		Name:         l.Name,
		Color:        l.Color,
		CreatedAt:    l.CreatedAt,
		UpdatedAt:    l.UpdatedAt,
	}
}

func (m labelModel) toDomain() domain.Label {
	return domain.Label{
		ID:           m.ID,
		RepositoryID: m.RepositoryID,
		Name:         m.Name,
		Color:        m.Color,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func labelsToDomain(models []labelModel) []domain.Label {
	out := make([]domain.Label, 0, len(models))
	for _, m := range models {
		out = append(out, m.toDomain())
	}
	
	return out
}
