-- 000010_auth_functions.up.sql
-- Funções SECURITY DEFINER que operam fora do escopo RLS (executadas como owner = migrator).
-- Necessárias para a autenticação: o login precisa localizar o usuário antes de
-- resolver o contexto de tenant/filial.

CREATE OR REPLACE FUNCTION auth_lookup_user(p_email citext)
RETURNS TABLE (
    user_id      text,
    full_name    text,
    email        citext,
    tenant_id    text,
    branch_id    text,
    role_key     text,
    password_hash text
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT u.id::text, u.full_name, u.email, u.tenant_id::text,
           u.branch_id::text, r.key, u.password_hash
    FROM users u
    JOIN roles r ON r.id = u.role_id
    WHERE u.email = p_email
      AND u.is_active
    LIMIT 1;
$$;

-- Contadores base para métricas (usado fora do scopo RLS quando necessário, p.ex. Sede).
CREATE OR REPLACE FUNCTION sys_health()
RETURNS TABLE (works boolean)
LANGUAGE sql STABLE SECURITY DEFINER SET search_path = public
AS $$ SELECT true $$;

GRANT EXECUTE ON FUNCTION auth_lookup_user(citext) TO public;
