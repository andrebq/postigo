-- name: GetCollectionIDByName :one
select c.colid
from collections c
where name = ?
limit 1;
-- name: GetObjectSeq :one
select content, seq, updated_at_unixms, created_at_unixms
from objects
where oid = ? and colid = ?;
-- name: PutObjectSeq :one
insert into objects (uid, oid, colid, content, updated_at_unixms, created_at_unixms, db_epoch, seq)
values (?, ?, ?, ?, ?, ?, ?, 1)
on conflict (oid, colid) do
update
set content = excluded.content,
    updated_at_unixms = excluded.updated_at_unixms,
    db_epoch = excluded.db_epoch,
    seq = seq + 1
returning seq;

-- name: PutNew :exec
insert into objects (uid, oid, colid, content, updated_at_unixms, created_at_unixms, db_epoch, seq)
values (?, ?, ?, ?, ?, ?, ?, 1);

-- name: UpdateObject :execrows
update objects
set content = ?,
    updated_at_unixms = ?,
    db_epoch = ?,
    seq = seq + 1
where
    oid = ?
    and colid = ?
    and seq = ?;
-- name: UpsertCollection :one
insert into collections(name)
values (?) on conflict (name) do
update
set name = excluded.name
returning colid;

-- name: GetIntSetting :one
select value from _internal_db_settings_int
where name = ?
limit 1;

-- name: SetIntSetting :exec
insert into _internal_db_settings_int(name, value)
values (?, ?)
on conflict (name) do
update 
set value = excluded.value;