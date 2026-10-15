-- 000098_tenant_backup_fix.up.sql
-- Correcao do backup/restore (000097): `role_permissions` e uma tabela-filha
-- de `roles` (sem tenant_id proprio) e era exportada DUAS vezes - no campo
-- dedicado `role_permissions` E em `children` (pela deteccao generica de FK).
-- No restore isso reinseria as mesmas linhas e quebrava com
-- "duplicate key value violates unique constraint role_permissions_pkey".
--
-- Correcao: `role_permissions` sai da deteccao generica de filhas e passa a
-- ser removida explicitamente no restore (antes dos papeis).

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
      AND NOT (c.relname = ANY(p_tables))
      -- role_permissions e tratada a parte (tem campo dedicado no payload).
      AND c.relname <> 'role_permissions';
$$;

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

    -- 6) Reinsere as tabelas de escopo do tenant.
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

    RETURN jsonb_build_object('tenant_id', p_tenant, 'counts', v_counts, 'users_mapped', v_map);
END;
$$;
