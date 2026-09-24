-- 000027_fix_fin_hash.up.sql
-- Corrige o hash-chain de financial_transactions: o corpo do hash concatenava
-- NEW.payment_method SEM COALESCE. Como payment_method e anulavel, um lancamento
-- sem forma de pagamento gerava body = NULL, digest(NULL) = NULL e o INSERT
-- falhava no NOT NULL de `hash` ("null value in column hash"). Passou
-- despercebido porque os seeds/fixtures sempre informavam a forma de pagamento.
--
-- Tambem padroniza a busca do hash anterior no formato tenant-scoped por
-- (created_at, id), evitando cadeia nao-temporal e vazamento entre tenants.

CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            COALESCE(NEW.payment_method,'') || '|' || COALESCE(NEW.account_id::text,'') || '|' ||
            NEW.occurred_at::text || '|' || COALESCE(NEW.description,'');
    NEW.prev_hash := (SELECT hash FROM financial_transactions
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;
