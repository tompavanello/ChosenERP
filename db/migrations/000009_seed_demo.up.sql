-- 000009_seed_demo.up.sql
-- Permissões padrão + dados de demonstração (tenant, branch, papéis, admin)

-- Permissões padrão (catálogo)
INSERT INTO permissions (key, module, name) VALUES
    ('members.read',       'people', 'Ler membros'),
    ('members.write',      'people', 'Criar/editar membros'),
    ('members.delete',     'people', 'Excluir membro'),
    ('families.read',      'people', 'Ler famílias'),
    ('visitors.read',      'people', 'Ler visitantes'),
    ('finance.read',       'finance', 'Ler financeiro'),
    ('finance.write',      'finance', 'Lancamento financeiro'),
    ('finance.authorize',  'finance', 'Aprovar orçamento'),
    ('ministries.read',    'ministries', 'Ler ministérios'),
    ('ministries.write',   'ministries', 'Gerir ministérios/escalas'),
    ('governance.read',    'governance', 'Ler governança'),
    ('governance.write',   'governance', 'Gerir atas/votações'),
    ('reports.read',       'reports', 'Ler relatórios')
ON CONFLICT (key) DO NOTHING;

-- Tenant de demonstração
INSERT INTO tenants (id, name, slug, plan) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Igreja Demonstrativa', 'demo', 'starter')
ON CONFLICT (slug) DO NOTHING;

-- Filial Matriz (Sede)
INSERT INTO branches (id, tenant_id, name, slug, kind) VALUES
    ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111',
     'Sede Matriz', 'matriz', 'branch')
ON CONFLICT (tenant_id, slug) DO NOTHING;

-- Papéis padrão do tenant demo
INSERT INTO roles (tenant_id, key, name, is_system) VALUES
    ('11111111-1111-1111-1111-111111111111', 'super_admin', 'Super Admin', true),
    ('11111111-1111-1111-1111-111111111111', 'admin_sede',  'Admin da Sede', true),
    ('11111111-1111-1111-1111-111111111111', 'pastor_filial', 'Pastor da Filial', true),
    ('11111111-1111-1111-1111-111111111111', 'tesoureiro',  'Tesoureiro', true),
    ('11111111-1111-1111-1111-111111111111', 'secretario',  'Secretário(a)', true),
    ('11111111-1111-1111-1111-111111111111', 'membro',      'Membro', true)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- Usuário admin de demonstração
-- Senha: "admin123" (bcrypt) — trocar em produção
INSERT INTO users (id, tenant_id, branch_id, role_id, email, password_hash, full_name)
SELECT
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    (SELECT id FROM roles WHERE tenant_id='11111111-1111-1111-1111-111111111111' AND key='super_admin'),
    'admin@demo.local',
    '$2a$12$V4Zd8V3Y1m1M1K6b0Y6O9OQAQyN1nH1u3v9Zx6z1y2v9Y9z8x7w6S',
    'Admin Demonstração'
ON CONFLICT (email) DO NOTHING;

-- Concede permissões completas ao super_admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.tenant_id='11111111-1111-1111-1111-111111111111' AND r.key='super_admin'
ON CONFLICT DO NOTHING;
