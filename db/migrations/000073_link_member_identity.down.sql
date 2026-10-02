-- 000073_link_member_identity.down.sql
-- Reverte o vinculo identidade <-> membro: restaura auth_memberships sem
-- member_id e remove a coluna/indice.

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

DROP INDEX IF EXISTS uq_memberships_tenant_member;
ALTER TABLE memberships DROP COLUMN IF EXISTS member_id;
