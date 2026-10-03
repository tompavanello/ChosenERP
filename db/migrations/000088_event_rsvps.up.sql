-- 000088_event_rsvps.up.sql
-- Confirmacao de presenca (RSVP) do membro a um evento: "Eu vou" / "Talvez" /
-- "Nao vou". Serve de estimativa de publico antes do evento. O check-in real
-- continua em `event_attendance` (chamada nominal).
CREATE TABLE event_rsvps (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    event_id   uuid NOT NULL REFERENCES church_events(id) ON DELETE CASCADE,
    member_id  uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    status     text NOT NULL DEFAULT 'going',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT event_rsvps_status_check CHECK (status IN ('going', 'maybe', 'declined')),
    CONSTRAINT event_rsvps_unique UNIQUE (event_id, member_id)
);

CREATE INDEX idx_event_rsvps_event ON event_rsvps (event_id, status);
CREATE INDEX idx_event_rsvps_member ON event_rsvps (member_id);

ALTER TABLE event_rsvps ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_rsvps FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS event_rsvps_sel ON event_rsvps;
CREATE POLICY event_rsvps_sel ON event_rsvps FOR SELECT
  USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS event_rsvps_all ON event_rsvps;
CREATE POLICY event_rsvps_all ON event_rsvps FOR ALL
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
