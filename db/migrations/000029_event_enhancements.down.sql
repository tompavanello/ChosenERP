-- 000029_event_enhancements.down.sql
DROP TABLE IF EXISTS event_invitees;
ALTER TABLE church_events DROP COLUMN IF EXISTS estimated_cost;
ALTER TABLE church_events DROP COLUMN IF EXISTS attendance_mode;
