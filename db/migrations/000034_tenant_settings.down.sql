-- 000034_tenant_settings.down.sql
DROP POLICY IF EXISTS tenant_upd ON tenants;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE key IN ('settings.read','settings.write'));

DELETE FROM permissions WHERE key IN ('settings.read','settings.write');
