-- 000019_finance_attachments.up.sql
-- Anexos de documentos comprovatorios (fotos de recibos, notas fiscais)
-- associados a lancamentos financeiros. Como financial_transactions e
-- append-only, os anexos vivem em uma tabela propria com FK para a transacao.

CREATE TABLE financial_attachments (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid REFERENCES branches(id) ON DELETE SET NULL,
    transaction_id  uuid NOT NULL REFERENCES financial_transactions(id) ON DELETE CASCADE,
    file_name       text NOT NULL,
    file_url        text NOT NULL,
    content_type    text,
    file_size       int,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_fin_attach_tenant ON financial_attachments(tenant_id);
CREATE INDEX idx_fin_attach_txn ON financial_attachments(transaction_id);

ALTER TABLE financial_attachments ENABLE ROW LEVEL SECURITY;

CREATE POLICY finattach_sel ON financial_attachments
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY finattach_ins ON financial_attachments
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY finattach_del ON financial_attachments
  FOR DELETE USING (rls_write(tenant_id, branch_id, true));
