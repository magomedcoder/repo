package sqlite

import (
	"context"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type FolderMember struct {
	gormutil.Repo[model.FolderMember]
}

func NewFolderMemberRepo(db *gorm.DB) *FolderMember {
	return &FolderMember{Repo: gormutil.NewRepo[model.FolderMember](db)}
}

func (s *FolderMember) Create(mem *domain.FolderMember) error {
	row := model.FolderMember{
		FolderID: mem.FolderID,
		UserID:   mem.UserID,
		Role:     string(mem.Role),
		Caps:     uint32(mem.Caps),
	}
	if err := s.Repo.Create(context.Background(), &row); err != nil {
		return err
	}
	*mem = row.ToDomain()

	return nil
}

func (s *FolderMember) Update(mem *domain.FolderMember) error {
	row := model.FolderMember{
		ID:       mem.ID,
		FolderID: mem.FolderID,
		UserID:   mem.UserID,
		Role:     string(mem.Role),
		Caps:     uint32(mem.Caps),
	}
	if err := s.Save(context.Background(), &row); err != nil {
		return err
	}
	*mem = row.ToDomain()

	return nil
}

func (s *FolderMember) Delete(folderID, userID uint) error {
	return s.DeleteWhere(context.Background(), "folder_id = ? AND user_id = ?", folderID, userID)
}

func (s *FolderMember) Find(folderID, userID uint) (*domain.FolderMember, error) {
	row, err := s.FindByWhere(context.Background(), "folder_id = ? AND user_id = ?", folderID, userID)
	if err != nil {
		return nil, mapDBErr(err)
	}
	out := row.ToDomain()

	return &out, nil
}

func (s *FolderMember) ListByFolder(folderID uint) ([]domain.FolderMember, error) {
	models, err := s.FindByWhereAndOrder(context.Background(), "folder_id = ?", "id ASC", folderID)
	if err != nil {
		return nil, err
	}

	out := make([]domain.FolderMember, 0, len(models))
	for _, m := range models {
		out = append(out, m.ToDomain())
	}

	return out, nil
}
