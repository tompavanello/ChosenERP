-- 000074_prayer_requests.up.sql
-- PEDIDOS DE ORACAO (app do membro, V1).
--
-- Um membro envia um pedido escolhendo quem pode ve-lo (visibilidade):
--   pastor | pastor_conselho | grupo | igreja
-- O mesmo pedido pode receber "estou orando" (reacao ANONIMA: ninguem ve QUEM
-- orou, so a contagem). O RLS isola por tenant/filial; o filtro fino de
-- visibilidade e aplicado em internal/prayer.

CREATE TABLE prayer_requests (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    member_id      uuid REFERENCES members(id) ON DELETE SET NULL,
    small_group_id uuid REFERENCES small_groups(id) ON DELETE SET NULL,
    author_name    text,
    body           text NOT NULL,
    visibility     text NOT NULL DEFAULT 'igreja',
    is_anonymous   boolean NOT NULL DEFAULT false,
    status         text NOT NULL DEFAULT 'open',
    answered_note  text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT prayer_requests_visibility_check CHECK (visibility IN ('pastor','pastor_conselho','grupo','igreja')),
    CONSTRAINT prayer_requests_status_check CHECK (status IN ('open','answered','archived')),
    CONSTRAINT prayer_requests_body_check CHECK (length(btrim(body)) > 0)
);

CREATE INDEX idx_prayer_requests_branch ON prayer_requests(tenant_id, branch_id, created_at DESC);
CREATE INDEX idx_prayer_requests_member ON prayer_requests(member_id);

ALTER TABLE prayer_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY prayer_requests_sel ON prayer_requests
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY prayer_requests_ins ON prayer_requests
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY prayer_requests_upd ON prayer_requests
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY prayer_requests_del ON prayer_requests
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- Reacoes "estou orando" (uma por identidade e pedido; anonima para os demais).
CREATE TABLE prayer_reactions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id         uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    prayer_request_id uuid NOT NULL REFERENCES prayer_requests(id) ON DELETE CASCADE,
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_id         uuid REFERENCES members(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (prayer_request_id, user_id)
);

CREATE INDEX idx_prayer_reactions_request ON prayer_reactions(prayer_request_id);

ALTER TABLE prayer_reactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY prayer_reactions_sel ON prayer_reactions
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY prayer_reactions_ins ON prayer_reactions
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY prayer_reactions_del ON prayer_reactions
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Permissoes: leitura e moderacao dos pedidos.
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('prayer.read',     'prayer', 'Ler pedidos de oracao'),
    ('prayer.moderate', 'prayer', 'Moderar pedidos de oracao')
ON CONFLICT (key) DO NOTHING;

-- super_admin / admin_sede: todas as permissoes (inclui as novas).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key IN ('super_admin', 'admin_sede')
  AND p.key IN ('prayer.read', 'prayer.moderate')
ON CONFLICT DO NOTHING;

-- Pastor/pastor da filial moderam; lideres apenas leem.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
    ('pastor_filial', ARRAY['prayer.read', 'prayer.moderate']),
    ('pastor',        ARRAY['prayer.read', 'prayer.moderate']),
    ('lider',         ARRAY['prayer.read'])
) AS v(role_key, perms)
JOIN roles r ON r.key = v.role_key
JOIN permissions p ON p.key = ANY(v.perms)
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- create_tenant: novos tenants ja nascem com as permissoes de oracao nos
-- papeis pastor/pastor_filial/lider (super_admin/admin_sede pegam "todas").
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION create_tenant(
    p_name text,
    p_slug text,
    p_plan text DEFAULT 'starter'
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tenant uuid;
    v_slug   text := lower(trim(p_slug));
    reserved text[] := ARRAY['app', 'www', 'api', 'admin', 'localhost'];
BEGIN
    IF p_name IS NULL OR trim(p_name) = '' THEN
        RAISE EXCEPTION 'nome da igreja obrigatorio' USING ERRCODE = '22023';
    END IF;
    IF v_slug !~ '^[a-z0-9][a-z0-9-]{1,38}$' THEN
        RAISE EXCEPTION 'slug invalido (minusculas, numeros e hifen; 2 a 39 chars)' USING ERRCODE = '22023';
    END IF;
    IF v_slug = ANY (reserved) THEN
        RAISE EXCEPTION 'slug reservado' USING ERRCODE = '22023';
    END IF;
    IF EXISTS (SELECT 1 FROM tenants WHERE slug = v_slug) THEN
        RAISE EXCEPTION 'slug ja existe' USING ERRCODE = '23505';
    END IF;

    INSERT INTO tenants (name, slug, plan)
    VALUES (trim(p_name), v_slug, COALESCE(NULLIF(trim(p_plan), ''), 'starter'))
    RETURNING id INTO v_tenant;

    INSERT INTO branches (tenant_id, name, slug, kind)
    VALUES (v_tenant, 'Sede Matriz', 'matriz', 'matriz');

    INSERT INTO roles (tenant_id, key, name, is_system) VALUES
        (v_tenant, 'super_admin',   'Super Admin',                true),
        (v_tenant, 'admin_sede',    'Admin da Sede',              true),
        (v_tenant, 'pastor_filial', 'Pastor da Filial',           true),
        (v_tenant, 'tesoureiro',    'Tesoureiro',                 true),
        (v_tenant, 'secretario',    'Secretario(a)',              true),
        (v_tenant, 'membro',        'Membro',                     true),
        (v_tenant, 'lider',         'Lider de Ministerio/Celula', true),
        (v_tenant, 'pastor',        'Pastor/Conselheiro',         true),
        (v_tenant, 'contador',      'Contador Externo',           true),
        (v_tenant, 'visitante',     'Visitante',                  true);

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM (VALUES
        ('pastor_filial', ARRAY['families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','prayer.read','prayer.moderate','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['families.read','members.read','ministries.read','ministries.write','prayer.read','reports.read']),
        ('pastor',        ARRAY['families.read','governance.read','members.read','ministries.read','prayer.read','prayer.moderate','reports.read']),
        ('contador',      ARRAY['finance.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;

GRANT EXECUTE ON FUNCTION create_tenant(text, text, text) TO chosenerp_app;
