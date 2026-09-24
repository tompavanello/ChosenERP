-- 000053_identity_memberships.up.sql
-- Identidade global + vinculo N:N com a igreja (membership).
--
-- Antes: `users` guardava a propria identidade E o contexto de acesso
-- (tenant_id/branch_id/role_id), com e-mail UNIQUE global. Isso impedia que a
-- mesma pessoa acessasse mais de uma igreja.
--
-- Agora: `users` e a identidade (global). `memberships` liga pessoa  igreja,
-- guardando papel e filial. O JWT continua carregando o tenant ATIVO (tid/bid/role)
-- resolvido a partir do membership - o RLS nao muda de eixo.
--
-- Rollout: a tabela e criada e populada (backfill) ANTES das colunas antigas
-- serem removidas, tudo na mesma transacao (o runner migra em transacao).

-- ---------------------------------------------------------------------------
-- 1) Vinculo pessoa  igreja
-- ---------------------------------------------------------------------------
CREATE TABLE memberships (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role_id    uuid NOT NULL REFERENCES roles(id),
    branch_id  uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => Sede
    is_active  boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, tenant_id)
);
CREATE INDEX idx_memberships_user   ON memberships(user_id);
CREATE INDEX idx_memberships_tenant ON memberships(tenant_id, branch_id);

-- ---------------------------------------------------------------------------
-- 2) Backfill: cada usuario atual vira 1 membership (idempotente)
-- ---------------------------------------------------------------------------
INSERT INTO memberships (user_id, tenant_id, role_id, branch_id, is_active, created_at)
SELECT id, tenant_id, role_id, branch_id, is_active, created_at
FROM users
ON CONFLICT (user_id, tenant_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 3) Helpers de identidade
-- ---------------------------------------------------------------------------
-- current_user_id() le o GUC app.user_id (sessao RLS). E o que permite ao RLS
-- reconhecer a propria identidade mesmo sem tenant (ex.: seletor de igreja).
CREATE OR REPLACE FUNCTION current_user_id() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.user_id', true), '')::uuid
$$;

-- ---------------------------------------------------------------------------
-- 4) users passa a ser global: remove o contexto de acesso
--    (as politicas atuais dependem de tenant_id => precisam sair antes)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS users_sel ON users;
DROP POLICY IF EXISTS users_all ON users;

ALTER TABLE users
    DROP COLUMN tenant_id,
    DROP COLUMN branch_id,
    DROP COLUMN role_id;
-- email UNIQUE global permanece.

-- ---------------------------------------------------------------------------
-- 5) RLS das identidades e dos vinculos
-- ---------------------------------------------------------------------------
CREATE POLICY users_sel ON users FOR SELECT USING (
    is_system()
    OR id = current_user_id()
    OR EXISTS (
        SELECT 1 FROM memberships me
        WHERE me.user_id = users.id AND me.tenant_id = current_tenant()
    )
);

-- INSERT so via funcao SECURITY DEFINER user_attach_to_tenant().
CREATE POLICY users_ins ON users FOR INSERT WITH CHECK (false);

CREATE POLICY users_upd ON users FOR UPDATE USING (
    is_system()
    OR id = current_user_id()
    OR EXISTS (
        SELECT 1 FROM memberships me
        WHERE me.user_id = users.id AND me.tenant_id = current_tenant()
    )
) WITH CHECK (
    is_system()
    OR id = current_user_id()
    OR EXISTS (
        SELECT 1 FROM memberships me
        WHERE me.user_id = users.id AND me.tenant_id = current_tenant()
    )
);

ALTER TABLE memberships ENABLE ROW LEVEL SECURITY;

-- Self enxerga todos os proprios vinculos (necessario para o seletor de igreja);
-- os demais seguem o escopo do tenant.
CREATE POLICY memberships_sel ON memberships FOR SELECT USING (
    is_system()
    OR user_id = current_user_id()
    OR rls_read(tenant_id, branch_id, false)
);
CREATE POLICY memberships_ins ON memberships FOR INSERT
    WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY memberships_upd ON memberships FOR UPDATE
    USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY memberships_del ON memberships FOR DELETE
    USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- 6) Funcoes de autenticacao (SECURITY DEFINER => fora do RLS)
-- ---------------------------------------------------------------------------
-- auth_lookup_user passa a devolver a IDENTIDADE (sem tenant/role).
DROP FUNCTION IF EXISTS auth_lookup_user(citext);

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

-- auth_identity carrega a identidade por id (usado no select/switch de igreja).
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

-- auth_memberships lista as igrejas de uma identidade (ativas primeiro).
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

-- user_attach_to_tenant: cria a identidade (se o e-mail e novo) e anexa o
-- membership. Se a identidade ja existe, NAO altera a senha existente.
CREATE FUNCTION user_attach_to_tenant(
    p_email         citext,
    p_password_hash text,
    p_full_name     text,
    p_tenant_id     uuid,
    p_role_key      text,
    p_branch_id     uuid,
    p_is_active     boolean
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id uuid;
    v_role_id uuid;
BEGIN
    SELECT id INTO v_role_id
    FROM roles
    WHERE tenant_id = p_tenant_id AND key = p_role_key;

    IF v_role_id IS NULL THEN
        RAISE EXCEPTION 'perfil nao encontrado' USING ERRCODE = 'P0002';
    END IF;

    SELECT id INTO v_user_id FROM users WHERE email = p_email;

    IF v_user_id IS NULL THEN
        IF p_password_hash IS NULL OR p_password_hash = '' THEN
            RAISE EXCEPTION 'senha obrigatoria para nova identidade' USING ERRCODE = '22023';
        END IF;
        INSERT INTO users (email, password_hash, full_name, is_active)
        VALUES (p_email, p_password_hash, p_full_name, COALESCE(p_is_active, true))
        RETURNING id INTO v_user_id;
    ELSIF EXISTS (SELECT 1 FROM memberships
                  WHERE user_id = v_user_id AND tenant_id = p_tenant_id) THEN
        RAISE EXCEPTION 'identidade ja vinculada a esta igreja' USING ERRCODE = '23505';
    END IF;

    INSERT INTO memberships (user_id, tenant_id, role_id, branch_id, is_active)
    VALUES (v_user_id, p_tenant_id, v_role_id, p_branch_id, COALESCE(p_is_active, true));

    RETURN v_user_id;
END;
$$;

GRANT EXECUTE ON FUNCTION auth_lookup_user(citext) TO public;
GRANT EXECUTE ON FUNCTION auth_identity(uuid)     TO public;
GRANT EXECUTE ON FUNCTION auth_memberships(uuid)  TO public;
GRANT EXECUTE ON FUNCTION user_attach_to_tenant(citext, text, text, uuid, text, uuid, boolean) TO public;
