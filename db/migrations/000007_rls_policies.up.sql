-- 000007_rls_policies.up.sql
-- Row-Level Security: isolamento multi-tenant por branch_id / tenant_id
-- Sessao do gateway seta: app.tenant_id, app.branch_id, app.role

-- ---------------------------------------------------------------
-- Helper de policy: usuario tem acesso ao branch ou e escopo "Sede"
-- Sede => app.branch_id = '00000000-0000-0000-0000-000000000000' ou NULL
-- ---------------------------------------------------------------
CREATE OR REPLACE FUNCTION current_branch() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.branch_id', true), '')::uuid
$$;

CREATE OR REPLACE FUNCTION current_tenant() RETURNS uuid
LANGUAGE sql STABLE AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')::uuid
$$;

CREATE OR REPLACE FUNCTION is_headquarters() RETURNS boolean
LANGUAGE sql STABLE AS $$
    SELECT current_branch() IS NULL AND current_setting('app.role', true) IN ('super_admin','admin_sede','system')
$$;

-- ---------------------------------------------------------------
-- HABILITAR RLS nas tabelas tenant-scoped
-- ---------------------------------------------------------------
ALTER TABLE tenants                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE branches                ENABLE ROW LEVEL SECURITY;
ALTER TABLE roles                   ENABLE ROW LEVEL SECURITY;
ALTER TABLE users                   ENABLE ROW LEVEL SECURITY;
ALTER TABLE members                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE families                ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_relationships   ENABLE ROW LEVEL SECURITY;
ALTER TABLE visitors                ENABLE ROW LEVEL SECURITY;
ALTER TABLE benefactors            ENABLE ROW LEVEL SECURITY;
ALTER TABLE ministries             ENABLE ROW LEVEL SECURITY;
ALTER TABLE ministry_members       ENABLE ROW LEVEL SECURITY;
ALTER TABLE small_groups           ENABLE ROW LEVEL SECURITY;
ALTER TABLE group_attendance       ENABLE ROW LEVEL SECURITY;
ALTER TABLE financial_categories   ENABLE ROW LEVEL SECURITY;
ALTER TABLE financial_transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE transfers              ENABLE ROW LEVEL SECURITY;
ALTER TABLE documents              ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log              ENABLE ROW LEVEL SECURITY;

-- ---------------------------------------------------------------
-- POLITICAS (SELECT / INSERT para escopo; tabelas don't need UPDATE/DELETE policies except none)
-- ---------------------------------------------------------------

-- tenants: sede global ou "sistema"
CREATE POLICY tenant_sel ON tenants USING (id = current_tenant() OR is_headquarters());

-- branches: tenant do contexto
CREATE POLICY branch_sel ON branches USING (tenant_id = current_tenant() OR is_headquarters());
CREATE POLICY branch_all ON branches
  FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

-- roles & users
CREATE POLICY roles_sel ON roles USING (tenant_id = current_tenant());
CREATE POLICY roles_all ON roles FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());
CREATE POLICY users_sel ON users USING (tenant_id = current_tenant());
CREATE POLICY users_all ON users FOR ALL USING (tenant_id = current_tenant()) WITH CHECK (tenant_id = current_tenant());

-- People-scoped por branch
CREATE POLICY members_sel ON members USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY members_all ON members FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY families_sel ON families USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY families_all ON families FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY rels_sel ON member_relationships USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND (m.branch_id = current_branch() OR is_headquarters())
));
CREATE POLICY rels_all ON member_relationships FOR ALL USING (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND m.branch_id = current_branch()
)) WITH CHECK (EXISTS (
    SELECT 1 FROM members m WHERE m.id = member_id AND m.branch_id = current_branch()
));
CREATE POLICY guests_sel ON visitors USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY visitors_all ON visitors FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY benefactors_sel ON benefactors USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY benefactors_all ON benefactors FOR ALL USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

CREATE POLICY ministries_sel ON ministries USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY ministries_all ON ministries FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY ministry_mem_sel ON ministry_members USING (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND (m.branch_id = current_branch() OR is_headquarters())
));
CREATE POLICY ministry_mem_all ON ministry_members FOR ALL USING (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND (m.branch_id = current_branch())
)) WITH CHECK (EXISTS (
    SELECT 1 FROM ministries m WHERE m.id = ministry_id AND m.branch_id = current_branch()
));

CREATE POLICY groups_sel ON small_groups USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY groups_all ON small_groups FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY att_sel ON group_attendance USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY att_all ON group_attendance FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

-- Finance
CREATE POLICY fincat_sel ON financial_categories USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY fincat_all ON financial_categories FOR ALL USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

CREATE POLICY fintx_sel ON financial_transactions USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY fintx_all ON financial_transactions FOR ALL USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());

CREATE POLICY transfers_sel ON transfers USING (from_branch_id = current_branch() OR to_branch_id = current_branch() OR is_headquarters());

CREATE POLICY docs_sel ON documents USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY docs_all ON documents FOR ALL USING (branch_id = current_branch() OR branch_id IS NULL) WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);

-- Audit log: sistema escreve (service role), Sede/sistema le
CREATE POLICY audit_sel ON audit_log FOR SELECT USING (is_headquarters());
CREATE POLICY audit_ins ON audit_log FOR INSERT WITH CHECK (true);
