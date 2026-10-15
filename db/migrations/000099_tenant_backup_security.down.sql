-- 000099_tenant_backup_security.down.sql
-- Reverte para as definicoes de export_tenant (000097) e restore_tenant
-- (000098) e restaura as permissoes de execucao das funcoes internas.

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

    FOREACH v_table IN ARRAY v_tables LOOP
        EXECUTE format(
            'SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(t)), ''[]''::jsonb) FROM public.%I t WHERE t.tenant_id = $1',
            v_table)
        INTO v_rows USING p_tenant;
        v_payload := jsonb_set(v_payload, ARRAY['tables', v_table], v_rows);
    END LOOP;

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

    SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(u)), '[]'::jsonb)
    INTO v_rows
    FROM users u
    WHERE u.id IN (SELECT user_id FROM memberships WHERE tenant_id = p_tenant);
    v_payload := jsonb_set(v_payload, ARRAY['users'], v_rows);

    SELECT COALESCE(jsonb_agg(DISTINCT to_jsonb(rp)), '[]'::jsonb)
    INTO v_rows
    FROM role_permissions rp
    WHERE rp.role_id IN (SELECT id FROM roles WHERE tenant_id = p_tenant);
    v_payload := jsonb_set(v_payload, ARRAY['role_permissions'], v_rows);

    RETURN v_payload;
END;
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

    PERFORM set_config('session_replication_role', 'replica', true);

    FOREACH v_child IN ARRAY v_children LOOP
        v_pred := _backup_child_predicate(v_child, v_tables);
        IF v_pred IS NULL THEN
            CONTINUE;
        END IF;
        EXECUTE format('DELETE FROM public.%I c WHERE %s', v_child, v_pred) USING p_tenant;
    END LOOP;

    DELETE FROM role_permissions rp
    USING roles r
    WHERE rp.role_id = r.id AND r.tenant_id = p_tenant;

    FOREACH v_table IN ARRAY v_tables LOOP
        EXECUTE format('DELETE FROM public.%I WHERE tenant_id = $1', v_table) USING p_tenant;
    END LOOP;

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

    FOR v_table, v_rows IN
        SELECT key, value FROM jsonb_each(COALESCE(p_payload->'tables', '{}'::jsonb))
    LOOP
        v_n := _backup_insert_rows(v_table, v_rows);
        v_counts := jsonb_set(v_counts, ARRAY[v_table], to_jsonb(v_n));
    END LOOP;

    v_n := _backup_insert_rows('role_permissions', p_payload->'role_permissions');
    v_counts := jsonb_set(v_counts, ARRAY['role_permissions'], to_jsonb(v_n));

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

GRANT EXECUTE ON FUNCTION _backup_tenant_tables()               TO PUBLIC;
GRANT EXECUTE ON FUNCTION _backup_child_tables(text[])          TO PUBLIC;
GRANT EXECUTE ON FUNCTION _backup_child_predicate(text, text[]) TO PUBLIC;
GRANT EXECUTE ON FUNCTION _backup_insert_rows(text, jsonb)      TO PUBLIC;
