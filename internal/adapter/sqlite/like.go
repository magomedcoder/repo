package sqlite

import "magomedcoder.ru/repo/pkg/sqlutil"

func escapeLike(s string) string {
	return sqlutil.EscapeLike(s)
}
