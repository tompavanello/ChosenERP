-- 000022_member_lifecycle.down.sql
DROP TABLE IF EXISTS member_history;

ALTER TABLE members DROP CONSTRAINT IF EXISTS members_exit_reason_check;
ALTER TABLE members DROP CONSTRAINT IF EXISTS members_status_check;
DROP INDEX IF EXISTS idx_members_tenant_status;

ALTER TABLE members DROP COLUMN IF EXISTS address;
ALTER TABLE members DROP COLUMN IF EXISTS exit_reason;
ALTER TABLE members DROP COLUMN IF EXISTS exited_at;
