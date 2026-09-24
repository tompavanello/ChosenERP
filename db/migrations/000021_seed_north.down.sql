-- 000021_seed_north.down.sql
-- Reverte o seed da filial Norte. O usuário e a filial só são removidos se
-- existirem; a filial pode ter dados vinculados (membros/financeiro), então o
-- DELETE falharia por FK — nesse caso, remova os dados antes manualmente.
DELETE FROM role_permissions
WHERE role_id IN (
    SELECT r.id FROM roles r
    JOIN tenants t ON t.id = r.tenant_id
    WHERE t.slug = 'demo' AND r.key = 'pastor_filial'
);

DELETE FROM users WHERE email = 'pastor.norte@demo.local';

DELETE FROM branches
WHERE slug = 'norte'
  AND tenant_id = (SELECT id FROM tenants WHERE slug = 'demo');
