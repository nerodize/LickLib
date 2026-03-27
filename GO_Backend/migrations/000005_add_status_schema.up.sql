-- ============================================
-- Migration 000005: Add Track Status & Difficulty
-- Idempotent (kann mehrfach laufen)
-- ============================================

-- Create difficulty enum (if not exists)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'difficulty') THEN
        CREATE TYPE difficulty AS ENUM ('EASY', 'MEDIUM', 'HARD', 'EXPERT');
    END IF;
END $$;

-- Create track_status enum (if not exists)
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'track_status') THEN
        CREATE TYPE track_status AS ENUM ('UPLOADING', 'READY', 'FAILED', 'PROCESSING');
    END IF;
END $$;

-- Add status column (if not exists)
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'tracks' AND column_name = 'status'
    ) THEN
        ALTER TABLE tracks ADD COLUMN status track_status NOT NULL DEFAULT 'UPLOADING';
    END IF;
END $$;

-- Create index (if not exists)
CREATE INDEX IF NOT EXISTS idx_tracks_status ON tracks(status);