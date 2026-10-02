-- 000076_member_access.up.sql
-- ACESSO DO MEMBRO AO APP.
--
-- O membro precisa de uma identidade (users) com senha, vinculada ao seu
-- registro em `members`. O login passa a aceitar E-MAIL OU TELEFONE.
--
-- E-mail continua obrigatorio (users.email e UNIQUE global); o telefone e um
-- atalho de login (UNIQUE global, so digitos). Quando o membro divide o numero
-- com a familia, o login por telefone so vale para o dono - os demais usam
-- e-mail. A senha inicial e definida pela Sede e marcada como provisoria
-- (`must_change_password`), forcando a troca no primeiro acesso do app.

ALTER TABLE users ADD COLUMN phone text;
ALTER TABLE users ADD COLUMN must_change_password boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD CONSTRAINT users_phone_digits_check
    CHECK (phone IS NULL OR phone ~ '^[0-9]{8,15}$');
CREATE UNIQUE INDEX uq_users_phone ON users(phone) WHERE phone IS NOT NULL;

-- ---------------------------------------------------------------------------
-- Login por e-mail OU telefone. Substitui auth_lookup_user(citext).
-- ---------------------------------------------------------------------------
DROP FUNCTION IF EXISTS auth_lookup_user(citext);

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

-- auth_identity tambem devolve telefone + senha provisoria.
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

-- ---------------------------------------------------------------------------
-- Cria/anexa o acesso do membro: identidade + membership (papel membro) com
-- `member_id`, na filial do proprio membro (para o RLS enxergar o cadastro).
-- ---------------------------------------------------------------------------
CREATE FUNCTION user_attach_member(
    p_email         citext,
    p_phone         text,
    p_password_hash text,
    p_full_name     text,
    p_tenant_id     uuid,
    p_member_id     uuid
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_user_id  uuid;
    v_role_id  uuid;
    v_branch   uuid;
    v_phone    text := NULLIF(regexp_replace(COALESCE(p_phone, ''), '\D', '', 'g'), '');
BEGIN
    SELECT branch_id INTO v_branch
    FROM members WHERE id = p_member_id AND tenant_id = p_tenant_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'membro nao encontrado' USING ERRCODE = 'P0002';
    END IF;

    SELECT id INTO v_role_id
    FROM roles WHERE tenant_id = p_tenant_id AND key = 'membro';
    IF v_role_id IS NULL THEN
        RAISE EXCEPTION 'perfil membro nao encontrado' USING ERRCODE = 'P0002';
    END IF;

    SELECT id INTO v_user_id FROM users WHERE email = p_email;

    IF v_user_id IS NULL THEN
        IF p_password_hash IS NULL OR p_password_hash = '' THEN
            RAISE EXCEPTION 'senha obrigatoria para novo acesso' USING ERRCODE = '22023';
        END IF;
        INSERT INTO users (email, phone, password_hash, full_name, is_active, must_change_password)
        VALUES (p_email, v_phone, p_password_hash, p_full_name, true, true)
        RETURNING id INTO v_user_id;
    ELSE
        UPDATE users SET
            phone = COALESCE(v_phone, phone),
            password_hash = COALESCE(NULLIF(p_password_hash, ''), password_hash),
            must_change_password = CASE
                WHEN p_password_hash IS NOT NULL AND p_password_hash <> '' THEN true
                ELSE must_change_password END,
            updated_at = now()
        WHERE id = v_user_id;
    END IF;

    INSERT INTO memberships (user_id, tenant_id, role_id, branch_id, is_active, member_id)
    VALUES (v_user_id, p_tenant_id, v_role_id, v_branch, true, p_member_id)
    ON CONFLICT (user_id, tenant_id) DO UPDATE SET
        member_id = EXCLUDED.member_id,
        role_id = EXCLUDED.role_id,
        branch_id = EXCLUDED.branch_id,
        is_active = true,
        updated_at = now();

    RETURN v_user_id;
END;
$$;

GRANT EXECUTE ON FUNCTION user_attach_member(citext, text, text, text, uuid, uuid) TO chosenerp_app;
