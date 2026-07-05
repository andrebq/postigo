//go:build !use_sqlite_cgo

package sqlitedriver

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(fp string) (*sql.DB, error) {
	return sql.Open("sqlite", fp)
}
