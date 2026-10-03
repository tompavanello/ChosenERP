-- 000087_announcements_permission.up.sql
-- Permissao dedicada de Comunicados (antes: apenas feature-gated por plano).
-- `announcements.read`/`announcements.write` controlam leitura/disparo dos
-- comunicados e automacoes de WhatsApp; sem ela a rota devolve 403.
INSERT INTO permissions (key, module, name) VALUES
    ('announcements.read',  'communication', 'Ler comunicados'),
    ('announcements.write', 'communication', 'Gerir e disparar comunicados')
ON CONFLICT (key) DO NOTHING;

-- admin_sede/super_admin: todas as permissoes (inclui as novas).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key IN ('admin_sede', 'super_admin')
  AND p.key IN ('announcements.read', 'announcements.write')
ON CONFLICT DO NOTHING;

-- Perfis que gerenciam comunicacao.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
    ('secretario',    ARRAY['announcements.read','announcements.write']),
    ('pastor_filial', ARRAY['announcements.read','announcements.write']),
    ('pastor',        ARRAY['announcements.read','announcements.write']),
    ('lider',         ARRAY['announcements.read','announcements.write'])
) AS v(role_key, perms)
JOIN roles r ON r.key = v.role_key
CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
JOIN permissions p ON p.key = perm.key
ON CONFLICT DO NOTHING;

-- create_tenant passa a conceder as novas permissoes aos papeis curados
-- (super_admin/admin_sede ja recebem todas via CROSS JOIN).
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
        ('pastor_filial', ARRAY['announcements.read','announcements.write','families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['announcements.read','announcements.write','families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['announcements.read','announcements.write','families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['announcements.read','announcements.write','families.read','governance.read','members.read','ministries.read','reports.read']),
        ('contador',      ARRAY['finance.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;
