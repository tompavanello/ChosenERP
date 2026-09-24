-- 000058_reset_operational_data.up.sql
-- Funcao de RESET para testes operacionais: apaga TODO o dado operacional
-- (membros, financeiro, eventos, kids, governanca, comunicados, auditoria...)
-- mantendo apenas a base: tenants, branches, roles, permissions,
-- role_permissions, users, memberships e schema_migrations. Depois remove os
-- vinculos/identidades que nao sejam de super_admin, deixando so o superadmin.
--
-- SECURITY DEFINER: roda como dono (postgres), entao ignora RLS. O acesso e
-- controlado no handler da API (apenas super_admin) e pelo GRANT abaixo.
CREATE OR REPLACE FUNCTION reset_operational_data()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    -- Tabelas que NAO entram no truncate (base do sistema).
    keep text[] := ARRAY[
        'tenants', 'branches', 'roles', 'permissions', 'role_permissions',
        'users', 'memberships', 'schema_migrations'
    ];
    tbls text;
BEGIN
    -- 1) Trunca todas as demais tabelas do schema public. O CASCADE cobre
    --    tabelas de vinculo (sem tenant_id proprio) que referenciam as pais.
    SELECT string_agg(format('%I.%I', t.table_schema, t.table_name), ', ')
    INTO tbls
    FROM information_schema.tables t
    WHERE t.table_schema = 'public'
      AND t.table_type = 'BASE TABLE'
      AND t.table_name <> ALL (keep);

    IF tbls IS NOT NULL THEN
        EXECUTE 'TRUNCATE TABLE ' || tbls || ' RESTART IDENTITY CASCADE';
    END IF;

    -- 2) Remove vinculos que nao sejam de super_admin.
    DELETE FROM memberships m
    USING roles r
    WHERE m.role_id = r.id
      AND r.key <> 'super_admin';

    -- 3) Remove identidades que ficaram sem nenhum vinculo.
    DELETE FROM users u
    WHERE NOT EXISTS (SELECT 1 FROM memberships m WHERE m.user_id = u.id);
END;
$$;

GRANT EXECUTE ON FUNCTION reset_operational_data() TO chosenerp_app;
