//go:build use_sqlite_cgo

package sqlitedriver

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Open(fp string) (*sql.DB, error) {
	return sql.Open("sqlite3", "%v")
}
