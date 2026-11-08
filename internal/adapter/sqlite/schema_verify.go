package sqlite

import (
	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/adapter/sqlite/model"
	"magomedcoder.ru/repo/pkg/gormutil"
)

func bootstrapModels() []any {
	return model.BootstrapModels()
}

func VerifySchema(db *gorm.DB) error {
	return gormutil.VerifySchema(db, bootstrapModels()...)
}
