package sqlite

import (
	"errors"

	"gorm.io/gorm"
	"magomedcoder.ru/repo/internal/domain"
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
