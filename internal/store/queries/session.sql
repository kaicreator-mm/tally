-- name: InsertSession :exec
INSERT INTO session (token, expires_at)
VALUES ($1, $2);

-- name: GetSession :one
SELECT token
FROM session
WHERE token = $1 AND expires_at > now();

-- name: DeleteSession :exec
DELETE FROM session
WHERE token = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM session
WHERE expires_at <= now();
