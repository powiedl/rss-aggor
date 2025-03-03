-- +goose Up

CREATE TABLE users (
  id UUID PRIMARY KEY,
  created_at TIMESTAMP without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL,
  name text NOT NULL
  --CONSTRAINT users_pkey PRIMARY KEY (id)
);

-- +goose Down

DROP TABLE users;