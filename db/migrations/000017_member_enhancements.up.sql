-- 000017_member_enhancements.up.sql
-- baptism_location for members, indexes optimization

ALTER TABLE members ADD COLUMN baptism_location text;

-- Index for faster visitor->member lookups
CREATE INDEX IF NOT EXISTS idx_visitors_converted ON visitors(converted_to_member_id) WHERE converted_to_member_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_members_cpf ON members(cpf);
CREATE INDEX IF NOT EXISTS idx_members_whatsapp ON members(whatsapp);
