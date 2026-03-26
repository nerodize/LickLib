CREATE TYPE track_status AS ENUM ('UPLOADING', 'READY', 'FAILED', 'PROCESSING');
-- hier jetzt noch ein Rollback?
ALTER TABLE tracks ADD COLUMN status track_status NOT NULL DEFAULT 'UPLOADING';
CREATE INDEX idx_tracks_status ON tracks(status);