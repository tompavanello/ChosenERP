-- 000034_tenant_settings.up.sql
-- Configuração do tenant e suas filiais:
--   * permissões settings.read/settings.write (tela "Configurações");
--   * política de UPDATE em tenants (antes só havia SELECT — não era possível
--     editar nome/razão social/CNPJ/fuso da igreja).
--
-- As filiais já têm política de escrita (branch_all, migração 000016): a
-- autorização é feita no handler (só Sede/admin).

-- ---------------------------------------------------------------------------
-- 1) Permissões de configuração
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('settings.read',  'admin', 'Ver configurações'),
    ('settings.write', 'admin', 'Gerir configurações e filiais')
ON CONFLICT (key) DO NOTHING;

-- Sede do tenant tem controle total; demais perfis não acessam a tela.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN ('settings.read','settings.write')
WHERE r.key IN ('super_admin','admin_sede')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2) tenants: permitir UPDATE do próprio tenant (escrita segue autorizada no
--    handler). is_system() mantém os workers internos funcionando.
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS tenant_upd ON tenants;
CREATE POLICY tenant_upd ON tenants
  FOR UPDATE USING (id = current_tenant() OR is_system())
  WITH CHECK (id = current_tenant() OR is_system());
