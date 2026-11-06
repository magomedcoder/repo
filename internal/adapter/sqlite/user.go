package sqlite

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/internal/domain"
	"magomedcoder.ru/repo/pkg/gormutil"
)

type User struct {
	gormutil.Repo[model.User]
}

func NewUserRepo(db *gorm.DB) *User {
	return &User{Repo: gormutil.NewRepo[model.User](db)}
}

func (s *User) Create(user *domain.User) error {
	if !user.Role.Valid() {
		return fmt.Errorf("invalid role")
	}

	m := &model.User{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		AvatarPath:   user.AvatarPath,
		Role:         user.Role,
	}

	if err := s.Repo.Create(context.Background(), m); err != nil {
		return err
	}

	user.ID = m.ID
	user.Role = m.Role
	user.CreatedAt = m.CreatedAt
	user.UpdatedAt = m.UpdatedAt
	return nil
}

func (s *User) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	row, err := s.Repo.FindByID(ctx, id)
	if err != nil {
		return nil, mapDBErr(err)
	}

	return row.ToDomain(), nil
}

func (s *User) FindByIDs(ctx context.Context, ids []uint) (map[uint]*domain.User, error) {
	out := make(map[uint]*domain.User)
	if len(ids) == 0 {
		return out, nil
	}

	models, err := s.Repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range models {
		u := models[i].ToDomain()
		out[u.ID] = u
	}

	return out, nil
}

func (s *User) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	row, err := s.FindByWhere(ctx, "username = ?", username)
	if err != nil {
		return nil, mapDBErr(err)
	}

	return row.ToDomain(), nil
}

func (s *User) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := s.FindByWhere(ctx, "email = ?", email)
	if err != nil {
		return nil, mapDBErr(err)
	}

	return row.ToDomain(), nil
}

func (s *User) ExistsByUsername(username string) (bool, error) {
	return s.QueryExistOpts(context.Background(), "username = ?", []any{username}, gormutil.WithUnscoped())
}

func (s *User) ExistsByEmail(email string) (bool, error) {
	return s.QueryExistOpts(context.Background(), "email = ?", []any{email}, gormutil.WithUnscoped())
}

func (s *User) Update(user *domain.User) error {
	if !user.Role.Valid() {
		return fmt.Errorf("invalid role")
	}

	_, err := s.UpdateByID(context.Background(), user.ID, map[string]any{
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"avatar_path":   user.AvatarPath,
		"role":          user.Role,
	})
	return err
}

func (s *User) Delete(id uint) error {
	return s.DeleteByID(context.Background(), id)
}

func (s *User) List(filter domain.UserListFilter) ([]domain.User, int, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := max(filter.Offset, 0)
	ctx := context.Background()

	q := s.Model(ctx)
	if query := strings.TrimSpace(filter.Query); query != "" {
		like := "%" + escapeLike(strings.ToLower(query)) + "%"
		q = q.Where(`lower(username) LIKE ? ESCAPE '\' OR lower(email) LIKE ? ESCAPE '\'`, like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var models []model.User
	if err := q.Order("id ASC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	out := make([]domain.User, 0, len(models))
	for i := range models {
		out = append(out, *models[i].ToDomain())
	}

	return out, int(total), nil
}

func (s *User) CountByRole(role domain.Role) (int64, error) {
	if !role.Valid() {
		role = domain.RoleUser
	}

	return s.QueryCount(context.Background(), "role = ?", role)
}
