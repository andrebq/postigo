PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS collections(
    colid INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    UNIQUE(name)
);

CREATE TABLE IF NOT EXISTS objects(
    uid BLOG PRIMARY KEY NOT NULL,
    colid INTEGER NOT NULL,
    oid TEXT NOT NULL,
    content BLOB NOT NULL,
    seq INTEGER NOT NULL,
    updated_at_unixms INTEGER NOT NULL,
    created_at_unixms INTEGER NOT NULL,
    -- db_epoch is a internal field to help clients sync with
    -- the database
    db_epoch INTEGER NOT NULL,

    FOREIGN KEY(colid) REFERENCES collections(colid),
    UNIQUE (colid, oid)
);

CREATE INDEX IF NOT EXISTS idx_obj_by_epoch ON objects(db_epoch);

CREATE VIEW IF NOT EXISTS vw_objects AS
select o.uid, o.oid, c.name as collection, o.content, o.updated_at_unixms, o.created_at_unixms, o.db_epoch,
    o.colid
from objects o
inner join collections c
where o.colid = c.oid;

CREATE TABLE IF NOT EXISTS _internal_db_settings_int(
    name text primary key not null,
    value integer not null
);