-- 000009_seed_demo.up.sql
-- Permissoes padrao + dados de demonstracao (tenant, branch, papeis, admin)

-- Permissoes padrao (catalogo)
INSERT INTO permissions (key, module, name) VALUES
    ('members.read',       'people', 'Ler membros'),
    ('members.write',      'people', 'Criar/editar membros'),
    ('members.delete',     'people', 'Excluir membro'),
    ('families.read',      'people', 'Ler familias'),
    ('visitors.read',      'people', 'Ler visitantes'),
    ('finance.read',       'finance', 'Ler financeiro'),
    ('finance.write',      'finance', 'Lancamento financeiro'),
    ('finance.authorize',  'finance', 'Aprovar orcamento'),
    ('ministries.read',    'ministries', 'Ler ministerios'),
    ('ministries.write',   'ministries', 'Gerir ministerios/escalas'),
    ('governance.read',    'governance', 'Ler governanca'),
    ('governance.write',   'governance', 'Gerir atas/votacoes'),
    ('reports.read',       'reports', 'Ler relatorios')
ON CONFLICT (key) DO NOTHING;

-- Tenant de demonstracao
INSERT INTO tenants (id, name, slug, plan) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Igreja Demonstrativa', 'demo', 'starter')
ON CONFLICT (slug) DO NOTHING;

-- Filial Matriz (Sede)
INSERT INTO branches (id, tenant_id, name, slug, kind) VALUES
    ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111',
     'Sede Matriz', 'matriz', 'branch')
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Papeis padrao do tenant demo
INSERT INTO roles (tenant_id, key, name, is_system) VALUES
    ('11111111-1111-1111-1111-111111111111', 'super_admin', 'Super Admin', true),
    ('11111111-1111-1111-1111-111111111111', 'admin_sede',  'Admin da Sede', true),
    ('11111111-1111-1111-1111-111111111111', 'pastor_filial', 'Pastor da Filial', true),
    ('11111111-1111-1111-1111-111111111111', 'tesoureiro',  'Tesoureiro', true),
    ('11111111-1111-1111-1111-111111111111', 'secretario',  'Secretario(a)', true),
    ('11111111-1111-1111-1111-111111111111', 'membro',      'Membro', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Usuario admin de demonstracao.
--
--  O hash abaixo e de uma senha demo ja comprometida (publicada na
-- documentacao ate 2026-09-22) e nao corresponde mais a conta do ambiente de
-- desenvolvimento, cuja senha foi rotacionada. Numa instalacao NOVA, este seed
-- cria o admin com a senha antiga: troque-a logo apos o primeiro boot, ou o
-- ambiente nasce com credencial conhecida. A senha atual fica no .env
-- (DEMO_ADMIN_PASSWORD), fora do versionamento.
INSERT INTO users (id, tenant_id, branch_id, role_id, email, password_hash, full_name)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    (SELECT id FROM roles WHERE tenant_id='11111111-1111-1111-1111-111111111111' AND key='super_admin'),
    'admin@demo.local',
    '$2a$12$V4Zd8V3Y1m1M1K6b0Y6O9OQAQyN1nH1u3v9Zx6z1y2v9Y9z8x7w6S',
    'Admin Demonstracao'
ON CONFLICT (email) DO NOTHING;

-- Concede permissoes completas ao super_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.tenant_id='11111111-1111-1111-1111-111111111111' AND r.key='super_admin'
ON CONFLICT DO NOTHING;
