-- 000050_financial_audit.up.sql
-- Relatorio de AUDITORIA ANALITICA do financeiro: um "run" de auditoria por
-- periodo que lista os lancamentos, permite marcar cada um como auditado
-- (individual ou geral) e, ao FECHAR, grava a assinatura do responsavel. Depois
-- de fechada a auditoria e imutavel (nao aceita manutencao) e vira documento.
CREATE TABLE IF NOT EXISTS financial_audits (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    title          text NOT NULL DEFAULT 'Auditoria financeira',
    period_start   date NOT NULL,
    period_end     date NOT NULL,
    status         text NOT NULL DEFAULT 'aberta' CHECK (status IN ('aberta','fechada')),
    notes          text,
    closed_at      timestamptz,
    closed_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    signer_name    text,
    signer_role    text,
    -- Hash do conteudo (itens + periodo) selado no fechamento.
    signature_hash text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT financial_audits_period_check CHECK (period_end >= period_start)
);

CREATE INDEX IF NOT EXISTS idx_fin_audit_tenant
    ON financial_audits(tenant_id, branch_id, period_start DESC);

-- Marcacao por lancamento. As linhas sao sincronizadas a partir dos
-- lancamentos do periodo enquanto a auditoria esta aberta.
CREATE TABLE IF NOT EXISTS financial_audit_items (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id       uuid NOT NULL REFERENCES financial_audits(id) ON DELETE CASCADE,
    transaction_id uuid NOT NULL REFERENCES financial_transactions(id) ON DELETE CASCADE,
    audited        boolean NOT NULL DEFAULT false,
    audited_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    audited_at     timestamptz,
    notes          text,
    UNIQUE (audit_id, transaction_id)
);

CREATE INDEX IF NOT EXISTS idx_fin_audit_items_audit ON financial_audit_items(audit_id);
CREATE INDEX IF NOT EXISTS idx_fin_audit_items_tx ON financial_audit_items(transaction_id);

-- Imutabilidade: auditoria fechada nao aceita mais marcasse; a propria auditoria
-- nao pode ser alterada depois de fechada.
CREATE OR REPLACE FUNCTION fin_audits_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'fechada' THEN
        RAISE EXCEPTION 'auditoria fechada: documento imutavel';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS fin_audits_no_update ON financial_audits;
CREATE TRIGGER fin_audits_no_update
BEFORE UPDATE ON financial_audits
FOR EACH ROW EXECUTE FUNCTION fin_audits_guard();

CREATE OR REPLACE FUNCTION fin_audit_items_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    aid uuid;
    st  text;
BEGIN
    aid := COALESCE(NEW.audit_id, OLD.audit_id);
    SELECT status INTO st FROM financial_audits WHERE id = aid;
    IF st = 'fechada' THEN
        RAISE EXCEPTION 'auditoria fechada: sem manutencao nos itens';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS fin_audit_items_no_closed ON financial_audit_items;
CREATE TRIGGER fin_audit_items_no_closed
BEFORE INSERT OR UPDATE OR DELETE ON financial_audit_items
FOR EACH ROW EXECUTE FUNCTION fin_audit_items_guard();

-- RLS
ALTER TABLE financial_audits ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fin_audit_sel ON financial_audits;
CREATE POLICY fin_audit_sel ON financial_audits
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_audit_ins ON financial_audits;
CREATE POLICY fin_audit_ins ON financial_audits
    FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_audit_upd ON financial_audits;
CREATE POLICY fin_audit_upd ON financial_audits
    FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_audit_del ON financial_audits;
CREATE POLICY fin_audit_del ON financial_audits
    FOR DELETE USING (rls_write(tenant_id, branch_id, false));

ALTER TABLE financial_audit_items ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fin_audit_items_sel ON financial_audit_items;
CREATE POLICY fin_audit_items_sel ON financial_audit_items
    FOR SELECT USING (EXISTS (
        SELECT 1 FROM financial_audits a
        WHERE a.id = audit_id AND rls_read(a.tenant_id, a.branch_id, false)));
DROP POLICY IF EXISTS fin_audit_items_all ON financial_audit_items;
CREATE POLICY fin_audit_items_all ON financial_audit_items
    FOR ALL USING (EXISTS (
        SELECT 1 FROM financial_audits a
        WHERE a.id = audit_id AND rls_write(a.tenant_id, a.branch_id, false)))
    WITH CHECK (EXISTS (
        SELECT 1 FROM financial_audits a
        WHERE a.id = audit_id AND rls_write(a.tenant_id, a.branch_id, false)));
