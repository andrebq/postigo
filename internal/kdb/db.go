package kdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	_ "embed"

	"github.com/andrebq/postigo/internal/kdb/sqlitedriver"
	"github.com/andrebq/postigo/internal/kdb/sqlstore"
	"github.com/google/uuid"
)

//go:embed _sqlc/schema.sql
var sqliteSchema []byte

type (
	DB struct {
		conn *sql.DB

		settings struct {
			epoch int64
		}
	}

	WithID interface {
		GetID() string
	}

	Collection[T WithID] struct {
		colUID         uuid.UUID
		objectsRootUID uuid.UUID
		name           string
		colid          int64
		db             *DB
	}
)

var (
	rootUID        = uuid.MustParse("6a8fcbd8-32fd-425a-a0f1-38de4460cb1a")
	collectionsUID = uuid.NewSHA1(rootUID, []byte("collections"))

	errConcurrentUpdate = errors.New("concurrent update during CAS operation")
)

func IsConcurrentUpdate(err error) bool {
	return errors.Is(err, errConcurrentUpdate)
}

func Open(fp string) (*DB, error) {
	db := &DB{}
	var err error
	db.conn, err = sqlitedriver.Open(fp)
	if err != nil {
		return nil, err
	}
	err = initDb(db.conn)
	if err != nil {
		db.conn.Close()
		return nil, err
	}
	err = db.incrementEpoch(context.Background())
	if err != nil {
		return nil, err
	}
	return db, err
}

func (db *DB) incrementEpoch(ctx context.Context) error {
	q := sqlstore.New(db.conn)
	epoch, err := q.GetIntSetting(ctx, "epoch")
	if errors.Is(err, sql.ErrNoRows) {
		epoch = 0
	}
	epoch++
	db.settings.epoch = epoch
	return q.SetIntSetting(ctx, sqlstore.SetIntSettingParams{
		Name:  "epoch",
		Value: epoch,
	})
}

func initDb(conn *sql.DB) error {
	_, err := conn.Exec(string(sqliteSchema))
	return err
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func GetCollection[T WithID](ctx context.Context, db *DB, name string) (*Collection[T], error) {
	q, commit, rollback, err := getTx(ctx, db)
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
	colUID := uuid.NewSHA1(collectionsUID, []byte(name))
	return &Collection[T]{
		colUID:         colUID,
		objectsRootUID: uuid.NewSHA1(colUID, []byte("objects")),
		name:           name,
		colid:          colid,
		db:             db,
	}, nil
}

func (c *Collection[T]) computeUid(oid string) []byte {
	id := uuid.NewSHA1(c.objectsRootUID, []byte(oid))
	return id[:]
}

func (c *Collection[T]) Lookup(ctx context.Context, out *T, id string) error {
	_, err := c.lookup(ctx, out, id)
	return err
}

func (c *Collection[T]) lookup(ctx context.Context, out *T, id string) (int64, error) {
	var raw sqlstore.GetObjectSeqRow
	err := readTransaction(ctx, c.db, func(q *sqlstore.Queries) error {
		var err error
		raw, err = q.GetObjectSeq(ctx, sqlstore.GetObjectSeqParams{
			Colid: c.colid,
			Oid:   id,
		})
		return err
	})
	if err != nil {
		return -1, err
	}
	return raw.Seq, json.Unmarshal(raw.Content, out)
}

func (c *Collection[T]) CAS(ctx context.Context, id string, updateFn func(*T) (*T, error)) (bool, error) {
	var old T
	seq, err := c.lookup(ctx, &old, id)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	} else if err != nil {
		return false, err
	}
	updated, err := updateFn(&old)
	if err != nil {
		return false, err
	}
	if seq == -1 {
		err = c.putNew(ctx, updated)
		return err == nil, err
	} else {
		err = c.updateSeq(ctx, seq, updated)
		return err == nil, err
	}
}

func (c *Collection[T]) updateSeq(ctx context.Context, seq int64, content *T) error {
	id := (*content).GetID()
	buf, err := json.Marshal(content)
	if err != nil {
		return err
	}
	err = transactional(ctx, c.db, func(q *sqlstore.Queries) error {
		now := dbnow()
		affectedRows, err := q.UpdateObject(ctx, sqlstore.UpdateObjectParams{
			Colid:           c.colid,
			Oid:             id,
			Content:         buf,
			UpdatedAtUnixms: now,
			DbEpoch:         c.db.settings.epoch,
			Seq:             seq,
		})
		if err == nil && affectedRows == 0 {
			err = errConcurrentUpdate
		}
		return err
	})
	return err
}

func (c *Collection[T]) putNew(ctx context.Context, content *T) error {
	id := (*content).GetID()
	buf, err := json.Marshal(content)
	if err != nil {
		return err
	}
	uid := c.computeUid(id)
	return transactional(ctx, c.db, func(q *sqlstore.Queries) error {
		now := dbnow()
		err = q.PutNew(ctx, sqlstore.PutNewParams{
			Colid:           c.colid,
			Uid:             uid,
			Oid:             id,
			Content:         buf,
			UpdatedAtUnixms: now,
			CreatedAtUnixms: now,
			DbEpoch:         c.db.settings.epoch,
		})
		return err
	})
}

func (c *Collection[T]) Put(ctx context.Context, content *T) error {
	id := (*content).GetID()
	buf, err := json.Marshal(content)
	if err != nil {
		return err
	}
	return transactional(ctx, c.db, func(q *sqlstore.Queries) error {
		now := dbnow()
		_, err = q.PutObjectSeq(ctx, sqlstore.PutObjectSeqParams{
			Uid:             c.computeUid(id),
			Colid:           c.colid,
			Content:         buf,
			Oid:             id,
			CreatedAtUnixms: now,
			UpdatedAtUnixms: now,
			DbEpoch:         c.db.settings.epoch,
		})
		return err
	})
}

func dbnow() int64 {
	return time.Now().Truncate(time.Millisecond).UnixMilli()
}

func readTransaction(ctx context.Context, db *DB, fn func(q *sqlstore.Queries) error) error {
	tx, err := db.conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return fn(sqlstore.New(tx))
}

func transactional(ctx context.Context, db *DB, fn func(q *sqlstore.Queries) error) error {
	q, commit, rollback, err := getTx(ctx, db)
	if err != nil {
		return err
	}
	defer rollback()
	err = fn(q)
	if err != nil {
		rollback()
		return err
	}
	return commit()
}

func getTx(ctx context.Context, db *DB) (
	q *sqlstore.Queries,
	commit func() error,
	rollback func() error,
	err error) {
	var tx *sql.Tx
	tx, err = db.conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelLinearizable})
	if err != nil {
		return nil, noop, noop, err
	}
	return sqlstore.New(tx),
		func() error {
			if tx != nil {
				err := tx.Commit()
				tx = nil
				if err != nil {
					_ = db.incrementEpoch(ctx)
				}
				return err
			}
			return errors.New("Tx already close")
		}, func() error {
			if tx != nil {
				err := tx.Rollback()
				if err != nil {
					slog.DebugContext(ctx, "Rollback error", "error", err)
				}
				tx = nil
				return err
			}
			return errors.New("Tx already close")
		}, nil
}

func noop() error { return nil }
