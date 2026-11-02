-- 000117_agenda_planner.up.sql
-- Agenda unica + planejamento por dominio.
--
-- A ocorrencia continua em church_events (calendario, chamada, RSVP). A serie
-- recorrente ganha dono (ministerio), excecoes (cancelada/remarcada) e a data
-- da ocorrencia, para que publicar a grade de novo nao recrie/duplique o que foi
-- ajustado no calendario. Os demais modulos continuam vinculados por event_id.

ALTER TABLE programacoes ADD COLUMN IF NOT EXISTS owner_ministry_id uuid
    REFERENCES ministries(id) ON DELETE SET NULL;

ALTER TABLE church_events ADD COLUMN IF NOT EXISTS owner_ministry_id uuid
    REFERENCES ministries(id) ON DELETE SET NULL;
ALTER TABLE church_events ADD COLUMN IF NOT EXISTS status text NOT NULL DEFAULT 'planned';
ALTER TABLE church_events ADD COLUMN IF NOT EXISTS occurrence_date date;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'church_events_status_check'
    ) THEN
        ALTER TABLE church_events ADD CONSTRAINT church_events_status_check
            CHECK (status IN ('planned','confirmed','cancelled','done'));
    END IF;
END $$;

-- Data local da ocorrencia (no fuso da igreja). Sem isso, mover uma ocorrencia
-- no calendario deixaria a serie sem referencia para registrar a excecao.
UPDATE church_events e
SET occurrence_date = (
    e.starts_at AT TIME ZONE COALESCE(NULLIF(t.timezone,''),'America/Sao_Paulo')
)::date
FROM tenants t
WHERE t.id = e.tenant_id
  AND e.occurrence_date IS NULL
  AND e.origin IN ('programacao','escala');

CREATE UNIQUE INDEX IF NOT EXISTS idx_church_events_series_occurrence
    ON church_events (tenant_id, origin_id, occurrence_date)
    WHERE origin = 'programacao' AND origin_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_church_events_owner_ministry
    ON church_events (tenant_id, owner_ministry_id, starts_at);

CREATE TABLE IF NOT EXISTS programacao_exceptions (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id        uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    programacao_id   uuid NOT NULL REFERENCES programacoes(id) ON DELETE CASCADE,
    occurrence_date  date NOT NULL,
    action           text NOT NULL DEFAULT 'move',
    new_starts_at    timestamptz,
    new_ends_at      timestamptz,
    note             text,
    created_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT programacao_exceptions_action_check CHECK (action IN ('cancel','move')),
    CONSTRAINT programacao_exceptions_move_check CHECK (
        action <> 'move' OR new_starts_at IS NOT NULL
    ),
    UNIQUE (programacao_id, occurrence_date)
);

CREATE INDEX IF NOT EXISTS idx_programacao_exceptions_tenant
    ON programacao_exceptions (tenant_id, branch_id, occurrence_date);

ALTER TABLE programacao_exceptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY programacao_exceptions_sel ON programacao_exceptions
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY programacao_exceptions_ins ON programacao_exceptions
    FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY programacao_exceptions_upd ON programacao_exceptions
    FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY programacao_exceptions_del ON programacao_exceptions
    FOR DELETE USING (rls_write(tenant_id, branch_id, false));
