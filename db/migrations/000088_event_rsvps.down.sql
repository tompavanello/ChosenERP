-- 000088_event_rsvps.down.sql
DROP POLICY IF EXISTS event_rsvps_all ON event_rsvps;
DROP POLICY IF EXISTS event_rsvps_sel ON event_rsvps;
DROP TABLE IF EXISTS event_rsvps;
