-- 000035_branch_cnpj.down.sql
DROP INDEX IF EXISTS uq_branches_tenant_cnpj;
ALTER TABLE branches DROP COLUMN IF EXISTS cnpj;
