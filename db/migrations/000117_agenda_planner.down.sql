-- 000117_agenda_planner.down.sql
DROP TABLE IF EXISTS programacao_exceptions;

DROP INDEX IF EXISTS idx_church_events_owner_ministry;
DROP INDEX IF EXISTS idx_church_events_series_occurrence;

ALTER TABLE church_events DROP CONSTRAINT IF EXISTS church_events_status_check;
ALTER TABLE church_events DROP COLUMN IF EXISTS occurrence_date;
ALTER TABLE church_events DROP COLUMN IF EXISTS status;
ALTER TABLE church_events DROP COLUMN IF EXISTS owner_ministry_id;

ALTER TABLE programacoes DROP COLUMN IF EXISTS owner_ministry_id;
