-- 000085_fin_tx_payables.down.sql
DROP INDEX IF EXISTS idx_fin_tx_payables;
ALTER TABLE financial_transactions
    DROP COLUMN IF EXISTS due_date,
    DROP COLUMN IF EXISTS paid_at,
    DROP COLUMN IF EXISTS cost_center;
