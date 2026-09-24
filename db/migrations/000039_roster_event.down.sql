-- 000039_roster_event.down.sql
DROP INDEX IF EXISTS idx_rosters_event_kind;
ALTER TABLE rosters DROP COLUMN IF EXISTS generated_event;
ALTER TABLE rosters DROP COLUMN IF EXISTS event_kind_id;
