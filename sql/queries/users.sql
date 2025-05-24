-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_id(),
    NOW(),
    NOW(),
    $1
)
RETURNING *;
