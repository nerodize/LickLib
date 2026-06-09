-- ============================================
-- Migration 000009: Add Notation Status
-- ============================================

ALTER TABLE notations
    ALTER COLUMN storage_key DROP NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'notations' AND column_name = 'status'
    ) THEN
        ALTER TABLE notations ADD COLUMN status TEXT NOT NULL DEFAULT 'UPLOADING'
            CHECK (status IN ('UPLOADING', 'READY', 'FAILED'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_notations_status ON notations(status);
