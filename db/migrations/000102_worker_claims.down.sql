-- 000102_worker_claims.down.sql
DROP INDEX IF EXISTS idx_ann_dlv_claim;
DROP INDEX IF EXISTS idx_deliveries_claim;

ALTER TABLE announcement_deliveries
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS locked_at;

ALTER TABLE document_deliveries
    DROP COLUMN IF EXISTS next_attempt_at,
    DROP COLUMN IF EXISTS locked_at;
