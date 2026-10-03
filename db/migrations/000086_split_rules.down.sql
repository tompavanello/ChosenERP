-- 000086_split_rules.down.sql
DROP POLICY IF EXISTS transfer_rules_all ON transfer_rules;
DROP POLICY IF EXISTS transfer_rules_sel ON transfer_rules;
DROP TABLE IF EXISTS transfer_rules;
ALTER TABLE tenants DROP COLUMN IF EXISTS split_enabled;
