-- 000035_branch_cnpj.up.sql
-- CNPJ das filiais/congregacoes. O tenant ja tem `cnpj` (matriz); cada filial
-- pode ter o proprio CNPJ. Unico por tenant (dois tenants distintos podem ter
-- o mesmo numero? nao, mas o indice e por tenant para nao vazar a checagem).

ALTER TABLE branches ADD COLUMN cnpj text;

CREATE UNIQUE INDEX uq_branches_tenant_cnpj
    ON branches(tenant_id, cnpj) WHERE cnpj IS NOT NULL;
