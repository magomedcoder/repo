package sqlite

import (
	"errors"
	"strings"

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

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}
