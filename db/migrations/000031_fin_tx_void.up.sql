-- 000031_fin_tx_void.up.sql
-- Permite o ESTORNO de um lancamento sem quebrar o append-only.
--
-- financial_transactions e imutavel (trilha de auditoria/hash-chain). Em vez de
-- DELETE, o lancamento e ANULADO: permanece no historico, marcado como estornado,
-- e os relatorios passam a ignora-lo. A correcao (alteracao) e feita estornando o
-- original e lancando um novo.
--
-- O guardista passa a permitir UPDATE SOMENTE dos campos de anulacao; qualquer
-- mudanca em valor, tipo, data, conta, descricao etc. continua bloqueada.

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
        RAISE EXCEPTION 'financial_transactions is append-only (somente estorno e permitido)';
    END IF;
    RETURN NEW;
END;
$$;
