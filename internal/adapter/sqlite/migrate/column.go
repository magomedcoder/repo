package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func dropColumnIfExists(db *gorm.DB, table, column string) error {
	ok, err := TableHasColumn(db, table, column)
	if err != nil {
		return err
	}

	if !ok {
		return nil
	}

	stmt := fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", table, column)
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("drop column %s.%s: %w", table, column, err)
	}

	return nil
}

func TableHasColumn(db *gorm.DB, table, column string) (bool, error) {
	var n int64
	if err := db.Raw(`SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&n).Error; err != nil {
		return false, fmt.Errorf("pragma_table_info %s.%s: %w", table, column, err)
	}

	return n > 0, nil
}
