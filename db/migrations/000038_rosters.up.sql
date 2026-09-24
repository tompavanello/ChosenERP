-- 000038_rosters.up.sql
-- Escalas de voluntarios (Fase 2 / itens #24-#26):
--   * rosters: uma escala (culto/evento/ministerio) em um periodo;
--   * roster_assignments: quem foi escalado, com funcao e situacao
--     (convidado / confirmado / recusado) - e a "confirmacao de presenca".
--
-- Conflito de agenda (#25) e sugestao inteligente (#26) sao calculados em
-- internal/rosters a partir destas tabelas (nao ha coluna extra). O isolamento
-- e do RLS.

CREATE TABLE rosters (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    ministry_id uuid REFERENCES ministries(id) ON DELETE SET NULL,
    event_id    uuid REFERENCES church_events(id) ON DELETE SET NULL,
    title       text NOT NULL,
    starts_at   timestamptz NOT NULL,
    ends_at     timestamptz,
    location    text,
    notes       text,
    status      text NOT NULL DEFAULT 'rascunho',
    created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT rosters_status_check CHECK (status IN ('rascunho','publicada','concluida','cancelada')),
    CONSTRAINT rosters_period_check CHECK (ends_at IS NULL OR ends_at >= starts_at)
);

CREATE INDEX idx_rosters_branch ON rosters(tenant_id, branch_id, starts_at DESC);
CREATE INDEX idx_rosters_ministry ON rosters(ministry_id);
CREATE INDEX idx_rosters_event ON rosters(event_id);

ALTER TABLE rosters ENABLE ROW LEVEL SECURITY;
CREATE POLICY rosters_sel ON rosters
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY rosters_ins ON rosters
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY rosters_upd ON rosters
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY rosters_del ON rosters
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

CREATE TABLE roster_assignments (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    roster_id    uuid NOT NULL REFERENCES rosters(id) ON DELETE CASCADE,
    member_id    uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    role         text, -- funcao na escala (vocal, guitarra, diaconia...)
    status       text NOT NULL DEFAULT 'convidado',
    responded_at timestamptz,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT roster_assignments_status_check CHECK (status IN ('convidado','confirmado','recusado')),
    UNIQUE (roster_id, member_id)
);

CREATE INDEX idx_roster_assignments_roster ON roster_assignments(roster_id);
CREATE INDEX idx_roster_assignments_member ON roster_assignments(member_id);

ALTER TABLE roster_assignments ENABLE ROW LEVEL SECURITY;
CREATE POLICY roster_assignments_sel ON roster_assignments
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY roster_assignments_ins ON roster_assignments
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY roster_assignments_upd ON roster_assignments
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY roster_assignments_del ON roster_assignments
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));
