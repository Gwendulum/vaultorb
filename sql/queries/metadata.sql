-- name: GetMetadata :one
SELECT value
FROM metadata
WHERE key = ?1;

-- name: SetMetadata :exec
INSERT INTO metadata (key, value)
VALUES (?1, ?2)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;


-- name: CheckMasterKey :one
SELECT
