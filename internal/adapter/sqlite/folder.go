package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type Folder struct {
	gormutil.Repo[model.Folder]
}

func NewFolderRepo(db *gorm.DB) *Folder {
	return &Folder{Repo: gormutil.NewRepo[model.Folder](db)}
}

func (s *Folder) Create(folder *domain.Folder) error {
	m := model.FolderFromDomain(folder)
	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	*folder = m.ToDomain()
	return nil
}

func (s *Folder) Update(folder *domain.Folder) error {
	m := model.FolderFromDomain(folder)
	if err := s.Save(context.Background(), m); err != nil {
		return err
	}

	*folder = m.ToDomain()
	return nil
}

func (s *Folder) Delete(id uint) error {
	return s.DeleteByID(context.Background(), id)
}

func (s *Folder) FindByID(id uint) (*domain.Folder, error) {
	row, err := s.Repo.FindByID(context.Background(), id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := row.ToDomain()
	return &f, nil
}

func (s *Folder) FindByPath(path string) (*domain.Folder, error) {
	row, err := s.FindByWhere(context.Background(), "path = ?", path)
	if err != nil {
		return nil, mapDBErr(err)
	}

	f := row.ToDomain()
	return &f, nil
}

func (s *Folder) ListForMember(userID uint) ([]domain.Folder, error) {
	models, err := s.FindAll(context.Background(),
		gormutil.WithJoins("JOIN folder_members ON folder_members.folder_id = folders.id AND folder_members.deleted_at IS NULL"),
		gormutil.WithWhere("folder_members.user_id = ?", userID),
		gormutil.WithOrder("folders.name ASC"),
	)
	if err != nil {
		return nil, err
	}

	return foldersToDomain(models), nil
}

func (s *Folder) ExistsBySlug(slug string) (bool, error) {
	return s.QueryExist(context.Background(), "slug = ?", slug)
}

func foldersToDomain(models []model.Folder) []domain.Folder {
	out := make([]domain.Folder, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}
	return out
}
