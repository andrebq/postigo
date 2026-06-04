//go:build !use_sqlite_cgo

package kdb

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func innerConn(fp string) (*sql.DB, error) {
	return sql.Open("sqlite", fp)
}
