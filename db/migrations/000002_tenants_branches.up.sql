-- 000002_tenants_branches.up.sql
-- Nucleo multi-tenant: tenants (sedes/convencoes) e branches (filiais/congregacoes)

CREATE TABLE tenants (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL,
    slug          text NOT NULL UNIQUE,
    legal_name    text,
    cnpj          text,
    plan          text NOT NULL DEFAULT 'starter',
    locale        text NOT NULL DEFAULT 'pt-BR',
    timezone      text NOT NULL DEFAULT 'America/Sao_Paulo',
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE branches (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id     uuid REFERENCES branches(id) ON DELETE SET NULL, -- hierarquia Sede > Congregacoes
    name          text NOT NULL,
    slug          text NOT NULL,
    kind          text NOT NULL DEFAULT 'branch', -- branch | congregation | sub_congregation
    address       jsonb,
    geo           jsonb, -- { lat, lng }
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_branches_tenant ON branches(tenant_id);
CREATE INDEX idx_branches_parent ON branches(parent_id);
