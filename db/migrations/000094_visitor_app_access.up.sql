-- 000094_visitor_app_access.up.sql
-- ACESSO DE VISITANTE AO APP POR CODIGO (WhatsApp) + TRILHA DE ACOLHIMENTO.
--
-- O visitante baixa o app, informa o telefone e recebe um codigo por WhatsApp.
-- Ao validar, ganha uma identidade (users) vinculada a um registro em
-- `visitors`, com papel `visitante` e um membership TEMPORARIO (30 dias).
-- A conversao em membro (aprovada pela secretaria) promove o MESMO membership,
-- preservando a identidade.
--
-- Nada disso da acesso de staff: o papel `visitante` nao tem permissoes e as
-- rotas de membro continuam falhando por falta de `memberships.member_id`.

-- ---------------------------------------------------------------------------
-- 1) users: identidade so-telefone (e-mail passa a ser opcional)
-- ---------------------------------------------------------------------------
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
DROP INDEX IF EXISTS uq_users_email;
CREATE UNIQUE INDEX uq_users_email ON users(email) WHERE email IS NOT NULL;

-- auth_lookup_identity / auth_identity passam a devolver e-mail vazio quando
-- NULL (identidades de visitante tem so telefone). O tipo de retorno nao muda.
CREATE OR REPLACE FUNCTION auth_lookup_identity(p_identifier text)
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
    SELECT u.id::text, u.full_name, COALESCE(u.email, ''::citext), u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password, u.is_platform_admin
    FROM users u
    WHERE u.email = lower(btrim(p_identifier))::citext
       OR (regexp_replace(p_identifier, '\D', '', 'g') <> ''
           AND u.phone = regexp_replace(p_identifier, '\D', '', 'g'))
    LIMIT 1;
$$;

CREATE OR REPLACE FUNCTION auth_identity(p_user_id uuid)
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
    SELECT u.id::text, u.full_name, COALESCE(u.email, ''::citext), u.password_hash,
           u.mfa_enabled, COALESCE(u.mfa_secret, ''), u.is_active,
           COALESCE(u.phone, ''), u.must_change_password, u.is_platform_admin
    FROM users u
    WHERE u.id = p_user_id
    LIMIT 1;
$$;

-- ---------------------------------------------------------------------------
-- 2) memberships: vinculo com visitante + expiracao do acesso temporario
-- ---------------------------------------------------------------------------
ALTER TABLE memberships
    ADD COLUMN visitor_id uuid REFERENCES visitors(id) ON DELETE SET NULL,
    ADD COLUMN expires_at timestamptz; -- NULL => sem expiracao (staff/membro)

CREATE INDEX idx_memberships_visitor ON memberships(visitor_id);

-- auth_memberships passa a devolver visitor_id/expires_at e marca como inativo
-- o vinculo temporario ja vencido. Novo formato => recria a funcao.
DROP FUNCTION IF EXISTS auth_memberships(uuid);

CREATE FUNCTION auth_memberships(p_user_id uuid)
RETURNS TABLE (
    tenant_id   text,
    tenant_name text,
    tenant_slug text,
    role_key    text,
    branch_id   text,
    is_active   boolean,
    member_id   text,
    visitor_id  text,
    expires_at  timestamptz
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id::text, t.name, t.slug, r.key, m.branch_id::text,
           (m.is_active AND (m.expires_at IS NULL OR m.expires_at > now())),
           m.member_id::text,
           m.visitor_id::text,
           m.expires_at
    FROM memberships m
    JOIN tenants t ON t.id = m.tenant_id
    JOIN roles   r ON r.id = m.role_id
    WHERE m.user_id = p_user_id
    ORDER BY m.is_active DESC, t.name;
$$;

GRANT EXECUTE ON FUNCTION auth_lookup_identity(text) TO public;
GRANT EXECUTE ON FUNCTION auth_identity(uuid)     TO public;
GRANT EXECUTE ON FUNCTION auth_memberships(uuid)  TO public, chosenerp_app;

-- ---------------------------------------------------------------------------
-- 3) Codigos de acesso (OTP) enviados por WhatsApp
-- ---------------------------------------------------------------------------
-- Sem policies: o papel da aplicacao nao le/grava direto. Todo o acesso e
-- pelas funcoes SECURITY DEFINER abaixo (dono do schema ignora o RLS).
CREATE TABLE access_codes (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    identifier   text NOT NULL,               -- telefone normalizado (digitos)
    name         text,                        -- nome informado no pedido
    channel      text NOT NULL DEFAULT 'whatsapp',
    code_hash    text NOT NULL,               -- bcrypt do codigo (6 digitos)
    expires_at   timestamptz NOT NULL,
    attempts     int NOT NULL DEFAULT 0,
    max_attempts int NOT NULL DEFAULT 5,
    consumed_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_codes_lookup ON access_codes(tenant_id, identifier, created_at DESC);

ALTER TABLE access_codes ENABLE ROW LEVEL SECURITY;

-- Emite um novo codigo (invalida os anteriores nao consumidos) com anti-abuso:
-- 1 pedido por minuto e no maximo 5 por 15 minutos, por telefone/igreja.
CREATE FUNCTION visitor_code_issue(
    p_tenant     uuid,
    p_identifier text,
    p_name       text,
    p_hash       text,
    p_ttl_seconds int DEFAULT 300
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM access_codes
        WHERE tenant_id = p_tenant AND identifier = p_identifier
          AND created_at > now() - interval '60 seconds') THEN
        RAISE EXCEPTION 'aguarde para pedir outro codigo' USING ERRCODE = 'P0001';
    END IF;
    IF (SELECT count(*) FROM access_codes
        WHERE tenant_id = p_tenant AND identifier = p_identifier
          AND created_at > now() - interval '15 minutes') >= 5 THEN
        RAISE EXCEPTION 'muitos codigos solicitados' USING ERRCODE = 'P0001';
    END IF;

    UPDATE access_codes SET consumed_at = now()
    WHERE tenant_id = p_tenant AND identifier = p_identifier AND consumed_at IS NULL;

    INSERT INTO access_codes (tenant_id, identifier, name, code_hash, expires_at)
    VALUES (p_tenant, p_identifier, NULLIF(btrim(p_name), ''), p_hash,
            now() + make_interval(secs => GREATEST(p_ttl_seconds, 60)));
END;
$$;

-- Ultimo codigo nao consumido do telefone/igreja.
CREATE FUNCTION visitor_code_fetch(p_tenant uuid, p_identifier text)
RETURNS TABLE (
    id           uuid,
    name         text,
    code_hash    text,
    expires_at   timestamptz,
    attempts     int,
    max_attempts int
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT id, name, code_hash, expires_at, attempts, max_attempts
    FROM access_codes
    WHERE tenant_id = p_tenant AND identifier = p_identifier AND consumed_at IS NULL
    ORDER BY created_at DESC
    LIMIT 1;
$$;

-- Registra a tentativa; `p_consume` marca o codigo como usado.
CREATE FUNCTION visitor_code_use(p_id uuid, p_consume boolean)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
BEGIN
    UPDATE access_codes SET
        attempts = attempts + 1,
        consumed_at = CASE WHEN p_consume THEN now() ELSE consumed_at END
    WHERE id = p_id;
END;
$$;

-- ---------------------------------------------------------------------------
-- 4) Trilha de acolhimento: historico APPEND-ONLY das mudancas de etapa
-- ---------------------------------------------------------------------------
CREATE TABLE visitor_journey (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    visitor_id  uuid NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    from_stage  text,       -- NULL no primeiro registro
    to_stage    text NOT NULL,
    notes       text,
    changed_by  uuid REFERENCES users(id) ON DELETE SET NULL, -- NULL => automatico
    occurred_at timestamptz NOT NULL DEFAULT now(),
    created_at  timestamptz NOT NULL DEFAULT now(),
    prev_hash   text,
    hash        text NOT NULL DEFAULT ''
);

CREATE INDEX idx_visitor_journey_visitor ON visitor_journey(visitor_id, created_at DESC);
CREATE INDEX idx_visitor_journey_tenant  ON visitor_journey(tenant_id, branch_id);

-- Imutabilidade (append-only), com a mesma excecao de cascata do member_history.
CREATE OR REPLACE FUNCTION visitor_journey_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'visitor_journey is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER visitor_journey_no_update
BEFORE UPDATE OR DELETE ON visitor_journey
FOR EACH ROW EXECUTE FUNCTION visitor_journey_guard();

-- Hash-chain cronologico por tenant (mesmo padrao do member_history).
CREATE OR REPLACE FUNCTION visitor_journey_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.visitor_id::text || '|' || COALESCE(NEW.from_stage,'') || '|' ||
            NEW.to_stage || '|' || COALESCE(NEW.notes,'') || '|' ||
            COALESCE(NEW.changed_by::text,'') || '|' ||
            NEW.occurred_at::text || '|' || NEW.created_at::text;
    NEW.prev_hash := (SELECT hash FROM visitor_journey
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER visitor_journey_hash
BEFORE INSERT ON visitor_journey
FOR EACH ROW EXECUTE FUNCTION visitor_journey_hash();

ALTER TABLE visitor_journey ENABLE ROW LEVEL SECURITY;

CREATE POLICY visitor_journey_sel ON visitor_journey
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY visitor_journey_ins ON visitor_journey
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));

-- Toda mudanca de journey_stage em `visitors` vira uma linha na trilha,
-- independentemente do caminho de codigo (manual, app ou conversao).
CREATE OR REPLACE FUNCTION visitors_journey_log() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND NEW.journey_stage IS NOT DISTINCT FROM OLD.journey_stage THEN
        RETURN NEW;
    END IF;
    INSERT INTO visitor_journey (tenant_id, branch_id, visitor_id, from_stage, to_stage)
    VALUES (NEW.tenant_id, NEW.branch_id, NEW.id,
            CASE WHEN TG_OP = 'UPDATE' THEN OLD.journey_stage ELSE NULL END,
            NEW.journey_stage);
    RETURN NEW;
END;
$$;

CREATE TRIGGER visitors_journey_log
AFTER INSERT OR UPDATE OF journey_stage ON visitors
FOR EACH ROW EXECUTE FUNCTION visitors_journey_log();

-- ---------------------------------------------------------------------------
-- 5) Backfill: papel `visitante` em igrejas antigas (criadas antes do 000074)
-- ---------------------------------------------------------------------------
INSERT INTO roles (tenant_id, key, name, is_system)
SELECT t.id, 'visitante', 'Visitante', true
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM roles r WHERE r.tenant_id = t.id AND r.key = 'visitante');

-- ---------------------------------------------------------------------------
-- 6) Lookup publico do tenant pelo slug (id + filial raiz + nome)
-- ---------------------------------------------------------------------------
CREATE FUNCTION visitor_tenant_lookup(p_slug text)
RETURNS TABLE (tenant_id uuid, branch_id uuid, tenant_name text)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT t.id,
           (SELECT b.id FROM branches b WHERE b.tenant_id = t.id
            ORDER BY (b.parent_id IS NULL) DESC, b.created_at ASC LIMIT 1),
           t.name
    FROM tenants t
    WHERE t.slug = lower(btrim(p_slug)) AND t.is_active
    LIMIT 1;
$$;

GRANT EXECUTE ON FUNCTION visitor_tenant_lookup(text) TO public, chosenerp_app;

-- ---------------------------------------------------------------------------
-- 7) Cria/renova o acesso do visitante (identidade + visitor + membership)
-- ---------------------------------------------------------------------------
-- Idempotente por telefone/igreja. Recusa (23505) quando o telefone ja pertence
-- a uma identidade com acesso de membro/staff NESTA igreja (nao faz downgrade).
CREATE FUNCTION user_attach_visitor(
    p_phone     text,
    p_full_name text,
    p_tenant_id uuid
)
RETURNS TABLE (user_id uuid, visitor_id uuid, branch_id uuid)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_phone    text := NULLIF(regexp_replace(COALESCE(p_phone, ''), '\D', '', 'g'), '');
    v_user     uuid;
    v_visitor  uuid;
    v_branch   uuid;
    v_role     uuid;
    v_existing uuid;
    v_exist_role text;
    v_name     text := COALESCE(NULLIF(btrim(p_full_name), ''), 'Visitante');
BEGIN
    IF v_phone IS NULL THEN
        RAISE EXCEPTION 'telefone obrigatorio' USING ERRCODE = '22023';
    END IF;

    SELECT id INTO v_role FROM roles WHERE tenant_id = p_tenant_id AND key = 'visitante';
    IF v_role IS NULL THEN
        RAISE EXCEPTION 'perfil visitante nao encontrado' USING ERRCODE = 'P0002';
    END IF;

    SELECT id INTO v_branch FROM branches
    WHERE tenant_id = p_tenant_id
    ORDER BY (parent_id IS NULL) DESC, created_at ASC
    LIMIT 1;
    IF v_branch IS NULL THEN
        RAISE EXCEPTION 'igreja sem filial' USING ERRCODE = 'P0002';
    END IF;

    SELECT id INTO v_user FROM users WHERE phone = v_phone;
    IF v_user IS NULL THEN
        INSERT INTO users (phone, full_name, is_active)
        VALUES (v_phone, v_name, true)
        RETURNING id INTO v_user;
    END IF;

    -- Se a identidade ja tem vinculo NESTA igreja, so pode ser de visitante.
    SELECT m.id, r.key INTO v_existing, v_exist_role
    FROM memberships m JOIN roles r ON r.id = m.role_id
    WHERE m.user_id = v_user AND m.tenant_id = p_tenant_id;
    IF v_existing IS NOT NULL AND v_exist_role <> 'visitante' THEN
        RAISE EXCEPTION 'identidade ja possui acesso nesta igreja' USING ERRCODE = '23505';
    END IF;

    -- Reaproveita o visitante pelo telefone (na propria filial) ou cria.
    SELECT id INTO v_visitor FROM visitors
    WHERE tenant_id = p_tenant_id AND phone = v_phone
    ORDER BY created_at ASC
    LIMIT 1;
    IF v_visitor IS NULL THEN
        INSERT INTO visitors (tenant_id, branch_id, first_name, last_name, phone, source)
        VALUES (p_tenant_id, v_branch, v_name, '', v_phone, 'app')
        RETURNING id INTO v_visitor;
    END IF;

    INSERT INTO memberships (user_id, tenant_id, role_id, branch_id, is_active, visitor_id, expires_at)
    VALUES (v_user, p_tenant_id, v_role, v_branch, true, v_visitor, now() + interval '30 days')
    ON CONFLICT (user_id, tenant_id) DO UPDATE SET
        visitor_id = EXCLUDED.visitor_id,
        role_id    = EXCLUDED.role_id,
        branch_id  = EXCLUDED.branch_id,
        expires_at = EXCLUDED.expires_at,
        is_active  = true,
        updated_at = now();

    RETURN QUERY SELECT v_user, v_visitor, v_branch;
END;
$$;

GRANT EXECUTE ON FUNCTION user_attach_visitor(text, text, uuid) TO chosenerp_app;

-- ---------------------------------------------------------------------------
-- 8) Promove o acesso do visitante a membro (aprovacao da secretaria)
-- ---------------------------------------------------------------------------
CREATE FUNCTION promote_visitor_to_member(
    p_user_id   uuid,
    p_tenant_id uuid,
    p_member_id uuid
)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_role uuid;
BEGIN
    SELECT id INTO v_role FROM roles WHERE tenant_id = p_tenant_id AND key = 'membro';
    IF v_role IS NULL THEN
        RAISE EXCEPTION 'perfil membro nao encontrado' USING ERRCODE = 'P0002';
    END IF;

    UPDATE memberships SET
        role_id    = v_role,
        member_id  = p_member_id,
        visitor_id = NULL,
        expires_at = NULL,
        is_active  = true,
        updated_at = now()
    WHERE user_id = p_user_id AND tenant_id = p_tenant_id;
END;
$$;

GRANT EXECUTE ON FUNCTION promote_visitor_to_member(uuid, uuid, uuid) TO chosenerp_app;

-- ---------------------------------------------------------------------------
-- 9) Sessao de visitante viva? (checagem leve no middleware)
-- ---------------------------------------------------------------------------
CREATE FUNCTION visitor_access_active(p_user_id uuid, p_tenant_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT EXISTS (
        SELECT 1 FROM memberships m JOIN roles r ON r.id = m.role_id
        WHERE m.user_id = p_user_id AND m.tenant_id = p_tenant_id
          AND r.key = 'visitante' AND m.is_active
          AND (m.expires_at IS NULL OR m.expires_at > now()));
$$;

GRANT EXECUTE ON FUNCTION visitor_access_active(uuid, uuid) TO public, chosenerp_app;
