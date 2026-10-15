-- 000097_tenant_backup.up.sql
-- Backup e restauracao de DADOS COMPLETOS por igreja (tenant).
--
-- Export: `export_tenant(uuid)` monta um payload jsonb com TODAS as linhas
-- das tabelas de escopo do tenant (coluna tenant_id) + as tabelas-filhas sem
-- tenant_id (que herdam o escopo do pai via FK) + as identidades (users)
-- referenciadas por memberships + os role_permissions dos papeis do tenant.
--
-- Restore: `restore_tenant(uuid, jsonb)` SUBSTITUI todo o conteudo daquele
-- tenant (as demais igrejas ficam intactas). Como e uma operacao destrutiva e
-- as tabelas append-only (audit_log, financial_transactions, member_history,
-- visitor_journey, ...) bloqueiam DELETE/UPDATE por trigger, a funcao roda
-- como dono (SECURITY DEFINER) e desliga os triggers de usuario durante a
-- transacao (`session_replication_role = replica`). As tabelas-filhas e as
-- FKs sao resolvidas dinamicamente pelo catalogo (pg_constraint), entao novas
-- tabelas passam a ser cobertas automaticamente.
--
-- As tabelas de metadados do backup (tenant_backups/backup_settings) NAO usam
-- a coluna `tenant_id` (usam target_tenant_id) justamente para nao entrarem no
-- enunciado dinamico e nao se auto-restaurarem.

-- ---------------------------------------------------------------------------
-- 1) Catalogo de backups por igreja (metadados)
-- ---------------------------------------------------------------------------
CREATE TABLE tenant_backups (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    filename         text NOT NULL,
    size_bytes       bigint NOT NULL DEFAULT 0,
    kind             text NOT NULL DEFAULT 'manual', -- manual | scheduled | pre_restore
    status           text NOT NULL DEFAULT 'ready',  -- creating | ready | failed
    error            text,
    counts           jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by       uuid,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenant_backups_kind_check   CHECK (kind IN ('manual', 'scheduled', 'pre_restore')),
    CONSTRAINT tenant_backups_status_check CHECK (status IN ('creating', 'ready', 'failed'))
);
CREATE INDEX idx_tenant_backups_tenant ON tenant_backups(target_tenant_id, created_at DESC);
CREATE INDEX idx_tenant_backups_created ON tenant_backups(created_at);

ALTER TABLE tenant_backups ENABLE ROW LEVEL SECURITY;
-- Apenas o contexto system/Sede (workers e console da plataforma) acessa.
DROP POLICY IF EXISTS tenant_backups_sys ON tenant_backups;
CREATE POLICY tenant_backups_sys ON tenant_backups
    FOR ALL USING (is_system()) WITH CHECK (is_system());

-- ---------------------------------------------------------------------------
-- 2) Configuracao global da rotina (linha unica)
-- ---------------------------------------------------------------------------
CREATE TABLE backup_settings (
    id              boolean PRIMARY KEY DEFAULT true CHECK (id),
    enabled         boolean NOT NULL DEFAULT false,
    interval_hours  integer NOT NULL DEFAULT 24,
    retention_days  integer NOT NULL DEFAULT 30,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

INSERT INTO backup_settings (id) VALUES (true) ON CONFLICT (id) DO NOTHING;

ALTER TABLE backup_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS backup_settings_sys ON backup_settings;
CREATE POLICY backup_settings_sys ON backup_settings
    FOR ALL USING (is_system()) WITH CHECK (is_system());

-- ---------------------------------------------------------------------------
-- 3) Helpers internos (rodam sob o dono via as funcoes SECURITY DEFINER)
-- ---------------------------------------------------------------------------

-- Lista as tabelas com coluna tenant_id (escopo direto do tenant).
CREATE OR REPLACE FUNCTION _backup_tenant_tables()
RETURNS text[]
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(array_agg(table_name), ARRAY[]::text[])
    FROM information_schema.columns
    WHERE table_schema = 'public' AND column_name = 'tenant_id';
$$;

-- Lista as tabelas-filhas (sem tenant_id proprio) que referenciam alguma
-- tabela de escopo do tenant. Resolvido pelo catalogo.
CREATE OR REPLACE FUNCTION _backup_child_tables(p_tables text[])
RETURNS text[]
LANGUAGE sql STABLE AS $$
    SELECT COALESCE(array_agg(DISTINCT c.relname), ARRAY[]::text[])
    FROM pg_constraint con
    JOIN pg_class c     ON c.oid = con.conrelid
    JOIN pg_class p     ON p.oid = con.confrelid
    JOIN pg_namespace cn ON cn.oid = c.relnamespace
    JOIN pg_namespace pn ON pn.oid = p.relnamespace
    WHERE con.contype = 'f'
      AND cn.nspname = 'public' AND pn.nspname = 'public'
      AND p.relname = ANY(p_tables)
      AND NOT (c.relname = ANY(p_tables));
$$;

-- Monta o predicado (OR de EXISTS) que seleciona as linhas de uma filha que
-- pertencem ao tenant, seguindo cada FK para uma tabela de escopo.
CREATE OR REPLACE FUNCTION _backup_child_predicate(p_child text, p_tables text[])
RETURNS text
LANGUAGE sql STABLE AS $$
    SELECT string_agg(
        format('EXISTS (SELECT 1 FROM public.%I p WHERE (%s) = (%s) AND p.tenant_id = $1)',
            p.relname,
            (SELECT string_agg(format('c.%I', ca.attname), ', ' ORDER BY u.ord)
               FROM unnest(con.conkey) WITH ORDINALITY AS u(attnum, ord)
               JOIN pg_attribute ca ON ca.attrelid = con.conrelid AND ca.attnum = u.attnum),
            (SELECT string_agg(format('p.%I', pa.attname), ', ' ORDER BY u.ord)
               FROM unnest(con.confkey) WITH ORDINALITY AS u(attnum, ord)
               JOIN pg_attribute pa ON pa.attrelid = con.confrelid AND pa.attnum = u.attnum)
        ), ' OR ')
    FROM pg_constraint con
    JOIN pg_class c     ON c.oid = con.conrelid
    JOIN pg_class p     ON p.oid = con.confrelid
    JOIN pg_namespace cn ON cn.oid = c.relnamespace
    JOIN pg_namespace pn ON pn.oid = p.relnamespace
    WHERE con.contype = 'f'
      AND cn.nspname = 'public' AND pn.nspname = 'public'
      AND c.relname = p_child
      AND p.relname = ANY(p_tables)
      AND NOT (c.relname = ANY(p_tables));
$$;

-- Insere um conjunto de linhas (jsonb array de objetos) numa tabela qualquer,
-- mapeando as colunas pelo catalogo e convertendo via jsonb_to_recordset.
-- Assim o restore nao precisa conhecer o schema de cada tabela.
CREATE OR REPLACE FUNCTION _backup_insert_rows(p_table text, p_rows jsonb)
RETURNS integer
LANGUAGE plpgsql AS $$
DECLARE
    v_coldef text;
    v_collist text;
    v_n integer := 0;
BEGIN
    IF p_rows IS NULL OR jsonb_typeof(p_rows) <> 'array' OR jsonb_array_length(p_rows) = 0 THEN
        RETURN 0;
    END IF;

    SELECT
        string_agg(format('%I %s', a.attname, format_type(a.atttypid, a.atttypmod)), ', ' ORDER BY a.attnum),
        string_agg(format('%I', a.attname), ', ' ORDER BY a.attnum)
    INTO v_coldef, v_collist
    FROM pg_attribute a
    JOIN pg_class c     ON c.oid = a.attrelid
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname = 'public' AND c.relname = p_table
      AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = '';

    IF v_collist IS NULL THEN
        RETURN 0;
    END IF;

    EXECUTE format('INSERT INTO public.%I (%s) SELECT %s FROM jsonb_to_recordset($1) AS x(%s)',
                   p_table, v_collist, v_collist, v_coldef)
        USING p_rows;
    GET DIAGNOSTICS v_n = ROW_COUNT;
    RETURN v_n;
END;
$$;

-- ---------------------------------------------------------------------------
-- 4) Export: monta o payload jsonb completo do tenant
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION export_tenant(p_tenant uuid)
RETURNS jsonb
LANGUAGE plpgsql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tables    text[];
    v_children  text[];
    v_payload   jsonb;
    v_table     text;
    v_child     text;
    v_pred      text;
    v_rows      jsonb;
BEGIN
    v_tables := _backup_tenant_tables();
    v_children := _backup_child_tables(v_tables);

    v_payload := jsonb_build_object(
        'tenant_id', p_tenant,
        'tables', '{}'::jsonb,
        'children', '{}'::jsonb,
        'users', '[]'::jsonb,
        'role_permissions', '[]'::jsonb
    );

    -- Tabelas de escopo direto.
    FOREACH v_table IN ARRAY v_tables LOOP
        EXECUTE format(
            'SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(t)), ''[]''::jsonb) FROM public.%I t WHERE t.tenant_id = $1',
            v_table)
        INTO v_rows USING p_tenant;
        v_payload := jsonb_set(v_payload, ARRAY['tables', v_table], v_rows);
    END LOOP;

    -- Tabelas-filhas (sem tenant_id).
    FOREACH v_child IN ARRAY v_children LOOP
        v_pred := _backup_child_predicate(v_child, v_tables);
        IF v_pred IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE format(
            'SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(c)), ''[]''::jsonb) FROM public.%I c WHERE %s',
            v_child, v_pred)
        INTO v_rows USING p_tenant;
        v_payload := jsonb_set(v_payload, ARRAY['children', v_child], v_rows);
    END LOOP;

    -- Identidades (users) referenciadas pelos vinculos do tenant.
    SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(u)), '[]'::jsonb)
    INTO v_rows
    FROM users u
    WHERE u.id IN (SELECT user_id FROM memberships WHERE tenant_id = p_tenant);
    v_payload := jsonb_set(v_payload, ARRAY['users'], v_rows);

    -- Permissoes dos papeis do tenant.
    SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(rp)), '[]'::jsonb)
    INTO v_rows
    FROM role_permissions rp
    WHERE rp.role_id IN (SELECT id FROM roles WHERE tenant_id = p_tenant);
    v_payload := jsonb_set(v_payload, ARRAY['role_permissions'], v_rows);

    RETURN v_payload;
END;
$$;

-- ---------------------------------------------------------------------------
-- 5) Restore: substitui todo o conteudo do tenant
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION restore_tenant(p_tenant uuid, p_payload jsonb)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tables    text[];
    v_children  text[];
    v_table     text;
    v_child     text;
    v_pred      text;
    v_rows      jsonb;
    v_elem      jsonb;
    v_email     text;
    v_phone     text;
    v_old       uuid;
    v_new       uuid;
    v_map       jsonb := '{}'::jsonb;
    v_ms        jsonb;
    v_counts    jsonb := '{}'::jsonb;
    v_n         integer;
BEGIN
    IF p_payload IS NULL OR jsonb_typeof(p_payload) <> 'object' THEN
        RAISE EXCEPTION 'payload de backup invalido' USING ERRCODE = '22023';
    END IF;

    v_tables := _backup_tenant_tables();
    v_children := _backup_child_tables(v_tables);

    -- Desliga triggers de usuario (append-only/FK/guardas) durante a troca.
    -- Roda como dono (postgres/superuser) por ser SECURITY DEFINER.
    PERFORM set_config('session_replication_role', 'replica', true);

    -- 1) Remove as linhas-filhas ANTES dos pais (sem cascade no modo replica).
    FOREACH v_child IN ARRAY v_children LOOP
        v_pred := _backup_child_predicate(v_child, v_tables);
        IF v_pred IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE format('DELETE FROM public.%I c WHERE %s', v_child, v_pred) USING p_tenant;
    END LOOP;

    -- 2) Remove o conteudo atual do tenant (todas as tabelas com tenant_id).
    FOREACH v_table IN ARRAY v_tables LOOP
        EXECUTE format('DELETE FROM public.%I WHERE tenant_id = $1', v_table) USING p_tenant;
    END LOOP;

    -- 3) Identidades: faz upsert por id/email/telefone e monta o mapa
    --    old_user_id -> new_user_id (identidades sao globais e podem ja existir).
    FOR v_elem IN SELECT * FROM jsonb_array_elements(COALESCE(p_payload->'users', '[]'::jsonb)) LOOP
        v_old := NULLIF(v_elem->>'id', '')::uuid;
        v_email := NULLIF(v_elem->>'email', '');
        v_phone := NULLIF(v_elem->>'phone', '');
        v_new := NULL;

        IF v_old IS NOT NULL THEN
            SELECT id INTO v_new FROM users WHERE id = v_old;
        END IF;
        IF v_new IS NULL AND v_email IS NOT NULL THEN
            SELECT id INTO v_new FROM users WHERE email = v_email;
        END IF;
        IF v_new IS NULL AND v_phone IS NOT NULL THEN
            SELECT id INTO v_new FROM users WHERE phone = v_phone;
        END IF;

        IF v_new IS NULL THEN
            -- Nao existe: insere a identidade original (nunca elevando a
            -- platform admin a partir de um backup de tenant). O campo e
            -- forcado para false porque o INSERT monta TODAS as colunas.
            PERFORM _backup_insert_rows('users',
                jsonb_build_array(v_elem || jsonb_build_object('is_platform_admin', false)));
            v_new := v_old;
        ELSE
            UPDATE users SET
                email                = COALESCE(v_email, email),
                phone                = CASE WHEN v_phone IS NOT NULL THEN v_phone ELSE phone END,
                password_hash        = COALESCE(NULLIF(v_elem->>'password_hash', ''), password_hash),
                full_name            = COALESCE(v_elem->>'full_name', full_name),
                is_active            = COALESCE((v_elem->>'is_active')::boolean, is_active),
                mfa_enabled          = COALESCE((v_elem->>'mfa_enabled')::boolean, mfa_enabled),
                mfa_secret           = COALESCE(v_elem->>'mfa_secret', mfa_secret),
                must_change_password = COALESCE((v_elem->>'must_change_password')::boolean, must_change_password)
            WHERE id = v_new;
        END IF;

        IF v_old IS NOT NULL AND v_new IS NOT NULL THEN
            v_map := v_map || jsonb_build_object(v_old::text, v_new::text);
        END IF;
    END LOOP;

    -- 4) Reescreve memberships.user_id com o mapa de identidades.
    IF p_payload->'tables' ? 'memberships' THEN
        v_ms := p_payload->'tables'->'memberships';
        v_ms := (
            SELECT COALESCE(jsonb_agg(
                CASE WHEN v_map ? (e->>'user_id')
                     THEN jsonb_set(e, '{user_id}', to_jsonb(v_map->>(e->>'user_id')))
                     ELSE e END), '[]'::jsonb)
            FROM jsonb_array_elements(v_ms) e
        );
        p_payload := jsonb_set(p_payload, ARRAY['tables', 'memberships'], v_ms);
    END IF;

    -- 5) Reinsere as tabelas de escopo do tenant.
    FOR v_table, v_rows IN
        SELECT key, value FROM jsonb_each(COALESCE(p_payload->'tables', '{}'::jsonb))
    LOOP
        v_n := _backup_insert_rows(v_table, v_rows);
        v_counts := jsonb_set(v_counts, ARRAY[v_table], to_jsonb(v_n));
    END LOOP;

    -- 6) Permissoes dos papeis.
    v_n := _backup_insert_rows('role_permissions', p_payload->'role_permissions');
    v_counts := jsonb_set(v_counts, ARRAY['role_permissions'], to_jsonb(v_n));

    -- 7) Reinsere as tabelas-filhas.
    FOR v_child, v_rows IN
        SELECT key, value FROM jsonb_each(COALESCE(p_payload->'children', '{}'::jsonb))
    LOOP
        v_n := _backup_insert_rows(v_child, v_rows);
        v_counts := jsonb_set(v_counts, ARRAY[v_child], to_jsonb(v_n));
    END LOOP;

    RETURN jsonb_build_object('tenant_id', p_tenant, 'counts', v_counts, 'users_mapped', v_map);
END;
$$;

GRANT EXECUTE ON FUNCTION export_tenant(uuid)               TO chosenerp_app;
GRANT EXECUTE ON FUNCTION restore_tenant(uuid, jsonb)       TO chosenerp_app;
