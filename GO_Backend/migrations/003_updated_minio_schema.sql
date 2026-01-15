-- 1. Spalte hinzufügen (erstmal ohne NOT NULL, falls Daten existieren)
ALTER TABLE tracks ADD COLUMN storage_key TEXT;