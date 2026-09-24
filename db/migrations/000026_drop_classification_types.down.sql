-- 000026_drop_classification_types.down.sql
-- Recria a tabela redundante a partir do plano de contas atual (mesmos codigos).
CREATE TABLE financial_classification_types (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid REFERENCES branches(id) ON DELETE SET NULL,
    direction   text NOT NULL,
    code        text NOT NULL,
    name        text NOT NULL,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, direction, code)
);

CREATE INDEX idx_fin_class_tenant ON financial_classification_types(tenant_id);
CREATE INDEX idx_fin_class_dir    ON financial_classification_types(tenant_id, direction);

ALTER TABLE financial_classification_types ENABLE ROW LEVEL SECURITY;

CREATE POLICY fintype_sel ON financial_classification_types
  USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY fintype_all ON financial_classification_types
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

INSERT INTO financial_classification_types (tenant_id, branch_id, direction, code, name)
SELECT tenant_id, branch_id, type, code, name
FROM financial_categories
WHERE is_active
ON CONFLICT (tenant_id, direction, code) DO NOTHING;
