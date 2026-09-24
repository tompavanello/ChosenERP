-- 000059_tenant_onboarding.up.sql
-- Onboarding de igreja (tenant): cria a igreja com a base minima para operar --
-- tenant + Matriz (branch) + papeis padrao + permissoes -- de forma atomica.
-- O primeiro admin e anexado depois por user_attach_to_tenant (000053).
--
-- SECURITY DEFINER: tenants/roles tem RLS por tenant, e a criacao de um tenant
-- novo acontece fora de qualquer contexto de tenant, entao roda como dono.

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

    -- Filial raiz (Matriz) -- sem ela, lancamentos que exigem branch_id falham.
    INSERT INTO branches (tenant_id, name, slug, kind)
    VALUES (v_tenant, 'Sede Matriz', 'matriz', 'matriz');

    -- Papeis padrao da igreja.
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

    -- super_admin e admin_sede: todas as permissoes.
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    -- Demais papeis: conjunto curado por funcao (mesma matriz do seed).
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM (VALUES
        ('pastor_filial', ARRAY['families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['families.read','governance.read','members.read','ministries.read','reports.read']),
        ('contador',      ARRAY['finance.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;

-- Lista todas as igrejas (para a tela de onboarding do super_admin).
CREATE OR REPLACE FUNCTION list_tenants()
RETURNS TABLE (
    id           text,
    name         text,
    slug         text,
    plan         text,
    is_active    boolean,
    created_at   timestamptz,
    branch_count int,
    member_count int
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id::text, t.name, t.slug, t.plan, t.is_active, t.created_at,
           (SELECT count(*) FROM branches b WHERE b.tenant_id = t.id)::int,
           (SELECT count(*) FROM members m WHERE m.tenant_id = t.id)::int
    FROM tenants t
    ORDER BY t.created_at;
$$;

GRANT EXECUTE ON FUNCTION create_tenant(text, text, text) TO chosenerp_app;
GRANT EXECUTE ON FUNCTION list_tenants() TO chosenerp_app;
