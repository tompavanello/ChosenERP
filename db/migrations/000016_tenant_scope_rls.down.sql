-- 000016_tenant_scope_rls.down.sql
-- Reverte para as politicas anteriores a 000015 (escopo Sede sem checagem de
-- tenant). Mantido apenas para permitir rollback; NAO use em producao, pois
-- restaura o vazamento multi-tenant corrigido por 000016.

CREATE OR REPLACE FUNCTION is_headquarters() RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT current_branch() IS NULL AND current_setting('app.role', true) IN ('super_admin','admin_sede','system')
$$;

-- tenants / branches / roles / users / consent_terms
DROP POLICY IF EXISTS tenant_sel ON tenants;
CREATE POLICY tenant_sel ON tenants USING (id = current_tenant() OR is_headquarters());

DROP POLICY IF EXISTS branch_sel ON branches;
CREATE POLICY branch_sel ON branches USING (tenant_id = current_tenant() OR is_headquarters());
DROP POLICY IF EXISTS branch_all ON branches;
CREATE POLICY branch_all ON branches
  USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

DROP POLICY IF EXISTS roles_sel ON roles;
CREATE POLICY roles_sel ON roles USING (tenant_id = current_tenant());
DROP POLICY IF EXISTS roles_all ON roles;
CREATE POLICY roles_all ON roles FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

DROP POLICY IF EXISTS users_sel ON users;
CREATE POLICY users_sel ON users USING (tenant_id = current_tenant());
DROP POLICY IF EXISTS users_all ON users;
CREATE POLICY users_all ON users FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

DROP POLICY IF EXISTS consent_terms_sel ON consent_terms;
CREATE POLICY consent_terms_sel ON consent_terms USING (tenant_id = current_tenant() OR is_headquarters());
DROP POLICY IF EXISTS consent_terms_all ON consent_terms;
CREATE POLICY consent_terms_all ON consent_terms FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

-- Pessoas
DROP POLICY IF EXISTS members_sel ON members;
CREATE POLICY members_sel ON members USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS members_all ON members;
CREATE POLICY members_all ON members FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS families_sel ON families;
CREATE POLICY families_sel ON families USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS families_all ON families;
CREATE POLICY families_all ON families FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS guests_sel ON visitors;
CREATE POLICY guests_sel ON visitors USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS visitors_all ON visitors;
CREATE POLICY visitors_all ON visitors FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS member_consents_sel ON member_consents;
CREATE POLICY member_consents_sel ON member_consents USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS member_consents_all ON member_consents;
CREATE POLICY member_consents_all ON member_consents FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS rels_sel ON member_relationships;
CREATE POLICY rels_sel ON member_relationships USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND (m.branch_id = current_branch() OR is_headquarters())
));
DROP POLICY IF EXISTS rels_all ON member_relationships;
CREATE POLICY rels_all ON member_relationships FOR ALL USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND m.branch_id = current_branch()
)) WITH CHECK (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND m.branch_id = current_branch()
));

DROP POLICY IF EXISTS ministry_mem_sel ON ministry_members;
CREATE POLICY ministry_mem_sel ON ministry_members USING (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND (m.branch_id = current_branch() OR is_headquarters())
));
DROP POLICY IF EXISTS ministry_mem_all ON ministry_members;
CREATE POLICY ministry_mem_all ON ministry_members FOR ALL USING (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND (m.branch_id = current_branch())
)) WITH CHECK (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND m.branch_id = current_branch()
));

DROP POLICY IF EXISTS ministries_sel ON ministries;
CREATE POLICY ministries_sel ON ministries USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS ministries_all ON ministries;
CREATE POLICY ministries_all ON ministries FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS groups_sel ON small_groups;
CREATE POLICY groups_sel ON small_groups USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS groups_all ON small_groups;
CREATE POLICY groups_all ON small_groups FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS att_sel ON group_attendance;
CREATE POLICY att_sel ON group_attendance USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS att_all ON group_attendance;
CREATE POLICY att_all ON group_attendance FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

-- Financeiro
DROP POLICY IF EXISTS fincat_sel ON financial_categories;
CREATE POLICY fincat_sel ON financial_categories USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
DROP POLICY IF EXISTS fincat_all ON financial_categories;
CREATE POLICY fincat_all ON financial_categories FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

DROP POLICY IF EXISTS fintx_sel ON financial_transactions;
CREATE POLICY fintx_sel ON financial_transactions USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS fintx_all ON financial_transactions;
CREATE POLICY fintx_all ON financial_transactions FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS recurring_sel ON recurring_donations;
CREATE POLICY recurring_sel ON recurring_donations FOR SELECT USING (branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS recurring_ins ON recurring_donations;
CREATE POLICY recurring_ins ON recurring_donations FOR INSERT WITH CHECK (branch_id = current_branch());
DROP POLICY IF EXISTS recurring_upd ON recurring_donations;
CREATE POLICY recurring_upd ON recurring_donations FOR UPDATE USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

DROP POLICY IF EXISTS transfers_sel ON transfers;
CREATE POLICY transfers_sel ON transfers USING (from_branch_id = current_branch() OR to_branch_id = current_branch() OR is_headquarters());
DROP POLICY IF EXISTS transfers_ins ON transfers;
CREATE POLICY transfers_ins ON transfers FOR INSERT WITH CHECK (
    from_branch_id = current_branch() OR
    to_branch_id   = current_branch() OR
    is_headquarters()
);

-- Documentos / avisos / entregas / benfeitores
DROP POLICY IF EXISTS docs_sel ON documents;
CREATE POLICY docs_sel ON documents USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
DROP POLICY IF EXISTS docs_all ON documents;
CREATE POLICY docs_all ON documents FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

DROP POLICY IF EXISTS deliveries_sel ON document_deliveries;
CREATE POLICY deliveries_sel ON document_deliveries
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
DROP POLICY IF EXISTS deliveries_all ON document_deliveries;
CREATE POLICY deliveries_all ON document_deliveries FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

DROP POLICY IF EXISTS announcements_sel ON announcements;
CREATE POLICY announcements_sel ON announcements
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
DROP POLICY IF EXISTS announcements_all ON announcements;
CREATE POLICY announcements_all ON announcements FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

DROP POLICY IF EXISTS benefactors_sel ON benefactors;
CREATE POLICY benefactors_sel ON benefactors USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
DROP POLICY IF EXISTS benefactors_all ON benefactors;
CREATE POLICY benefactors_all ON benefactors FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

-- Auditoria
DROP POLICY IF EXISTS audit_sel ON audit_log;
CREATE POLICY audit_sel ON audit_log FOR SELECT USING (is_headquarters());
DROP POLICY IF EXISTS audit_ins ON audit_log;
CREATE POLICY audit_ins ON audit_log FOR INSERT WITH CHECK (true);

-- Remove os helpers introduzidos por 000016.
DROP FUNCTION IF EXISTS rls_hq(uuid);
DROP FUNCTION IF EXISTS rls_write(uuid, uuid, boolean);
DROP FUNCTION IF EXISTS rls_read(uuid, uuid, boolean);
DROP FUNCTION IF EXISTS rls_branch_match(uuid, boolean);
DROP FUNCTION IF EXISTS is_system();
