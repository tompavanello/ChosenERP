-- 000017_member_enhancements.down.sql

ALTER TABLE members DROP COLUMN IF EXISTS baptism_location;
DROP INDEX IF EXISTS idx_visitors_converted;
DROP INDEX IF EXISTS idx_members_cpf;
DROP INDEX IF EXISTS idx_members_whatsapp;
