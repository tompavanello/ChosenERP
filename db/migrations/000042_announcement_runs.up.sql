-- 000042_announcement_runs.up.sql
-- Histórico de execuções dos comunicados agendados (tela "Execuções").
-- Cada disparo do ScheduleWorker grava uma linha aqui, com quantos
-- destinatários foram enfileirados naquele período.

CREATE TABLE announcement_runs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid REFERENCES branches(id) ON DELETE SET NULL,
    announcement_id uuid NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    schedule_type   text NOT NULL,
    period_key      text,
    recipient_count int NOT NULL DEFAULT 0,
    fired_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ann_runs_announcement ON announcement_runs(announcement_id, fired_at DESC);
CREATE INDEX idx_ann_runs_tenant ON announcement_runs(tenant_id, fired_at DESC);

ALTER TABLE announcement_runs ENABLE ROW LEVEL SECURITY;
CREATE POLICY announcement_runs_sel ON announcement_runs
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY announcement_runs_all ON announcement_runs
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
