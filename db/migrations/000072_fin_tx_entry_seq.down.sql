-- 000072_fin_tx_entry_seq.down.sql
DROP INDEX IF EXISTS idx_fin_tx_entry_seq;
ALTER TABLE financial_transactions DROP COLUMN IF EXISTS entry_seq;
