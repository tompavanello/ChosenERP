-- 000034_tenant_settings.up.sql
-- Configuracao do tenant e suas filiais:
--   * permissoes settings.read/settings.write (tela "Configuracoes");
--   * politica de UPDATE em tenants (antes so havia SELECT - nao era possivel
--     editar nome/razao social/CNPJ/fuso da igreja).
--
-- As filiais ja tem politica de escrita (branch_all, migracao 000016): a
-- autorizacao e feita no handler (so Sede/admin).

-- ---------------------------------------------------------------------------
-- 1) Permissoes de configuracao
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('settings.read',  'admin', 'Ver configuracoes'),
    ('settings.write', 'admin', 'Gerir configuracoes e filiais')
ON CONFLICT (key) DO NOTHING;

-- Sede do tenant tem controle total; demais perfis nao acessam a tela.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.key IN ('settings.read','settings.write')
WHERE r.key IN ('super_admin','admin_sede')
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2) tenants: permitir UPDATE do proprio tenant (escrita segue autorizada no
--    handler). is_system() mantem os workers internos funcionando.
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS tenant_upd ON tenants;
CREATE POLICY tenant_upd ON tenants
  FOR UPDATE USING (id = current_tenant() OR is_system())
  WITH CHECK (id = current_tenant() OR is_system());
