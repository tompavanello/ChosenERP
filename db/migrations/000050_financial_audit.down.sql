-- 000050_financial_audit.down.sql
DROP TRIGGER IF EXISTS fin_audit_items_no_closed ON financial_audit_items;
DROP FUNCTION IF EXISTS fin_audit_items_guard();
DROP TRIGGER IF EXISTS fin_audits_no_update ON financial_audits;
DROP FUNCTION IF EXISTS fin_audits_guard();
DROP TABLE IF EXISTS financial_audit_items;
DROP TABLE IF EXISTS financial_audits;
