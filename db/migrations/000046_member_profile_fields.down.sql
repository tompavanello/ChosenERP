-- 000046_member_profile_fields.down.sql
ALTER TABLE members
    DROP COLUMN IF EXISTS nationality,
    DROP COLUMN IF EXISTS education,
    DROP COLUMN IF EXISTS notes;
