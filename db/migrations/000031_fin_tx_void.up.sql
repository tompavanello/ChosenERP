-- 000031_fin_tx_void.up.sql
-- Permite o ESTORNO de um lançamento sem quebrar o append-only.
--
-- financial_transactions é imutável (trilha de auditoria/hash-chain). Em vez de
-- DELETE, o lançamento é ANULADO: permanece no histórico, marcado como estornado,
-- e os relatórios passam a ignorá-lo. A correção (alteração) é feita estornando o
-- original e lançando um novo.
--
-- O guardista passa a permitir UPDATE SOMENTE dos campos de anulação; qualquer
-- mudança em valor, tipo, data, conta, descrição etc. continua bloqueada.

ALTER TABLE financial_transactions
    ADD COLUMN voided_at    timestamptz,
    ADD COLUMN voided_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN void_reason  text;

CREATE INDEX idx_fin_tx_voided ON financial_transactions(tenant_id, voided_at) WHERE voided_at IS NOT NULL;

CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'financial_transactions is append-only (no DELETE; use estorno)';
    END IF;
    IF NEW.tenant_id       IS DISTINCT FROM OLD.tenant_id
       OR NEW.branch_id    IS DISTINCT FROM OLD.branch_id
       OR NEW.category_id  IS DISTINCT FROM OLD.category_id
       OR NEW.type         IS DISTINCT FROM OLD.type
       OR NEW.amount       IS DISTINCT FROM OLD.amount
       OR NEW.currency     IS DISTINCT FROM OLD.currency
       OR NEW.donor_member_id IS DISTINCT FROM OLD.donor_member_id
       OR NEW.benefactor_id   IS DISTINCT FROM OLD.benefactor_id
       OR NEW.payment_method  IS DISTINCT FROM OLD.payment_method
       OR NEW.is_anonymous    IS DISTINCT FROM OLD.is_anonymous
       OR NEW.occurred_at     IS DISTINCT FROM OLD.occurred_at
       OR NEW.description     IS DISTINCT FROM OLD.description
       OR NEW.account_id      IS DISTINCT FROM OLD.account_id
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente estorno é permitido)';
    END IF;
    RETURN NEW;
END;
$$;
