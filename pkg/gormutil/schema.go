package gormutil

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

func VerifySchema(db *gorm.DB, models ...any) error {
	if db == nil {
		return fmt.Errorf("schema verify: db is nil")
	}

	m := db.Migrator()
	var missing []string

	for _, schemaModel := range models {
		if !m.HasTable(schemaModel) {
			stmt := &gorm.Statement{DB: db}
			_ = stmt.Parse(schemaModel)
			name := "unknown"
			if stmt.Schema != nil {
				name = stmt.Schema.Table
			}

			missing = append(missing, "table:"+name)
			continue
		}

		stmt := &gorm.Statement{
			DB: db,
		}
		if err := stmt.Parse(schemaModel); err != nil {
			return fmt.Errorf("schema verify: parse %T: %w", schemaModel, err)
		}

		for _, field := range stmt.Schema.Fields {
			if field.DBName == "" || field.IgnoreMigration {
				continue
			}

			if !m.HasColumn(schemaModel, field.DBName) {
				missing = append(missing, stmt.Schema.Table+"."+field.DBName)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("schema verify: missing %s", strings.Join(missing, ", "))
	}

	return nil
}
