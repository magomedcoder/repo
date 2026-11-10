package migrate

import (
	"github.com/magomedcoder/repo/pkg/gormutil"
	"gorm.io/gorm"
)

func VerifySchema(db *gorm.DB) error {
	return gormutil.VerifySchema(db, bootstrapModels()...)
}
