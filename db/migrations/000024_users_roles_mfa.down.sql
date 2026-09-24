-- 000024_users_roles_mfa.down.sql
DROP FUNCTION IF EXISTS auth_lookup_user(citext);

CREATE FUNCTION auth_lookup_user(p_email citext)
RETURNS TABLE (
    user_id      text,
    full_name    text,
    email        citext,
    tenant_id    text,
    branch_id    text,
    role_key     text,
    password_hash text
)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.tenant_id::text,
           COALESCE(u.branch_id::text, ''), r.key, u.password_hash
    FROM users u
    JOIN roles r ON r.id = u.role_id
    WHERE u.email = p_email
      AND u.is_active
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION auth_lookup_user(citext) TO public;

DELETE FROM role_permissions WHERE permission_id IN (
    SELECT id FROM permissions WHERE key IN ('users.read','users.write')
);
DELETE FROM permissions WHERE key IN ('users.read','users.write');

DELETE FROM role_permissions WHERE role_id IN (
    SELECT id FROM roles WHERE key IN ('lider','pastor','contador','visitante')
);
DELETE FROM roles WHERE key IN ('lider','pastor','contador','visitante');
