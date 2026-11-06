package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type Team struct {
	gormutil.Repo[model.Team]
}

func NewTeamRepo(db *gorm.DB) *Team {
	return &Team{Repo: gormutil.NewRepo[model.Team](db)}
}

func (s *Team) Create(t *domain.Team) error {
	m := model.Team{
		FolderID: t.FolderID,
		Name:     t.Name,
		Slug:     t.Slug,
		Caps:     uint32(t.Caps),
	}
	if err := s.Repo.Create(context.Background(), &m); err != nil {
		return err
	}
	*t = m.ToDomain()

	return nil
}

func (s *Team) Update(t *domain.Team) error {
	m := model.Team{
		ID:       t.ID,
		FolderID: t.FolderID,
		Name:     t.Name,
		Slug:     t.Slug,
		Caps:     uint32(t.Caps),
	}
	if err := s.Save(context.Background(), &m); err != nil {
		return err
	}
	*t = m.ToDomain()

	return nil
}

func (s *Team) Delete(id uint) error {
	return s.DeleteByID(context.Background(), id)
}

func (s *Team) FindByID(id uint) (*domain.Team, error) {
	row, err := s.Repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *Team) FindByFolderSlug(folderID uint, slug string) (*domain.Team, error) {
	row, err := s.FindByWhere(context.Background(), "folder_id = ? AND slug = ?", folderID, slug)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *Team) ListByFolder(folderID uint) ([]domain.Team, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "folder_id = ?", "name ASC", folderID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.Team, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}

func (s *Team) ExistsByFolderSlug(folderID uint, slug string) (bool, error) {
	return s.QueryExist(context.Background(), "folder_id = ? AND slug = ?", folderID, slug)
}
