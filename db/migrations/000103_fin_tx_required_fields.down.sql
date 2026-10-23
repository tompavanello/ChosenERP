-- 000103_fin_tx_required_fields.down.sql
DROP TRIGGER IF EXISTS fin_tx_require_account ON financial_transactions;
DROP FUNCTION IF EXISTS fin_tx_require_account();
