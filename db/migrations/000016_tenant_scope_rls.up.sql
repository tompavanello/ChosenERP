-- 000016_tenant_scope_rls.up.sql
-- Correcao de isolamento MULTI-TENANT nas politicas de RLS.
--
-- Problema: is_headquarters() verificava apenas branch nulo + papel
-- (super_admin/admin_sede), ignorando o tenant do contexto. Um usuario "Sede"
-- de qualquer tenant enxergava (e gravava em) dados de TODOS os tenants.
-- Detectado pelos testes automatizados de RLS (internal/store/rls_test.go).
--
-- Novo modelo de escopo:
--   * is_system()      -> workers internos (role 'system'): acesso total.
--   * is_headquarters()-> Sede, mas SOMENTE dentro do proprio tenant.
--   * rls_read()       -> leitura: tenant do contexto + (branch | global | Sede).
--   * rls_write()      -> gravacao: tenant do contexto + branch exato (ou global).
--   * rls_hq()         -> tenant do contexto + escopo Sede.

-- ---------------------------------------------------------------------------
-- Helpers de escopo
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION is_headquarters() RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT current_branch() IS NULL
       AND current_tenant() IS NOT NULL
       AND current_setting('app.role', true) IN ('super_admin','admin_sede')
$$;

-- Workers internos (outbox de entrega, recorrencias, app do membro).
-- Tanto store.WithSystem() (tenant vazio) quanto o worker de recorrencias
-- (role 'system' com tenant/filial definidos) caem aqui.
CREATE OR REPLACE FUNCTION is_system() RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT current_setting('app.role', true) = 'system'
$$;

-- A linha pertence ao branch do contexto (ou e registro global do tenant,
-- quando p_allow_global e branch_id e NULL).
CREATE OR REPLACE FUNCTION rls_branch_match(p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT p_branch = current_branch()
        OR (p_allow_global AND p_branch IS NULL)
$$;

-- Leitura: sistema, ou tenant do contexto + (branch | global | Sede do tenant).
CREATE OR REPLACE FUNCTION rls_read(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND (rls_branch_match(p_branch, p_allow_global) OR is_headquarters()))
$$;

-- Gravacao: sistema, ou tenant do contexto + branch exato (ou global).
-- Escopo Sede NAO ganha gravacao em branch alheio - so via rls_hq() explicito.
CREATE OR REPLACE FUNCTION rls_write(p_tenant uuid, p_branch uuid, p_allow_global boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND rls_branch_match(p_branch, p_allow_global))
$$;

-- Acesso restrito a Sede (ou sistema) dentro do proprio tenant.
CREATE OR REPLACE FUNCTION rls_hq(p_tenant uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT is_system()
        OR (current_tenant() IS NOT NULL
            AND p_tenant = current_tenant()
            AND is_headquarters())
$$;

-- ---------------------------------------------------------------------------
-- tenants / branches / roles / users / consent_terms  (escala de tenant)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS tenant_sel ON tenants;
CREATE POLICY tenant_sel ON tenants USING (id = current_tenant() OR is_system());

DROP POLICY IF EXISTS branch_sel ON branches;
CREATE POLICY branch_sel ON branches USING (tenant_id = current_tenant() OR is_system());
DROP POLICY IF EXISTS branch_all ON branches;
CREATE POLICY branch_all ON branches
  USING (tenant_id = current_tenant() OR is_system())
  WITH CHECK (tenant_id = current_tenant() OR is_system());

DROP POLICY IF EXISTS roles_sel ON roles;
CREATE POLICY roles_sel ON roles USING (tenant_id = current_tenant() OR is_system());
DROP POLICY IF EXISTS roles_all ON roles;
CREATE POLICY roles_all ON roles
  USING (tenant_id = current_tenant() OR is_system())
  WITH CHECK (tenant_id = current_tenant() OR is_system());

DROP POLICY IF EXISTS users_sel ON users;
CREATE POLICY users_sel ON users USING (tenant_id = current_tenant() OR is_system());
DROP POLICY IF EXISTS users_all ON users;
CREATE POLICY users_all ON users
  USING (tenant_id = current_tenant() OR is_system())
  WITH CHECK (tenant_id = current_tenant() OR is_system());

DROP POLICY IF EXISTS consent_terms_sel ON consent_terms;
CREATE POLICY consent_terms_sel ON consent_terms USING (tenant_id = current_tenant() OR is_system());
DROP POLICY IF EXISTS consent_terms_all ON consent_terms;
CREATE POLICY consent_terms_all ON consent_terms
  USING (tenant_id = current_tenant() OR is_system())
  WITH CHECK (tenant_id = current_tenant() OR is_system());

-- ---------------------------------------------------------------------------
-- Pessoas (branch NOT NULL)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS members_sel ON members;
CREATE POLICY members_sel ON members USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS members_all ON members;
CREATE POLICY members_all ON members
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS families_sel ON families;
CREATE POLICY families_sel ON families USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS families_all ON families;
CREATE POLICY families_all ON families
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS guests_sel ON visitors;
CREATE POLICY guests_sel ON visitors USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS visitors_all ON visitors;
CREATE POLICY visitors_all ON visitors
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS member_consents_sel ON member_consents;
CREATE POLICY member_consents_sel ON member_consents USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS member_consents_all ON member_consents;
CREATE POLICY member_consents_all ON member_consents
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- Tabelas sem tenant_id proprio: herdam o escopo pela tabela pai.
DROP POLICY IF EXISTS rels_sel ON member_relationships;
CREATE POLICY rels_sel ON member_relationships USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND rls_read(m.tenant_id, m.branch_id, false)
));
DROP POLICY IF EXISTS rels_all ON member_relationships;
CREATE POLICY rels_all ON member_relationships
  USING (EXISTS (SELECT 1 FROM members m WHERE m.id = member_id AND rls_write(m.tenant_id, m.branch_id, false)))
  WITH CHECK (EXISTS (SELECT 1 FROM members m WHERE m.id = member_id AND rls_write(m.tenant_id, m.branch_id, false)));

DROP POLICY IF EXISTS ministry_mem_sel ON ministry_members;
CREATE POLICY ministry_mem_sel ON ministry_members USING (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND rls_read(m.tenant_id, m.branch_id, false)
));
DROP POLICY IF EXISTS ministry_mem_all ON ministry_members;
CREATE POLICY ministry_mem_all ON ministry_members
  USING (EXISTS (SELECT 1 FROM ministries m WHERE m.id = ministry_id AND rls_write(m.tenant_id, m.branch_id, false)))
  WITH CHECK (EXISTS (SELECT 1 FROM ministries m WHERE m.id = ministry_id AND rls_write(m.tenant_id, m.branch_id, false)));

-- ---------------------------------------------------------------------------
-- Ministerios, grupos e frequencia (branch NOT NULL)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS ministries_sel ON ministries;
CREATE POLICY ministries_sel ON ministries USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS ministries_all ON ministries;
CREATE POLICY ministries_all ON ministries
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS groups_sel ON small_groups;
CREATE POLICY groups_sel ON small_groups USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS groups_all ON small_groups;
CREATE POLICY groups_all ON small_groups
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS att_sel ON group_attendance;
CREATE POLICY att_sel ON group_attendance USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS att_all ON group_attendance;
CREATE POLICY att_all ON group_attendance
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Financeiro
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS fincat_sel ON financial_categories;
CREATE POLICY fincat_sel ON financial_categories USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS fincat_all ON financial_categories;
CREATE POLICY fincat_all ON financial_categories
  USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

DROP POLICY IF EXISTS fintx_sel ON financial_transactions;
CREATE POLICY fintx_sel ON financial_transactions USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fintx_all ON financial_transactions;
CREATE POLICY fintx_all ON financial_transactions
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS recurring_sel ON recurring_donations;
CREATE POLICY recurring_sel ON recurring_donations FOR SELECT
  USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS recurring_ins ON recurring_donations;
CREATE POLICY recurring_ins ON recurring_donations FOR INSERT
  WITH CHECK (rls_write(tenant_id, branch_id, false));
DROP POLICY IF EXISTS recurring_upd ON recurring_donations;
CREATE POLICY recurring_upd ON recurring_donations FOR UPDATE
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- Repasses: visivel para origem/destino/Sede do tenant; criavel por
-- origem, destino ou Sede do tenant.
DROP POLICY IF EXISTS transfers_sel ON transfers;
CREATE POLICY transfers_sel ON transfers
  USING (rls_read(tenant_id, from_branch_id, false) OR rls_read(tenant_id, to_branch_id, false));
DROP POLICY IF EXISTS transfers_ins ON transfers;
CREATE POLICY transfers_ins ON transfers FOR INSERT
  WITH CHECK (
    rls_write(tenant_id, from_branch_id, false)
    OR rls_write(tenant_id, to_branch_id, false)
    OR rls_hq(tenant_id)
  );

-- ---------------------------------------------------------------------------
-- Documentos, avisos e fila de envio (branch nullable = global do tenant)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS docs_sel ON documents;
CREATE POLICY docs_sel ON documents USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS docs_all ON documents;
CREATE POLICY docs_all ON documents
  USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

DROP POLICY IF EXISTS deliveries_sel ON document_deliveries;
CREATE POLICY deliveries_sel ON document_deliveries USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS deliveries_all ON document_deliveries;
CREATE POLICY deliveries_all ON document_deliveries
  USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

DROP POLICY IF EXISTS announcements_sel ON announcements;
CREATE POLICY announcements_sel ON announcements USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS announcements_all ON announcements;
CREATE POLICY announcements_all ON announcements
  USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- Benfeitores (branch nullable = benfeitor global do tenant)
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS benefactors_sel ON benefactors;
CREATE POLICY benefactors_sel ON benefactors USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS benefactors_all ON benefactors;
CREATE POLICY benefactors_all ON benefactors
  USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- Auditoria: leitura so pela Sede do tenant (ou sistema); escrita livre
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS audit_sel ON audit_log;
CREATE POLICY audit_sel ON audit_log FOR SELECT USING (rls_hq(tenant_id));
DROP POLICY IF EXISTS audit_ins ON audit_log;
CREATE POLICY audit_ins ON audit_log FOR INSERT WITH CHECK (true);
