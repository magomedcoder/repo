package migrate

import (
	"context"

	"github.com/magomedcoder/repo/pkg/sqliteutil"
	"gorm.io/gorm"
)

func Run(db *gorm.DB, ensureCodeIndex func(context.Context) error) error {
	empty, err := sqliteutil.IsEmpty(db)
	if err != nil {
		return err
	}

	bootstrapped := false
	if empty {
		if err := bootstrapSchema(db); err != nil {
			return err
		}

		if ensureCodeIndex != nil {
			if err := ensureCodeIndex(context.Background()); err != nil {
				return err
			}
		}

		bootstrapped = true
	}

	applied, err := applySchemaUpgrades(db)
	if err != nil {
		return err
	}

	if bootstrapped || applied > 0 {
		return VerifySchema(db)
	}

	return nil
}
