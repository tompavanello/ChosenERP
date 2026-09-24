-- 000036_branch_hierarchy.down.sql
-- Desfaz a leitura hierarquica: rls_read volta a definicao da 000016.
DROP FUNCTION IF EXISTS rls_read_scope(uuid, boolean);

CREATE OR REPLACE FUNCTION rls_read(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND (rls_branch_match(p_branch, p_allow_global) OR is_headquarters()))
$$;
