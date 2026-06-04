package kdb

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	_ "embed"

	"github.com/andrebq/postigo/internal/kdb/sqlstore"
)

//go:embed _sqlc/schema.sql
var sqliteSchema []byte

type (
	DB struct {
		conn *sql.DB
	}

	Collection struct {
		name  string
		colid int64
		db    *DB
	}
)

func Open(fp string) (*DB, error) {
	db := &DB{}
	var err error
	db.conn, err = innerConn(fp)
	if err != nil {
		return nil, err
	}
	err = initDb(db.conn)
	if err != nil {
		db.conn.Close()
		return nil, err
	}
	return db, err
}

func initDb(conn *sql.DB) error {
	_, err := conn.Exec(string(sqliteSchema))
	return err
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Collection(ctx context.Context, name string) (*Collection, error) {
	q, commit, rollback, err := getTx(ctx, d.conn)
	defer rollback()
	if err != nil {
		return nil, err
	}
	colid, err := q.UpsertCollection(ctx, name)
	if err != nil {
		return nil, err
	}
	err = commit()
	if err != nil {
		return nil, err
	}
	return &Collection{
		name:  name,
		colid: colid,
		db:    d,
	}, nil
}

// Overwrite the value of key, creates a new entry if not found
func (c *Collection) Overwrite(ctx context.Context, key string, value []byte) error {
	sum := sha256.Sum256(value)
	q, commit, rollback, err := getTx(ctx, c.db.conn)
	if err != nil {
		return err
	}
	defer rollback()
	head, err := q.GetValue(ctx, sqlstore.GetValueParams{
		Path:  key,
		Colid: c.colid,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	nextGen := head.Generation + 1
	if err := q.PutValueHistory(ctx, sqlstore.PutValueHistoryParams{
		Colid:        c.colid,
		ParentValUid: head.ValUid,
		ValUid:       sum[:],
	}); err != nil {
		return fmt.Errorf("unable to update history table: %w", err)
	}
	if err := q.PutValue(ctx, sqlstore.PutValueParams{
		Colid:      c.colid,
		Generation: nextGen,
		ValUid:     sum[:],
		Content:    value,
	}); err != nil {
		return fmt.Errorf("unable to update value table: %w", err)
	}
	if err := q.PutKey(ctx, sqlstore.PutKeyParams{
		Colid:  c.colid,
		Path:   key,
		ValUid: sum[:],
	}); err != nil {
		return fmt.Errorf("unable to update key table: %w", err)
	}
	return commit()
}

// Lookup the value and generation of the given key.
//
// Returns nil value if the key does not exist
func (c *Collection) Lookup(ctx context.Context, key string) ([]byte, int64, error) {
	q := sqlstore.New(c.db.conn)
	head, err := q.GetValue(ctx, sqlstore.GetValueParams{
		Colid: c.colid,
		Path:  key,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	return head.Content, head.Generation, nil
}

func getTx(ctx context.Context, conn *sql.DB) (
	q *sqlstore.Queries,
	commit func() error,
	rollback func() error,
	err error) {
	var tx *sql.Tx
	tx, err = conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelLinearizable})
	if err != nil {
		return nil, noop, noop, err
	}
	return sqlstore.New(tx),
		func() error {
			if tx != nil {
				err := tx.Commit()
				tx = nil
				return err
			}
			return errors.New("Tx already close")
		}, func() error {
			if tx != nil {
				err := tx.Rollback()
				tx = nil
				return err
			}
			return errors.New("Tx already close")
		}, nil
}

func noop() error { return nil }
