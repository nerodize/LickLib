
-- 1. Alten Index löschen
DROP INDEX IF EXISTS idx_userid_title;

-- 2. Neuer PARTIAL Index (nur für status=READY)
CREATE UNIQUE INDEX idx_userid_title_ready 
ON tracks (user_id, title) 
WHERE status = 'READY';