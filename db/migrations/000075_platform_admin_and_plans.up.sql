-- 000075_platform_admin_and_plans.up.sql
-- CONSOLE DA PLATAFORMA (dono do SaaS) + catalogo de planos.
--
-- Ate aqui "super_admin" era um papel POR IGREJA (escopo do tenant via RLS) e
-- tambem gate da plataforma (listar/criar igrejas). Isso permitia que qualquer
-- super_admin de igreja listasse/criasse tenants. Agora existe uma flag GLOBAL
-- de identidade (`users.is_platform_admin`) que distingue o operador do SaaS do
-- administrador de uma igreja.
--
-- `plans` e um catalogo global (sem tenant_id), como `permissions`. `tenants.plan`
-- continua sendo a chave textual; `tenants.limits` permite overrides por igreja.
-- O enforcement dos limites fica para uma fase posterior (aqui so gerencia/exibe).

ALTER TABLE users ADD COLUMN is_platform_admin boolean NOT NULL DEFAULT false;

-- Le a flag da propria identidade (users permite SELECT do proprio registro).
CREATE OR REPLACE FUNCTION is_platform_admin() RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT COALESCE((
        SELECT u.is_platform_admin FROM users u WHERE u.id = current_user_id()
    ), false)
$$;

-- ---------------------------------------------------------------------------
-- Catalogo de planos
-- ---------------------------------------------------------------------------
CREATE TABLE plans (
    key            text PRIMARY KEY,
    name           text NOT NULL,
    description    text,
    price_cents    integer NOT NULL DEFAULT 0,
    currency       text NOT NULL DEFAULT 'BRL',
    -- NULL => ilimitado
    max_members    integer,
    max_branches   integer,
    max_users      integer,
    max_storage_mb integer,
    features       jsonb NOT NULL DEFAULT '{}'::jsonb,
    is_active      boolean NOT NULL DEFAULT true,
    sort_order     integer NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE plans ENABLE ROW LEVEL SECURITY;
CREATE POLICY plans_sel ON plans
  FOR SELECT USING (is_system() OR is_platform_admin());
CREATE POLICY plans_ins ON plans
  FOR INSERT WITH CHECK (is_system() OR is_platform_admin());
CREATE POLICY plans_upd ON plans
  FOR UPDATE USING (is_system() OR is_platform_admin())
  WITH CHECK (is_system() OR is_platform_admin());
CREATE POLICY plans_del ON plans
  FOR DELETE USING (is_system() OR is_platform_admin());

INSERT INTO plans (key, name, description, price_cents, max_members, max_branches, max_users, max_storage_mb, sort_order) VALUES
    ('starter',    'Starter',    'Igrejas em inicio',        0,     300,  3,    15,   1024,   1),
    ('pro',        'Pro',        'Igrejas em crescimento',   0,    1500,  15,   60,   10240,  2),
    ('enterprise', 'Enterprise', 'Multi-filial / ilimitado', 0,    NULL,  NULL, NULL, NULL,   3)
ON CONFLICT (key) DO NOTHING;

-- Overrides de limites por igreja (jsonb). Vazio/NULL => usa o plano.
ALTER TABLE tenants ADD COLUMN limits jsonb;

-- Backfill: a conta de operacao do SaaS e platform admin.
UPDATE users SET is_platform_admin = true WHERE email = 'admin@erpchosen.com.br';
