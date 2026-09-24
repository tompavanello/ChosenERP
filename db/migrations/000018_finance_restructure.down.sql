-- 000018_finance_restructure.down.sql
DROP FUNCTION IF EXISTS fin_tx_hash();
DROP TRIGGER IF EXISTS fin_tx_hash ON financial_transactions;
-- Recria a função original (sem account_id) para restaurar o estado anterior.
CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            NEW.payment_method || '|' || NEW.occurred_at::text || '|' || COALESCE(NEW.description,'');
    NEW.prev_hash := (SELECT hash FROM financial_transactions WHERE id < NEW.id ORDER BY id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;
CREATE TRIGGER fin_tx_hash
BEFORE INSERT ON financial_transactions
FOR EACH ROW EXECUTE FUNCTION fin_tx_hash();

ALTER TABLE recurring_donations DROP COLUMN IF EXISTS account_id;
ALTER TABLE financial_transactions DROP COLUMN IF EXISTS account_id;
DROP TABLE IF EXISTS financial_accounts;
DROP TABLE IF EXISTS financial_classification_types;
