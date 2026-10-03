-- 000086_split_rules.up.sql
-- Split automatico de repasses: liga/desliga por igreja + regras de percentual
-- por filial de destino. Aplicado ao lancar uma ENTRADA (income).
--
-- `transfer_rules` e configuracao por IGREJA (sem branch_id proprio): qualquer
-- filial le as regras para aplicar (rls_read com branch NULL), e a ESCrita e
-- gated por permissao na API. O destino referencia `branches`.
ALTER TABLE tenants
    ADD COLUMN split_enabled boolean NOT NULL DEFAULT false;

CREATE TABLE transfer_rules (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    destination_branch_id uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    name                  text,
    percent               numeric(5,2) NOT NULL,
    is_active             boolean NOT NULL DEFAULT true,
    created_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT transfer_rules_percent_check CHECK (percent > 0 AND percent <= 100)
);

CREATE INDEX idx_transfer_rules_tenant ON transfer_rules (tenant_id);

ALTER TABLE transfer_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE transfer_rules FORCE ROW LEVEL SECURITY;

-- Leitura/escrita no escopo do tenant (branch NULL + allow_global): qualquer
-- filial do tenant enxerga as regras ao aplicar o split no lancamento.
DROP POLICY IF EXISTS transfer_rules_sel ON transfer_rules;
CREATE POLICY transfer_rules_sel ON transfer_rules FOR SELECT
  USING (rls_read(tenant_id, NULL, true));
DROP POLICY IF EXISTS transfer_rules_all ON transfer_rules;
CREATE POLICY transfer_rules_all ON transfer_rules
  USING (rls_write(tenant_id, NULL, true))
  WITH CHECK (rls_write(tenant_id, NULL, true));
