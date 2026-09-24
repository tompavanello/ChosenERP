-- 000024_users_roles_mfa.up.sql
-- Etapa 3 — Usuários e Acessos:
--   1) permissões de gestão de usuários (users.read/users.write);
--   2) perfis que faltavam no seed (lider, pastor, contador, visitante) e
--      permissões-base para tesoureiro/secretario/admin_sede;
--   3) auth_lookup_user passa a devolver mfa_enabled/mfa_secret (login com MFA).

-- ---------------------------------------------------------------------------
-- 1) Permissões de gestão de usuários
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('users.read',  'admin', 'Ler usuários'),
    ('users.write', 'admin', 'Gerir usuários')
ON CONFLICT (key) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2) Perfis faltantes (PRD §3), por tenant
-- ---------------------------------------------------------------------------
INSERT INTO roles (tenant_id, key, name, is_system)
SELECT t.id, v.key, v.name, true
FROM tenants t
CROSS JOIN (VALUES
    ('lider',     'Líder de Ministério/Célula'),
    ('pastor',    'Pastor/Conselheiro'),
    ('contador',  'Contador Externo'),
    ('visitante', 'Visitante')
) AS v(key, name)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 3) Permissões-base dos perfis
-- ---------------------------------------------------------------------------
-- admin_sede: acesso total do tenant (mesmo conjunto do super_admin).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key = 'admin_sede'
ON CONFLICT DO NOTHING;

-- super_admin: garante as permissões novas (users.read/write).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key = 'super_admin'
ON CONFLICT DO NOTHING;

-- Demais perfis: conjunto curado por função.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN (VALUES
    ('secretario',  ARRAY['members.read','members.write','families.read','visitors.read','reports.read']),
    ('tesoureiro',  ARRAY['members.read','finance.read','finance.write','reports.read']),
    ('pastor_filial', ARRAY['members.read','members.write','families.read','visitors.read','finance.read','ministries.read','ministries.write','reports.read','governance.read']),
    ('lider',       ARRAY['members.read','families.read','ministries.read','ministries.write','reports.read']),
    ('pastor',      ARRAY['members.read','families.read','ministries.read','governance.read','reports.read']),
    ('contador',    ARRAY['finance.read','reports.read'])
) AS v(role_key, perms)
JOIN permissions p ON p.key = ANY(v.perms)
WHERE r.key = v.role_key
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4) auth_lookup_user devolve MFA (o tipo de retorno muda => recria a função)
-- ---------------------------------------------------------------------------
DROP FUNCTION IF EXISTS auth_lookup_user(citext);

CREATE FUNCTION auth_lookup_user(p_email citext)
RETURNS TABLE (
    user_id       text,
    full_name     text,
    email         citext,
    tenant_id     text,
    branch_id     text,
    role_key      text,
    password_hash text,
    mfa_enabled   boolean,
    mfa_secret    text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.tenant_id::text,
           u.branch_id::text, r.key, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, '')
    FROM users u
    JOIN roles r ON r.id = u.role_id
    WHERE u.email = p_email
      AND u.is_active
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION auth_lookup_user(citext) TO public;
