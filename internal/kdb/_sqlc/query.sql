-- name: GetValue :one
select *
from viewkeyvalue
where path = ? and colid = ?
limit 1;
-- name: PutValue :exec
insert into key_values (colid, val_uid, content, generation)
values (?, ?, ?, ?);
-- name: PutValueHistory :exec
insert into key_val_history(colid, val_uid, parent_val_uid)
values (?, ?, ?);
-- name: PutKey :exec
insert into key_paths (colid, path, val_uid)
values (?, ?, ?)
on conflict (colid, path)
do update set val_uid = excluded.val_uid;
-- name: UpsertCollection :one
insert into collections(name)
values (?) on conflict (name) do
update
set name = excluded.name
returning colid;