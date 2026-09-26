-- name: CreateEntry :one
INSERT INTO entries (domain, username, encrypted_password, created_at)
VALUES (
    ?1,
    ?2,
    ?3,
	?4
)
RETURNING *;

-- name: GetEntry :one
SELECT encrypted_password
FROM entries
WHERE domain = ?1 AND username = ?2;

-- name: DeleteEntry :one
DELETE FROM entries
WHERE domain = ?1 AND username = ?2
RETURNING *;

-- name: ListEntries :many
SELECT id, domain, username, encrypted_password, created_at
FROM entries
ORDER BY domain ASC, username ASC;

-- name: ClearEntries :exec
DELETE FROM entries;



-- name: RestoreEntry :one
INSERT INTO entries (id, domain, username, encrypted_password, created_at)
VALUES (
    ?1,
    ?2,
    ?3,
	?4,
    ?5
)
RETURNING *;
