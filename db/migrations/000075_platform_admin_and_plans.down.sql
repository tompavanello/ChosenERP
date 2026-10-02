-- 000075_platform_admin_and_plans.down.sql
-- Reverte o console da plataforma: remove catalogo de planos, overrides de
-- limite e a flag de platform admin.

DROP TABLE IF EXISTS plans;
ALTER TABLE tenants DROP COLUMN IF EXISTS limits;
DROP FUNCTION IF EXISTS is_platform_admin();
ALTER TABLE users DROP COLUMN IF EXISTS is_platform_admin;
