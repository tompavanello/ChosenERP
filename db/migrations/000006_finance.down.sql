-- 000006_finance.down.sql
DROP TRIGGER IF EXISTS fin_tx_hash ON financial_transactions;
DROP TRIGGER IF EXISTS fin_tx_no_update ON financial_transactions;
DROP FUNCTION IF EXISTS fin_tx_hash();
DROP FUNCTION IF EXISTS fin_tx_guard();
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS transfers;
DROP TABLE IF EXISTS financial_transactions;
DROP TABLE IF EXISTS financial_categories;
