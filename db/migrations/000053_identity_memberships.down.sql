-- 000053_identity_memberships.down.sql
-- Reverte para o modelo antigo (contexto de acesso dentro de users).
-- Best-effort: se uma identidade tiver mais de um membership, apenas um deles
-- (o primeiro) é restaurado em users.

DROP POLICY IF EXISTS users_sel ON users;
DROP POLICY IF EXISTS users_ins ON users;
DROP POLICY IF EXISTS users_upd ON users;

ALTER TABLE users
    ADD COLUMN tenant_id uuid REFERENCES tenants(id) ON DELETE CASCADE,
    ADD COLUMN branch_id uuid REFERENCES branches(id) ON DELETE SET NULL,
    ADD COLUMN role_id   uuid REFERENCES roles(id);

UPDATE users u SET
    tenant_id = m.tenant_id,
    branch_id = m.branch_id,
    role_id   = m.role_id
FROM memberships m
WHERE m.user_id = u.id;

-- Só restaura o NOT NULL quando todas as identidades tiverem vínculo.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE tenant_id IS NULL OR role_id IS NULL) THEN
        ALTER TABLE users ALTER COLUMN tenant_id SET NOT NULL;
        ALTER TABLE users ALTER COLUMN role_id   SET NOT NULL;
    END IF;
END $$;

DROP POLICY IF EXISTS memberships_sel ON memberships;
DROP POLICY IF EXISTS memberships_ins ON memberships;
DROP POLICY IF EXISTS memberships_upd ON memberships;
DROP POLICY IF EXISTS memberships_del ON memberships;
DROP TABLE IF EXISTS memberships;

DROP POLICY IF EXISTS users_sel ON users;
CREATE POLICY users_sel ON users USING (tenant_id = current_tenant() OR is_system());
DROP POLICY IF EXISTS users_all ON users;
CREATE POLICY users_all ON users
  USING (tenant_id = current_tenant() OR is_system())
  WITH CHECK (tenant_id = current_tenant() OR is_system());

DROP FUNCTION IF EXISTS user_attach_to_tenant(citext, text, text, uuid, text, uuid, boolean);
DROP FUNCTION IF EXISTS auth_memberships(uuid);
DROP FUNCTION IF EXISTS auth_identity(uuid);
DROP FUNCTION IF EXISTS current_user_id();

-- Restaura auth_lookup_user ao formato com tenant/branch/role (MFA).
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
