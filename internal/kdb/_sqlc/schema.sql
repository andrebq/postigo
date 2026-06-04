PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS collections(
    colid INTEGER PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    UNIQUE(name)
);

CREATE TABLE IF NOT EXISTS key_val_history(
    colid INTEGER NOT NULL,
    val_uid BLOB NOT NULL,
    parent_val_uid BLOB,
    PRIMARY KEY(colid, val_uid),
    FOREIGN KEY(colid) REFERENCES collections(colid),
    FOREIGN KEY(colid, parent_val_uid) REFERENCES key_val_history(colid, val_uid)
);

CREATE TABLE IF NOT EXISTS key_values(
    colid INTEGER NOT NULL,
    val_uid BLOB NOT NULL,
    generation INTEGER NOT NULL,
    content BLOB NOT NULL,
    PRIMARY KEY(colid, val_uid),
    FOREIGN KEY(colid, val_uid) REFERENCES key_val_history(colid, val_uid)  -- fixed: key_history → key_val_history
);

CREATE TABLE IF NOT EXISTS key_paths(
    colid INTEGER NOT NULL,
    path TEXT NOT NULL,
    val_uid BLOB NOT NULL,
    PRIMARY KEY(colid, path),  -- fixed: path_hash → path (column doesn't exist)
    FOREIGN KEY(colid, val_uid) REFERENCES key_val_history(colid, val_uid)
);

CREATE VIEW IF NOT EXISTS viewkeyvalue AS
SELECT kp.colid  AS colid,
       kp.path   AS path,
       kp.val_uid AS val_uid,
       kv.content AS content,
       kv.generation AS generation
FROM key_paths kp
    INNER JOIN key_values kv ON kp.val_uid = kv.val_uid;