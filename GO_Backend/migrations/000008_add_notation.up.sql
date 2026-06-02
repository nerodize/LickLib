DROP TABLE IF EXISTS notations CASCADE;

CREATE TABLE notations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    track_id    UUID NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('TABS', 'NOTES')),
    storage_key TEXT NOT NULL,
    file_ext    TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    is_official BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notations_track  ON notations(track_id);
CREATE INDEX idx_notations_author ON notations(author_id);