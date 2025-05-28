-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, user_id, expires_at, revoked_at, created_at, updated_at)
VALUES (
    $1,
    $2,
    NOW() + INTERVAL '60 days',
    NULL,
    NOW(),
    NOW()
)
RETURNING *;

-- name: GetUserFromRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1;


-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1;
