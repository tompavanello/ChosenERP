-- 000062_fin_account_groups.up.sql
-- Grupos de contas do plano de contas (ex.: Receitas, Despesas, Investimentos).
--
-- Decisao do cliente: permitir criar grupos e ASSOCIAR cada conta contabil a um
-- grupo para agrupar relatorios. O grupo e por tenant e pode ser global
-- (branch_id NULL), como as demais tabelas de catalogo do financeiro.
--
-- A associacao usa group_id em financial_categories com ON DELETE SET NULL:
-- excluir um grupo simplesmente desvincula as contas (nao as apaga).

CREATE TABLE financial_category_groups (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => global do tenant
    name        text NOT NULL,
    sort_order  int NOT NULL DEFAULT 0,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_fin_catgrp_tenant ON financial_category_groups(tenant_id, branch_id);

ALTER TABLE financial_category_groups ENABLE ROW LEVEL SECURITY;

CREATE POLICY fincatgrp_sel ON financial_category_groups
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY fincatgrp_all ON financial_category_groups
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- Vinculo da conta contabil ao grupo.
ALTER TABLE financial_categories
    ADD COLUMN group_id uuid REFERENCES financial_category_groups(id) ON DELETE SET NULL;

CREATE INDEX idx_fin_cat_group ON financial_categories(group_id);
