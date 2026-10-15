-- 000093_tenant_active.up.sql
-- Suspensao automatica por inadimplencia: expoe o estado da igreja para o
-- login/middleware sem depender do RLS (que exige tenant no contexto).
CREATE OR REPLACE FUNCTION tenant_is_active(p_tenant uuid)
RETURNS boolean
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
    SELECT COALESCE((SELECT is_active FROM tenants WHERE id = p_tenant), false)
$$;
