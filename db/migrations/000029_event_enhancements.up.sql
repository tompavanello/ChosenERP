-- 000029_event_enhancements.up.sql
-- Melhorias no registro de eventos:
--   * attendance_mode: como a presença é registrada (chamada nominal OU total digitado);
--   * estimated_cost: custo estimado (opcional);
--   * event_invitees: pessoas e/ou ministérios CONVOCADOS (obrigados a participar);
--   * multi-dia já é suportado por starts_at/ends_at (retiro, acampamento) — o
--     frontend passou a ter data de término.

ALTER TABLE church_events
    ADD COLUMN attendance_mode text NOT NULL DEFAULT 'nominal'
        CHECK (attendance_mode IN ('nominal','count')),
    ADD COLUMN estimated_cost numeric(14,2)
        CHECK (estimated_cost IS NULL OR estimated_cost >= 0);

-- ---------------------------------------------------------------------------
-- Convocados (pessoas e/ou ministérios)
-- ---------------------------------------------------------------------------
CREATE TABLE event_invitees (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    event_id    uuid NOT NULL REFERENCES church_events(id) ON DELETE CASCADE,
    member_id   uuid REFERENCES members(id) ON DELETE CASCADE,
    ministry_id uuid REFERENCES ministries(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    -- Exatamente um dos dois: pessoa OU ministério.
    CONSTRAINT event_invitees_target_check CHECK (
        (member_id IS NOT NULL AND ministry_id IS NULL) OR
        (member_id IS NULL AND ministry_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_event_invitee_member ON event_invitees(event_id, member_id) WHERE member_id IS NOT NULL;
CREATE UNIQUE INDEX uq_event_invitee_ministry ON event_invitees(event_id, ministry_id) WHERE ministry_id IS NOT NULL;
CREATE INDEX idx_event_invitees_event ON event_invitees(event_id);

ALTER TABLE event_invitees ENABLE ROW LEVEL SECURITY;
CREATE POLICY event_invitees_sel ON event_invitees
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY event_invitees_ins ON event_invitees
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY event_invitees_upd ON event_invitees
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY event_invitees_del ON event_invitees
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));
