package sqlite

import (
	"github.com/magomedcoder/repo/internal/adapter/sqlite/model"
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

func bootstrapModels() []any {
	return model.BootstrapModels()
}

func VerifySchema(db *gorm.DB) error {
	return gormutil.VerifySchema(db, bootstrapModels()...)
}
