-- 000028_church_events.up.sql
-- Registro de Eventos e Frequencia (requisitos C8-C10 do cliente):
--   * event_kinds: catalogo de tipos de evento, customizavel (padrao cargos);
--   * church_events: evento com data/hora, tipo, total de participantes e notas;
--   * event_attendance: chamada nominal por pessoa (as DUAS opcoes: total
--     digitado E presenca por membro);
--   * member_frequency_history: frequencia do membro com historico (nunca
--     sobrescreve a anterior; a linha com ended_at IS NULL e a vigente).

-- ---------------------------------------------------------------------------
-- Catalogo de tipos de evento
-- ---------------------------------------------------------------------------
CREATE TABLE event_kinds (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    -- NULL = tipo global do tenant (visivel a todas as filiais).
    branch_id  uuid REFERENCES branches(id) ON DELETE SET NULL,
    name       text NOT NULL,
    slug       text NOT NULL,
    sort_order int NOT NULL DEFAULT 0,
    is_active  boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_event_kinds_tenant_slug ON event_kinds(tenant_id, slug);
CREATE INDEX idx_event_kinds_tenant ON event_kinds(tenant_id);

ALTER TABLE event_kinds ENABLE ROW LEVEL SECURITY;
CREATE POLICY event_kinds_sel ON event_kinds
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY event_kinds_ins ON event_kinds
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY event_kinds_upd ON event_kinds
  FOR UPDATE USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
CREATE POLICY event_kinds_del ON event_kinds
  FOR DELETE USING (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- Eventos
-- ---------------------------------------------------------------------------
CREATE TABLE church_events (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id          uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    kind_id            uuid REFERENCES event_kinds(id) ON DELETE SET NULL,
    starts_at          timestamptz NOT NULL,
    ends_at            timestamptz,
    participants_count int NOT NULL DEFAULT 0 CHECK (participants_count >= 0),
    notes              text,
    created_by         uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT church_events_period_check CHECK (ends_at IS NULL OR ends_at >= starts_at)
);

CREATE INDEX idx_church_events_branch ON church_events(tenant_id, branch_id, starts_at DESC);
CREATE INDEX idx_church_events_kind ON church_events(kind_id);

ALTER TABLE church_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY church_events_sel ON church_events
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY church_events_ins ON church_events
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY church_events_upd ON church_events
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY church_events_del ON church_events
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Chamada nominal (presenca por membro)
-- ---------------------------------------------------------------------------
CREATE TABLE event_attendance (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    event_id   uuid NOT NULL REFERENCES church_events(id) ON DELETE CASCADE,
    member_id  uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    present    boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, member_id)
);

CREATE INDEX idx_event_attendance_event ON event_attendance(event_id);
CREATE INDEX idx_event_attendance_member ON event_attendance(member_id);

ALTER TABLE event_attendance ENABLE ROW LEVEL SECURITY;
CREATE POLICY event_attendance_sel ON event_attendance
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY event_attendance_ins ON event_attendance
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY event_attendance_upd ON event_attendance
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY event_attendance_del ON event_attendance
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Historico de frequencia do membro (requisito 1.5)
-- ---------------------------------------------------------------------------
CREATE TABLE member_frequency_history (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    member_id  uuid NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    frequency  text NOT NULL,
    started_at date NOT NULL DEFAULT current_date,
    ended_at   date,
    notes      text,
    created_by uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT member_frequency_check CHECK (frequency IN ('frequente','pouco_frequente','nao_frequente'))
);

CREATE INDEX idx_member_freq_member ON member_frequency_history(member_id, started_at DESC);
CREATE INDEX idx_member_freq_tenant ON member_frequency_history(tenant_id, branch_id);

ALTER TABLE member_frequency_history ENABLE ROW LEVEL SECURITY;
CREATE POLICY member_freq_sel ON member_frequency_history
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY member_freq_all ON member_frequency_history
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Catalogo inicial de tipos (requisito C9)
-- ---------------------------------------------------------------------------
INSERT INTO event_kinds (tenant_id, branch_id, name, slug, sort_order)
SELECT t.id, NULL, v.name, v.slug, v.sort_order
FROM tenants t
CROSS JOIN (VALUES
    ('Escola Biblica Dominical', 'escola_biblica_dominical', 10),
    ('Culto',                    'culto',                    20),
    ('Culto Especial',           'culto_especial',           30),
    ('Reuniao',                  'reuniao',                  40),
    ('Pequeno Grupo',            'pequeno_grupo',            50),
    ('Evento de Jovens',         'evento_jovens',            60),
    ('Escola Biblica',           'escola_biblica',           70),
    ('Santa Ceia',               'santa_ceia',               80),
    ('Vigilia',                  'vigilia',                  90),
    ('Outros',                   'outros',                   999)
) AS v(name, slug, sort_order)
ON CONFLICT (tenant_id, slug) DO NOTHING;
