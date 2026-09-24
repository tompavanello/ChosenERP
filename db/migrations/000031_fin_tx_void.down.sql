-- 000031_fin_tx_void.down.sql
DROP INDEX IF EXISTS idx_fin_tx_voided;
ALTER TABLE financial_transactions
    DROP COLUMN IF EXISTS voided_at,
    DROP COLUMN IF EXISTS voided_by,
    DROP COLUMN IF EXISTS void_reason;

CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'financial_transactions is append-only (no UPDATE/DELETE)';
END;
$$;
