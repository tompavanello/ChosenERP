-- 000009_seed_demo.down.sql
DELETE FROM role_permissions WHERE role_id IN (
    SELECT id FROM roles WHERE tenant_id = '11111111-1111-1111-1111-111111111111'
);
DELETE FROM users WHERE id = '33333333-3333-3333-3333-333333333333';
DELETE FROM roles WHERE tenant_id = '11111111-1111-1111-1111-111111111111';
DELETE FROM branches WHERE id = '22222222-2222-2222-2222-222222222222';
DELETE FROM tenants WHERE id = '11111111-1111-1111-1111-111111111111';
-- NOTA: `permissions` e `role_permissions` sao catalogo GLOBAL, populado por
-- varias migracoes (000003, 000050, 000062, ...). O down deste seed NAO deve
-- apaga-lo: `DELETE FROM permissions` (com CASCADE em role_permissions)
-- destruiria o RBAC de TODAS as igrejas. Os vinculos do tenant demo ja foram
-- removidos no topo deste arquivo.
