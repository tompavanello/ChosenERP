-- 000045_rls_write_headquarters.down.sql
-- Restaura rls_write() sem o escopo Sede (gravação apenas no branch exato).
CREATE OR REPLACE FUNCTION rls_write(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND rls_branch_match(p_branch, p_allow_global))
$$;
