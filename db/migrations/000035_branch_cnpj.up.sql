-- 000035_branch_cnpj.up.sql
-- CNPJ das filiais/congregações. O tenant já tem `cnpj` (matriz); cada filial
-- pode ter o próprio CNPJ. Único por tenant (dois tenants distintos podem ter
-- o mesmo número? não, mas o índice é por tenant para não vazar a checagem).

ALTER TABLE branches ADD COLUMN cnpj text;

CREATE UNIQUE INDEX uq_branches_tenant_cnpj
    ON branches(tenant_id, cnpj) WHERE cnpj IS NOT NULL;
