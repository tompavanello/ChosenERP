-- 000077_platform_admin_login.up.sql
-- ADMIN DE PLATAFORMA SEM IGREJA.
--
-- O operador do SaaS nao deve ver dados operacionais das igrejas: ele existe
-- para administrar o ambiente e ver estatisticas gerais. Para isso:
--   1) as funcoes de identidade passam a devolver `is_platform_admin`, para o
--      login emitir tokens SEM tenant quando a identidade nao tem vinculo;
--   2) removemos o vinculo de igreja de quem e platform admin (a flag de
--      plataforma substitui o papel de igreja).
--
-- O RLS continua sendo a rede de seguranca: com app.tenant_id vazio e papel
-- `platform_admin`, nenhuma tabela operacional devolve linhas.

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
    must_change_password boolean,
    is_platform_admin    boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password, u.is_platform_admin
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
    must_change_password boolean,
    is_platform_admin    boolean
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password, u.is_platform_admin
    FROM users u
    WHERE u.id = p_user_id
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION auth_identity(uuid) TO public;

-- Separa de verdade: platform admin nao mantem vinculo operacional de igreja.
-- (O console usa SECURITY DEFINER/WithSystem, nao depende de membership.)
DELETE FROM memberships m
USING users u
WHERE m.user_id = u.id AND u.is_platform_admin;
