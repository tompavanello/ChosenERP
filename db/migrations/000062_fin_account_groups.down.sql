-- 000062_fin_account_groups.down.sql
DROP INDEX IF EXISTS idx_fin_cat_group;
ALTER TABLE financial_categories DROP COLUMN IF EXISTS group_id;

DROP POLICY IF EXISTS fincatgrp_all ON financial_category_groups;
DROP POLICY IF EXISTS fincatgrp_sel ON financial_category_groups;
ALTER TABLE financial_category_groups DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS financial_category_groups;
