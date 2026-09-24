-- 000047_suppliers.up.sql
-- Cadastro de FORNECEDORES (pessoas físicas ou jurídicas). No legado (IPI) os
-- fornecedores estavam misturados ao cadastro de membros (ncad_credsus_2 = 5 /
-- tipo 'FO'); aqui ganham tabela própria. CPF e CNPJ são ambos opcionais —
-- o fornecedor pode ser identificado só pelo nome.
CREATE TABLE IF NOT EXISTS suppliers (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid REFERENCES branches(id) ON DELETE SET NULL,
    name         text NOT NULL,
    trade_name   text,
    cpf          text,
    cnpj         text,
    email        text,
    phone        text,
    notes        text,
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_suppliers_tenant_branch ON suppliers(tenant_id, branch_id);
CREATE INDEX IF NOT EXISTS idx_suppliers_name ON suppliers(tenant_id, name);

-- RLS: mesma regra de benefactors — branch nulo = fornecedor global do tenant.
-- Sede (branch NULL) grava em qualquer filial; filial grava na própria.
ALTER TABLE suppliers ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS suppliers_sel ON suppliers;
CREATE POLICY suppliers_sel ON suppliers FOR SELECT
    USING (rls_read(tenant_id, branch_id, true));

DROP POLICY IF EXISTS suppliers_ins ON suppliers;
CREATE POLICY suppliers_ins ON suppliers FOR INSERT
    WITH CHECK (rls_write(tenant_id, branch_id, true));

DROP POLICY IF EXISTS suppliers_upd ON suppliers;
CREATE POLICY suppliers_upd ON suppliers FOR UPDATE
    USING (rls_write(tenant_id, branch_id, true))
    WITH CHECK (rls_write(tenant_id, branch_id, true));

DROP POLICY IF EXISTS suppliers_del ON suppliers;
CREATE POLICY suppliers_del ON suppliers FOR DELETE
    USING (rls_write(tenant_id, branch_id, true));
