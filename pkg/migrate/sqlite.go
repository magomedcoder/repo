package migrate

import (
	"fmt"
	"io"
	"os"
	"time"

	"gorm.io/gorm"
)

func IsEmptySQLite(db *gorm.DB) (bool, error) {
	var name string
	err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' LIMIT 1`).Scan(&name).Error
	if err != nil {
		return false, err
	}

	return name == "", nil
}

func TableExists(db *gorm.DB, table string) (bool, error) {
	var name string
	err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name).Error
	return name != "", err
}

func ColumnExists(db *gorm.DB, table, column string) (bool, error) {
	rows, err := db.Raw(fmt.Sprintf(`PRAGMA table_info(%s)`, table)).Rows()
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}

		if name == column {
			return true, nil
		}
	}

	return false, rows.Err()
}

func AddColumnIfMissing(db *gorm.DB, table, column, alterSQL string) error {
	ok, err := ColumnExists(db, table, column)
	if err != nil {
		return err
	}

	if ok {
		return nil
	}

	return db.Exec(alterSQL).Error
}

func BackupSQLite(dbPath string) (string, error) {
	if dbPath == "" || dbPath == ":memory:" {
		return "", nil
	}

	src, _ := splitDSN(dbPath)
	if src == "" || src == ":memory:" {
		return "", nil
	}

	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	dst := fmt.Sprintf("%s.bak.%s", src, time.Now().UTC().Format("20060102T150405Z"))
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}

	return dst, nil
}

func splitDSN(dsn string) (file string, query string) {
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == '?' {
			return dsn[:i], dsn[i+1:]
		}
	}

	return dsn, ""
}
