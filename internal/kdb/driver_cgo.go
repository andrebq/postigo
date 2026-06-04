//go:build use_sqlite_cgo

package kdb

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type ()

func innerConn(fp string) (*sql.DB, error) {
	return sql.Open("sqlite3", "%v")
}
