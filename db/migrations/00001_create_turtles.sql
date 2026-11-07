-- +goose Up
CREATE TABLE turtles (
  id   SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  age  INTEGER NOT NULL
);

-- +goose Down
DROP TABLE turtles;
