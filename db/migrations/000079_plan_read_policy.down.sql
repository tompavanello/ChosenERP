-- 000079_plan_read_policy.down.sql
DROP POLICY IF EXISTS plans_sel ON plans;
CREATE POLICY plans_sel ON plans FOR SELECT USING (
    is_system() OR is_platform_admin()
);
