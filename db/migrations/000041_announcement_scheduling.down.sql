-- 000041_announcement_scheduling.down.sql
ALTER TABLE announcement_deliveries
    DROP CONSTRAINT announcement_deliveries_source_check;
ALTER TABLE announcement_deliveries
    ADD CONSTRAINT announcement_deliveries_source_check
    CHECK (source IN ('manual','birthday','roster','visitor_welcome'));

DROP INDEX IF EXISTS idx_announcements_scheduled;

ALTER TABLE announcements
    DROP CONSTRAINT IF EXISTS announcements_schedule_fields_check;
ALTER TABLE announcements
    DROP CONSTRAINT IF EXISTS announcements_channel_check;
ALTER TABLE announcements
    DROP CONSTRAINT IF EXISTS announcements_schedule_type_check;

ALTER TABLE announcements
    DROP COLUMN IF EXISTS run_count,
    DROP COLUMN IF EXISTS last_run_at,
    DROP COLUMN IF EXISTS schedule_offset_minutes,
    DROP COLUMN IF EXISTS schedule_event_id,
    DROP COLUMN IF EXISTS schedule_time,
    DROP COLUMN IF EXISTS schedule_at,
    DROP COLUMN IF EXISTS schedule_type,
    DROP COLUMN IF EXISTS channel,
    DROP COLUMN IF EXISTS audience_filter;
