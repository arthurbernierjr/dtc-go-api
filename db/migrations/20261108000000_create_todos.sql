-- +goose Up
CREATE TABLE IF NOT EXISTS todos (
  id      SERIAL PRIMARY KEY,
  subject TEXT NOT NULL,
  details TEXT NOT NULL
);

-- +goose Down
DROP TABLE todos;
