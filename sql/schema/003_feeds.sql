-- +goose Up
CREATE TABLE feeds (
  id UUID PRIMARY KEY,
  created_at TIMESTAMP without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  name text NOT NULL,
  url text UNIQUE NOT NULL,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;
