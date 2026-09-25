-- 000063_financial_audit_lock.down.sql
DROP TRIGGER IF EXISTS fin_audits_period_guard ON financial_audits;
DROP FUNCTION IF EXISTS fin_audits_period_guard();
