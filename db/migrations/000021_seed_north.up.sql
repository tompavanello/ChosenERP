-- 000021_seed_north.up.sql
-- Seed da filial Norte + usuário pastor.norte@demo.local (item #40 do backlog).
--
-- Contexto: AGENTS.md, README.md e start.ps1 documentam a conta
-- `pastor.norte@demo.local` (escopo Filial) como credencial de desenvolvimento,
-- mas NENHUMA migração a criava — ela existia no banco de dev porque foi criada
-- à mão. Este seed torna o ambiente reproduzível do zero.
--
-- É idempotente: se a filial (slug `norte`) ou o usuário (email) já existirem,
-- o insert é ignorado. As FKs são resolvidas por subquery (slug/key), então os
-- IDs determinísticos abaixo valem apenas em instalação nova.
--
-- ⚠️ O hash da senha abaixo corresponde a `DEMO_NORTE_PASSWORD` no `.env` do
-- ambiente de desenvolvimento. Numa instalação NOVA ele cria a conta com essa
-- senha de demonstração: troque-a logo após o primeiro boot. A senha canônica
-- fica no `.env`, fora do versionamento.

-- Filial Norte (congregação vinculada à Sede Matriz).
INSERT INTO branches (id, tenant_id, parent_id, name, slug, kind)
SELECT
    '22222222-2222-2222-2222-222222222223',
    t.id,
    (SELECT b.id FROM branches b WHERE b.tenant_id = t.id AND b.slug = 'matriz'),
    'Congregacao Norte',
    'norte',
    'congregation'
FROM tenants t
WHERE t.slug = 'demo'
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Usuário pastor da filial (papel pastor_filial).
INSERT INTO users (id, tenant_id, branch_id, role_id, email, password_hash, full_name)
SELECT
    '33333333-3333-3333-3333-333333333334',
    t.id,
    (SELECT b.id FROM branches b WHERE b.tenant_id = t.id AND b.slug = 'norte'),
    (SELECT r.id FROM roles r WHERE r.tenant_id = t.id AND r.key = 'pastor_filial'),
    'pastor.norte@demo.local',
    '$2a$10$aiGRgEZKccDwjk7uSs9A7u1q7V8cNvLPiGiW72DvPsrRjMmV/GY5m',
    'Pastor Norte'
FROM tenants t
WHERE t.slug = 'demo'
ON CONFLICT (email) DO NOTHING;

-- Permissões do pastor de filial: operação local do dia a dia, sem exclusão de
-- membros e sem escrita financeira (o tesoureiro é outro perfil).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN (
    'members.read', 'members.write', 'families.read',
    'visitors.read', 'finance.read', 'ministries.read',
    'ministries.write', 'reports.read', 'governance.read'
)
WHERE r.tenant_id = (SELECT id FROM tenants WHERE slug = 'demo')
  AND r.key = 'pastor_filial'
ON CONFLICT DO NOTHING;
