-- +goose Up
ALTER TABLE users
ADD COLUMN is_chirpy_red BOOLEAN NOT NULL
DEFAULT false;

-- UPDATE users SET hashed_password = 'unset';

-- +goose Down
ALTER TABLE users
DROP COLUMN is_chirpy_red;
