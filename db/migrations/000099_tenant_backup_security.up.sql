-- 000099_tenant_backup_security.up.sql
-- Endurecimento do backup/restore (000097/000098).
--
-- 1) `restore_tenant` passa a:
--    - validar que o `tenant_id` do payload e igual ao tenant de destino;
--    - aceitar SOMENTE tabelas de escopo do tenant e tabelas-filhas conhecidas
--      (allow-list derivada do catalogo). Um pacote forjado com chaves extras
--      (ex.: `users`, `permissions`) e REJEITADO em vez de inserir em qualquer
--      tabela publica como superusuario (escalada de privilegio);
--    - restaurar os dados da propria linha de `tenants` (branding, plano,
--      features, Pix, conteudo ao vivo/Biblia) por uma lista de colunas segura.
-- 2) `export_tenant` passa a incluir a linha de `tenants` no payload.
-- 3) As funcoes internas `_backup_*` deixam de ser executaveis por PUBLIC/app.

-- ---------------------------------------------------------------------------
-- 1) Export: inclui a linha de `tenants`.
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
        'tenant', '{}'::jsonb,
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

    -- Configuracao da propria igreja (branding/plano/features/pix/conteudo).
    SELECT to_jsonb(t) INTO v_rows FROM tenants t WHERE t.id = p_tenant;
    v_payload := jsonb_set(v_payload, ARRAY['tenant'], COALESCE(v_rows, '{}'::jsonb));

    RETURN v_payload;
END;
$$;

-- ---------------------------------------------------------------------------
-- 2) Restore: valida tenant, allow-list de tabelas e restaura `tenants`.
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
    v_tenant    jsonb;
BEGIN
    IF p_payload IS NULL OR jsonb_typeof(p_payload) <> 'object' THEN
        RAISE EXCEPTION 'payload de backup invalido' USING ERRCODE = '22023';
    END IF;

    -- O backup so pode ser restaurado na MESMA igreja de origem.
    IF (p_payload->>'tenant_id') IS NOT NULL
       AND NULLIF(p_payload->>'tenant_id', '') IS NOT NULL
       AND (p_payload->>'tenant_id')::uuid <> p_tenant THEN
        RAISE EXCEPTION 'o backup pertence a outra igreja' USING ERRCODE = '22023';
    END IF;

    v_tables := _backup_tenant_tables();
    v_children := _backup_child_tables(v_tables);

    -- Allow-list: rejeita chaves desconhecidas em `tables`/`children`.
    IF EXISTS (
        SELECT 1 FROM jsonb_object_keys(COALESCE(p_payload->'tables', '{}'::jsonb)) AS k
        WHERE k <> ALL (v_tables)
    ) THEN
        RAISE EXCEPTION 'payload de backup contem tabela nao autorizada' USING ERRCODE = '22023';
    END IF;
    IF EXISTS (
        SELECT 1 FROM jsonb_object_keys(COALESCE(p_payload->'children', '{}'::jsonb)) AS k
        WHERE k <> ALL (v_children) AND k <> 'role_permissions'
    ) THEN
        RAISE EXCEPTION 'payload de backup contem tabela-filha nao autorizada' USING ERRCODE = '22023';
    END IF;

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

    -- 2) role_permissions (tratada a parte): remove as dos papeis do tenant.
    DELETE FROM role_permissions rp
    USING roles r
    WHERE rp.role_id = r.id AND r.tenant_id = p_tenant;

    -- 3) Remove o conteudo atual do tenant (todas as tabelas com tenant_id).
    FOREACH v_table IN ARRAY v_tables LOOP
        EXECUTE format('DELETE FROM public.%I WHERE tenant_id = $1', v_table) USING p_tenant;
    END LOOP;

    -- 4) Identidades: faz upsert por id/email/telefone e monta o mapa
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
            -- platform admin a partir de um backup de tenant).
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

    -- 5) Reescreve memberships.user_id com o mapa de identidades.
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

    -- 6) Reinsere as tabelas de escopo do tenant (ja validadas pela allow-list).
    FOR v_table, v_rows IN
        SELECT key, value FROM jsonb_each(COALESCE(p_payload->'tables', '{}'::jsonb))
    LOOP
        v_n := _backup_insert_rows(v_table, v_rows);
        v_counts := jsonb_set(v_counts, ARRAY[v_table], to_jsonb(v_n));
    END LOOP;

    -- 7) Permissoes dos papeis.
    v_n := _backup_insert_rows('role_permissions', p_payload->'role_permissions');
    v_counts := jsonb_set(v_counts, ARRAY['role_permissions'], to_jsonb(v_n));

    -- 8) Reinsere as tabelas-filhas. `role_permissions` e ignorada aqui (ja
    --    foi inserida no passo 7), inclusive em pacotes antigos que a traziam
    --    duplicada dentro de `children`.
    FOR v_child, v_rows IN
        SELECT key, value FROM jsonb_each(COALESCE(p_payload->'children', '{}'::jsonb))
    LOOP
        IF v_child = 'role_permissions' THEN
            CONTINUE;
        END IF;
        v_n := _backup_insert_rows(v_child, v_rows);
        v_counts := jsonb_set(v_counts, ARRAY[v_child], to_jsonb(v_n));
    END LOOP;

    -- 9) Restaura a configuracao da igreja (somente colunas conhecidas; so
    --    sobrescreve as chaves presentes no payload).
    v_tenant := COALESCE(p_payload->'tenant', '{}'::jsonb);
    IF jsonb_typeof(v_tenant) = 'object' AND v_tenant <> '{}'::jsonb THEN
        UPDATE tenants t SET
            name             = CASE WHEN v_tenant ? 'name'             THEN v_tenant->>'name'             ELSE t.name END,
            legal_name       = CASE WHEN v_tenant ? 'legal_name'       THEN v_tenant->>'legal_name'       ELSE t.legal_name END,
            cnpj             = CASE WHEN v_tenant ? 'cnpj'             THEN v_tenant->>'cnpj'             ELSE t.cnpj END,
            plan             = CASE WHEN v_tenant ? 'plan'             THEN v_tenant->>'plan'             ELSE t.plan END,
            locale           = CASE WHEN v_tenant ? 'locale'           THEN v_tenant->>'locale'           ELSE t.locale END,
            timezone         = CASE WHEN v_tenant ? 'timezone'         THEN v_tenant->>'timezone'         ELSE t.timezone END,
            is_active        = CASE WHEN v_tenant ? 'is_active'        THEN (v_tenant->>'is_active')::boolean ELSE t.is_active END,
            limits           = CASE WHEN v_tenant ? 'limits'           THEN v_tenant->'limits'           ELSE t.limits END,
            features         = CASE WHEN v_tenant ? 'features'         THEN v_tenant->'features'         ELSE t.features END,
            billing_status   = CASE WHEN v_tenant ? 'billing_status'   THEN v_tenant->>'billing_status'   ELSE t.billing_status END,
            trial_ends_at    = CASE WHEN v_tenant ? 'trial_ends_at'    THEN NULLIF(v_tenant->>'trial_ends_at','')::timestamptz ELSE t.trial_ends_at END,
            billing_exempt   = CASE WHEN v_tenant ? 'billing_exempt'   THEN (v_tenant->>'billing_exempt')::boolean ELSE t.billing_exempt END,
            past_due_since   = CASE WHEN v_tenant ? 'past_due_since'   THEN NULLIF(v_tenant->>'past_due_since','')::timestamptz ELSE t.past_due_since END,
            suspended_at     = CASE WHEN v_tenant ? 'suspended_at'     THEN NULLIF(v_tenant->>'suspended_at','')::timestamptz ELSE t.suspended_at END,
            logo_url         = CASE WHEN v_tenant ? 'logo_url'         THEN v_tenant->>'logo_url'         ELSE t.logo_url END,
            brand_color      = CASE WHEN v_tenant ? 'brand_color'      THEN v_tenant->>'brand_color'      ELSE t.brand_color END,
            favicon_url      = CASE WHEN v_tenant ? 'favicon_url'      THEN v_tenant->>'favicon_url'      ELSE t.favicon_url END,
            custom_domain    = CASE WHEN v_tenant ? 'custom_domain'    THEN v_tenant->>'custom_domain'    ELSE t.custom_domain END,
            pix_key          = CASE WHEN v_tenant ? 'pix_key'          THEN v_tenant->>'pix_key'          ELSE t.pix_key END,
            pix_name         = CASE WHEN v_tenant ? 'pix_name'         THEN v_tenant->>'pix_name'         ELSE t.pix_name END,
            youtube_channel_id = CASE WHEN v_tenant ? 'youtube_channel_id' THEN v_tenant->>'youtube_channel_id' ELSE t.youtube_channel_id END,
            live_stream_url  = CASE WHEN v_tenant ? 'live_stream_url'  THEN v_tenant->>'live_stream_url'  ELSE t.live_stream_url END,
            bible_version_id = CASE WHEN v_tenant ? 'bible_version_id' THEN v_tenant->>'bible_version_id' ELSE t.bible_version_id END,
            updated_at       = now()
        WHERE t.id = p_tenant;
    END IF;

    RETURN jsonb_build_object('tenant_id', p_tenant, 'counts', v_counts, 'users_mapped', v_map);
END;
$$;

-- ---------------------------------------------------------------------------
-- 3) Hardening: funcoes internas nao devem ser executaveis por PUBLIC/app.
--    (export_tenant/restore_tenant as chamam como SECURITY DEFINER/dono.)
-- ---------------------------------------------------------------------------
REVOKE ALL ON FUNCTION _backup_tenant_tables()                 FROM PUBLIC;
REVOKE ALL ON FUNCTION _backup_child_tables(text[])            FROM PUBLIC;
REVOKE ALL ON FUNCTION _backup_child_predicate(text, text[])   FROM PUBLIC;
REVOKE ALL ON FUNCTION _backup_insert_rows(text, jsonb)        FROM PUBLIC;
REVOKE ALL ON FUNCTION _backup_insert_rows(text, jsonb)        FROM chosenerp_app;

GRANT EXECUTE ON FUNCTION export_tenant(uuid)                 TO chosenerp_app;
GRANT EXECUTE ON FUNCTION restore_tenant(uuid, jsonb)         TO chosenerp_app;
