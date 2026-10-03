-- 000087_announcements_permission.down.sql
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM permissions WHERE key IN ('announcements.read','announcements.write'));
DELETE FROM permissions WHERE key IN ('announcements.read','announcements.write');
