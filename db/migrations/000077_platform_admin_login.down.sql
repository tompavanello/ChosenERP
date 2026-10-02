-- 000077_platform_admin_login.down.sql
-- Reverte as funcoes de identidade ao formato sem is_platform_admin. Os
-- vinculos removidos no up NAO sao restaurados.

DROP FUNCTION IF EXISTS auth_lookup_identity(text);
CREATE FUNCTION auth_lookup_identity(p_identifier text)
RETURNS TABLE (
    user_id              text,
    full_name            text,
    email                citext,
    password_hash        text,
    mfa_enabled          boolean,
    mfa_secret           text,
    is_active            boolean,
    phone                text,
    must_change_password boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password
    FROM users u
    WHERE u.email = lower(btrim(p_identifier))::citext
       OR (regexp_replace(p_identifier, '\D', '', 'g') <> ''
           AND u.phone = regexp_replace(p_identifier, '\D', '', 'g'))
    LIMIT 1;
$$;
GRANT EXECUTE ON FUNCTION auth_lookup_identity(text) TO public;

DROP FUNCTION IF EXISTS auth_identity(uuid);
CREATE FUNCTION auth_identity(p_user_id uuid)
RETURNS TABLE (
    user_id              text,
    full_name            text,
    email                citext,
    password_hash        text,
    mfa_enabled          boolean,
    mfa_secret           text,
    is_active            boolean,
    phone                text,
    must_change_password boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password
    FROM users u
    WHERE u.id = p_user_id
    LIMIT 1;
$$;
GRANT EXECUTE ON FUNCTION auth_identity(uuid) TO public;
