-- 000048_txn_supplier.down.sql
DROP INDEX IF EXISTS idx_fin_tx_supplier;
ALTER TABLE financial_transactions DROP COLUMN IF EXISTS supplier_id;
