-- 000094_visitor_app_access.down.sql
-- Reverte o acesso de visitante por codigo e a trilha de acolhimento.

DROP FUNCTION IF EXISTS visitor_access_active(uuid, uuid);
DROP FUNCTION IF EXISTS promote_visitor_to_member(uuid, uuid, uuid);
DROP FUNCTION IF EXISTS user_attach_visitor(text, text, uuid);
DROP FUNCTION IF EXISTS visitor_tenant_lookup(text);

DROP TRIGGER IF EXISTS visitors_journey_log ON visitors;
DROP FUNCTION IF EXISTS visitors_journey_log();
DROP TABLE IF EXISTS visitor_journey;
DROP FUNCTION IF EXISTS visitor_journey_hash();
DROP FUNCTION IF EXISTS visitor_journey_guard();

DROP FUNCTION IF EXISTS visitor_code_use(uuid, boolean);
DROP FUNCTION IF EXISTS visitor_code_fetch(uuid, text);
DROP FUNCTION IF EXISTS visitor_code_issue(uuid, text, text, text, int);
DROP TABLE IF EXISTS access_codes;

-- Remove os vinculos de visitante antes de soltar as colunas.
DELETE FROM memberships WHERE visitor_id IS NOT NULL;
ALTER TABLE memberships DROP COLUMN IF EXISTS expires_at;
ALTER TABLE memberships DROP COLUMN IF EXISTS visitor_id;

-- auth_memberships volta ao formato anterior.
DROP FUNCTION IF EXISTS auth_memberships(uuid);

CREATE FUNCTION auth_memberships(p_user_id uuid)
RETURNS TABLE (
    tenant_id   text,
    tenant_name text,
    tenant_slug text,
    role_key    text,
    branch_id   text,
    is_active   boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id::text, t.name, t.slug, r.key, m.branch_id::text, m.is_active
    FROM memberships m
    JOIN tenants t ON t.id = m.tenant_id
    JOIN roles   r ON r.id = m.role_id
    WHERE m.user_id = p_user_id
    ORDER BY m.is_active DESC, t.name;
$$;

GRANT EXECUTE ON FUNCTION auth_memberships(uuid) TO public;

-- users.email volta a ser obrigatorio (apenas onde houver valor).
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_not_null_guard;
DROP INDEX IF EXISTS uq_users_email;
-- Sem e-mail obrigatorio de novo: remove identidades so-telefone para poder
-- restaurar NOT NULL sem violar a constraint.
DELETE FROM users WHERE email IS NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
