package migrations

import "magomedcoder.ru/repo/pkg/migrate"

func All() []migrate.Migration {
	return []migrate.Migration{
		v1(),
	}
}
