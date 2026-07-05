-- name: GetCollectionIDByName :one
select c.colid
from collections c
where name = ?
limit 1;
-- name: GetObject :one
select content
from objects
where oid = ? and colid = ?;
-- name: GetObjectByCollection :one
select content
from vw_objects
where oid = ? and collection = ?;
-- name: PutObject :exec
insert into objects (uid, oid, colid, content, updated_at_unixms, created_at_unixms, db_epoch)
values (?, ?, ?, ?, ?, ?, ?)
on conflict (oid, colid) do
update
set content = excluded.content,
    updated_at_unixms = excluded.updated_at_unixms,
    db_epoch = excluded.db_epoch;
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