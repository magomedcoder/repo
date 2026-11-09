package sqlite

import "github.com/magomedcoder/repo/pkg/sqlutil"

func escapeLike(s string) string {
	return sqlutil.EscapeLike(s)
}
