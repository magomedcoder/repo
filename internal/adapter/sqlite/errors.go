package sqlite

import (
	"errors"

	"github.com/magomedcoder/repo/internal/domain"
	"gorm.io/gorm"
)

func mapDBErr(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}

	return err
}
