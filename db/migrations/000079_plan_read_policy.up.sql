-- 000079_plan_read_policy.up.sql
-- Correcao: o catalogo `plans` era legivel apenas por system/platform admin
-- (000075), mas a resolucao de entitlements roda no escopo da IGREJA. Sem ler o
-- proprio plano, todas as features caiam no default (habilitadas). Aqui
-- liberamos a LEITURA do plano usado pela igreja ativa.
DROP POLICY IF EXISTS plans_sel ON plans;
CREATE POLICY plans_sel ON plans FOR SELECT USING (
    is_system()
    OR is_platform_admin()
    OR key = (SELECT t.plan FROM tenants t WHERE t.id = current_tenant())
);
