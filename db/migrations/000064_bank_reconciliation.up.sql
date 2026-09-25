-- 000064_bank_reconciliation.up.sql
-- CONCILIACAO BANCARIA: importa o extrato do banco (OFX ou CSV) de uma conta,
-- casa cada lancamento do extrato com os lancamentos do ERP (por valor, tipo e
-- data +/- tolerancia), aponta as divergencias e permite gerar os lancamentos
-- faltantes (a partir de uma entrada do extrato sem correspondencia).
CREATE TABLE IF NOT EXISTS bank_statement_imports (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    account_id      uuid REFERENCES financial_accounts(id) ON DELETE SET NULL,
    filename        text NOT NULL DEFAULT '',
    format          text NOT NULL DEFAULT 'ofx' CHECK (format IN ('ofx','csv')),
    period_start    date,
    period_end      date,
    status          text NOT NULL DEFAULT 'aberta' CHECK (status IN ('aberta','conciliado')),
    total_entries   int NOT NULL DEFAULT 0,
    matched_entries int NOT NULL DEFAULT 0,
    missing_entries int NOT NULL DEFAULT 0,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bank_import_tenant
    ON bank_statement_imports(tenant_id, branch_id, created_at DESC);

CREATE TABLE IF NOT EXISTS bank_statement_entries (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    import_id      uuid NOT NULL REFERENCES bank_statement_imports(id) ON DELETE CASCADE,
    posted_at      date NOT NULL,
    amount         numeric(14,2) NOT NULL CHECK (amount > 0),
    direction      text NOT NULL CHECK (direction IN ('income','expense')),
    memo           text,
    fitid          text,
    status         text NOT NULL DEFAULT 'pendente'
        CHECK (status IN ('pendente','conciliado','ignorado','lancamento_gerado')),
    transaction_id uuid REFERENCES financial_transactions(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_bank_entries_import ON bank_statement_entries(import_id);
CREATE INDEX IF NOT EXISTS idx_bank_entries_tx ON bank_statement_entries(transaction_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_entries_fitid
    ON bank_statement_entries(import_id, fitid) WHERE fitid IS NOT NULL;

-- RLS
ALTER TABLE bank_statement_imports ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS bank_import_sel ON bank_statement_imports;
CREATE POLICY bank_import_sel ON bank_statement_imports
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS bank_import_all ON bank_statement_imports;
CREATE POLICY bank_import_all ON bank_statement_imports
    FOR ALL USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

ALTER TABLE bank_statement_entries ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS bank_entries_sel ON bank_statement_entries;
CREATE POLICY bank_entries_sel ON bank_statement_entries
    FOR SELECT USING (EXISTS (
        SELECT 1 FROM bank_statement_imports b
        WHERE b.id = import_id AND rls_read(b.tenant_id, b.branch_id, false)));
DROP POLICY IF EXISTS bank_entries_all ON bank_statement_entries;
CREATE POLICY bank_entries_all ON bank_statement_entries
    FOR ALL USING (EXISTS (
        SELECT 1 FROM bank_statement_imports b
        WHERE b.id = import_id AND rls_write(b.tenant_id, b.branch_id, false)))
    WITH CHECK (EXISTS (
        SELECT 1 FROM bank_statement_imports b
        WHERE b.id = import_id AND rls_write(b.tenant_id, b.branch_id, false)));
