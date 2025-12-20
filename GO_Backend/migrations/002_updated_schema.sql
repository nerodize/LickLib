DROP TABLE IF EXISTS tracks CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS notations CASCADE;

DROP TYPE IF EXISTS difficulty CASCADE;


-- ===============================================
-- USERS TABLE
-- ===============================================

CREATE TYPE difficulty AS ENUM ('EASY', 'MEDIUM', 'HARD', 'GOGGINS');

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT UNIQUE,
    password_hash TEXT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);



-- ===============================================
-- TRACKS TABLE
-- ===============================================

CREATE TABLE IF NOT EXISTS tracks (
    id SERIAL PRIMARY KEY,
    
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    title TEXT NOT NULL,
    description TEXT NOT NULL,

    difficulty difficulty,  -- oder ENUM, wenn du das willst
    file_ext TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Unique constraint: Ein User darf denselben Titel nicht mehrfach anlegen
CREATE UNIQUE INDEX IF NOT EXISTS idx_userid_title 
ON tracks (user_id, title);

-- ===============================================
-- NOTATIONS TABLE
-- ===============================================

-- NOTATION TYPE (optional enum)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'notationtype') THEN
        CREATE TYPE notationtype AS ENUM ('TABS', 'NOTES');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS notations (
    id SERIAL PRIMARY KEY,

    track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    type notationtype NOT NULL,
    content TEXT NOT NULL,

    file_ext TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Optional: schneller auffinden aller Notationen eines Tracks
CREATE INDEX IF NOT EXISTS idx_notations_track ON notations(track_id);

-- Optional: schneller auffinden aller Notationen eines Users
CREATE INDEX IF NOT EXISTS idx_notations_author ON notations(author_id);







