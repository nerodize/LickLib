DROP TABLE IF EXISTS tracks CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TYPE IF EXISTS difficulty CASCADE;


CREATE TYPE difficulty AS ENUM ('EASY', 'MEDIUM', 'HARD', 'GOGGINS');

CREATE TABLE IF NOT EXISTS users (
    username text PRIMARY KEY,
    email text UNIQUE,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (username = lower(username)),
    CHECK (username ~ '^[a-z][a-z0-9_-]{2,29}$')
);

CREATE TABLE IF NOT EXISTS tracks (
  id integer GENERATED ALWAYS AS IDENTITY (START WITH 1000) PRIMARY KEY,
  username text NOT NULL REFERENCES users(username) ON DELETE CASCADE,
  title text NOT NULL,
  description text NOT NULL, 
  difficulty difficulty,
  file_ext text NOT NULL,
  size_bytes bigint NOT NULL CHECK (size_bytes >= 0 AND size_bytes < 1073741824),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE users
ADD CONSTRAINT uni_users_email UNIQUE (email);


