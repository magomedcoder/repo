package gormutil

import (
	"context"

	"gorm.io/gorm"
)

type ITable interface {
	TableName() string
}

type Repo[T ITable] struct {
	model T
	Db    *gorm.DB
}

func NewRepo[T ITable](db *gorm.DB) Repo[T] {
	return Repo[T]{Db: db}
}

func (r *Repo[T]) DB(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return r.Db
	}

	return r.Db.WithContext(ctx)
}

func (r *Repo[T]) Model(ctx context.Context) *gorm.DB {
	return r.DB(ctx).Model(&r.model)
}

func applyOpts(db *gorm.DB, opts ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	for _, opt := range opts {
		db = opt(db)
	}

	return db
}

func (r *Repo[T]) BaseModel(ctx context.Context, opts ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	return applyOpts(r.Model(ctx), opts...)
}

func (r *Repo[T]) Create(ctx context.Context, data *T) error {
	return r.DB(ctx).Create(data).Error
}

func (r *Repo[T]) BatchCreate(ctx context.Context, data []*T) error {
	return r.DB(ctx).Create(data).Error
}

func (r *Repo[T]) CreateInBatches(ctx context.Context, items []T, batchSize int) error {
	return r.DB(ctx).CreateInBatches(items, batchSize).Error
}

func (r *Repo[T]) Save(ctx context.Context, data *T, opts ...func(*gorm.DB) *gorm.DB) error {
	return applyOpts(r.DB(ctx), opts...).Save(data).Error
}

func (r *Repo[T]) Txx(ctx context.Context, fc func(tx *gorm.DB) error) error {
	return r.DB(ctx).Transaction(fc)
}

func (r *Repo[T]) SelectFindByID(ctx context.Context, cols string, id uint) (*T, error) {
	var item T
	if err := r.DB(ctx).Select(cols).First(&item, id).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) SelectFindByWhere(ctx context.Context, cols string, where string, args ...any) (*T, error) {
	var item T
	if err := r.DB(ctx).Select(cols).Where(where, args...).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) FindAll(ctx context.Context, opts ...func(*gorm.DB) *gorm.DB) ([]T, error) {
	var items []T
	if err := applyOpts(r.Model(ctx), opts...).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repo[T]) ScanAll(ctx context.Context, opts ...func(*gorm.DB) *gorm.DB) ([]T, error) {
	var items []T
	if err := applyOpts(r.Model(ctx), opts...).Scan(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repo[T]) FindByID(ctx context.Context, id uint) (*T, error) {
	var item T
	if err := r.DB(ctx).First(&item, id).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) FindByIDs(ctx context.Context, ids []uint) ([]T, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	var items []T
	if err := r.DB(ctx).Find(&items, ids).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repo[T]) FindByWhere(ctx context.Context, where string, args ...any) (*T, error) {
	var item T
	if err := r.DB(ctx).Where(where, args...).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) FindByWhereAndOrder(ctx context.Context, where string, order string, args ...any) ([]T, error) {
	var items []T
	if err := r.DB(ctx).Where(where, args...).Order(order).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repo[T]) FindCustom(ctx context.Context, opts ...func(*gorm.DB) *gorm.DB) (*T, error) {
	var item T
	if err := applyOpts(r.Model(ctx), opts...).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) FindByWhereWithQuery(ctx context.Context, where string, args []any, opts ...func(*gorm.DB) *gorm.DB) (*T, error) {
	var item T
	db := applyOpts(r.Model(ctx).Where(where, args...), opts...)
	if err := db.First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repo[T]) QueryCount(ctx context.Context, where string, args ...any) (int64, error) {
	var count int64
	q := r.Model(ctx)
	if where != "" {
		q = q.Where(where, args...)
	}

	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repo[T]) QueryCountOpts(ctx context.Context, where string, args []any, opts ...func(*gorm.DB) *gorm.DB) (int64, error) {
	var count int64
	q := applyOpts(r.Model(ctx), opts...)
	if where != "" {
		q = q.Where(where, args...)
	}

	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repo[T]) QueryExist(ctx context.Context, where string, args ...any) (bool, error) {
	count, err := r.QueryCount(ctx, where, args...)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repo[T]) QueryExistOpts(ctx context.Context, where string, args []any, opts ...func(*gorm.DB) *gorm.DB) (bool, error) {
	count, err := r.QueryCountOpts(ctx, where, args, opts...)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *Repo[T]) UpdateByID(ctx context.Context, id any, data map[string]any) (int64, error) {
	res := r.Model(ctx).Where("id = ?", id).Updates(data)
	return res.RowsAffected, res.Error
}

func (r *Repo[T]) UpdateColumnByID(ctx context.Context, id any, column string, value any) (int64, error) {
	res := r.Model(ctx).Where("id = ?", id).UpdateColumn(column, value)
	return res.RowsAffected, res.Error
}

func (r *Repo[T]) UpdateColumnWhere(ctx context.Context, column string, value any, where string, args ...any) (int64, error) {
	res := r.Model(ctx).Where(where, args...).UpdateColumn(column, value)
	return res.RowsAffected, res.Error
}

func (r *Repo[T]) UpdateWhere(ctx context.Context, data map[string]any, where string, args ...any) (int64, error) {
	res := r.Model(ctx).Where(where, args...).Updates(data)
	return res.RowsAffected, res.Error
}

func (r *Repo[T]) Delete(ctx context.Context, conds ...any) error {
	return r.Model(ctx).Delete(&r.model, conds...).Error
}

func (r *Repo[T]) DeleteWhere(ctx context.Context, where string, args ...any) error {
	return r.Model(ctx).Where(where, args...).Delete(&r.model).Error
}

func (r *Repo[T]) DeleteWhereRows(ctx context.Context, where string, args ...any) (int64, error) {
	res := r.Model(ctx).Where(where, args...).Delete(&r.model)
	return res.RowsAffected, res.Error
}

func (r *Repo[T]) DeleteByID(ctx context.Context, id uint) error {
	return r.Model(ctx).Where("id = ?", id).Delete(&r.model).Error
}

func (r *Repo[T]) Pluck(ctx context.Context, column string, dest any, where string, args ...any) error {
	q := r.Model(ctx)
	if where != "" {
		q = q.Where(where, args...)
	}

	return q.Pluck(column, dest).Error
}

func (r *Repo[T]) PluckOpts(ctx context.Context, column string, dest any, opts ...func(*gorm.DB) *gorm.DB) error {
	return applyOpts(r.Model(ctx), opts...).Pluck(column, dest).Error
}

func WithUnscoped() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Unscoped() }
}

func WithOrder(order string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Order(order) }
}

func WithLimit(limit int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Limit(limit) }
}

func LimitOffset(limit, offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(offset).Limit(limit)
	}
}

func WithWhere(where string, args ...any) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Where(where, args...) }
}

func WithSelect(cols string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Select(cols) }
}

func WithOmit(cols ...string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Omit(cols...) }
}

func WithJoins(query string, args ...any) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Joins(query, args...) }
}
