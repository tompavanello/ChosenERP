-- 000076_member_access.down.sql
-- Reverte o acesso do membro: restaura auth_lookup_user/auth_identity, remove
-- user_attach_member e as colunas de telefone/senha provisoria.

DROP FUNCTION IF EXISTS user_attach_member(citext, text, text, text, uuid, uuid);

DROP FUNCTION IF EXISTS auth_identity(uuid);
CREATE FUNCTION auth_identity(p_user_id uuid)
RETURNS TABLE (
    user_id       text,
    full_name     text,
    email         citext,
    password_hash text,
    mfa_enabled   boolean,
    mfa_secret    text,
    is_active     boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active
    FROM users u
    WHERE u.id = p_user_id
    LIMIT 1;
$$;
GRANT EXECUTE ON FUNCTION auth_identity(uuid) TO public;

DROP FUNCTION IF EXISTS auth_lookup_identity(text);
CREATE FUNCTION auth_lookup_user(p_email citext)
RETURNS TABLE (
    user_id       text,
    full_name     text,
    email         citext,
    password_hash text,
    mfa_enabled   boolean,
    mfa_secret    text,
    is_active     boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active
    FROM users u
    WHERE u.email = p_email
    LIMIT 1;
$$;
GRANT EXECUTE ON FUNCTION auth_lookup_user(citext) TO public;

DROP INDEX IF EXISTS uq_users_phone;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_digits_check;
ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
