-- +goose Up
CREATE TABLE chirps (
    id uuid PRIMARY KEY,
    body TEXT NOT NULL,
    user_id uuid NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;
