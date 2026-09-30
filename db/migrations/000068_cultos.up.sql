-- 000068_cultos.up.sql
-- Modulo de Cultos: a GRADE de horarios recorrentes dos cultos da igreja.
--
-- Um "culto" descreve um horario fixo (dia da semana + hora + duracao + tipo de
-- evento + local). A partir dessa grade, a igreja PUBLICA as ocorrencias na
-- agenda de eventos (`church_events`): cada ocorrencia vira um evento normal,
-- com `culto_id` apontando para a definicao de origem. Assim a agenda, a
-- chamada nominal e os convocados do modulo de Eventos seguem valendo para os
-- cultos, sem duplicar dado.
--
-- Direcao do vinculo: `church_events.culto_id` (1 culto -> N eventos), ao
-- contrario de `rosters.event_id` (1 escala -> 1 evento). Excluir o culto nao
-- apaga a agenda ja publicada (ON DELETE SET NULL).

CREATE TABLE cultos (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id        uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    name             text NOT NULL,
    -- Tipo de evento usado ao publicar na agenda (opcional).
    event_kind_id    uuid REFERENCES event_kinds(id) ON DELETE SET NULL,
    -- 0=domingo .. 6=sabado (mesma convencao de EXTRACT(DOW) no Postgres).
    weekday          int NOT NULL CHECK (weekday >= 0 AND weekday <= 6),
    start_time       time NOT NULL,
    duration_minutes int NOT NULL DEFAULT 90 CHECK (duration_minutes > 0 AND duration_minutes <= 1440),
    location         text,
    notes            text,
    is_active        boolean NOT NULL DEFAULT true,
    sort_order       int NOT NULL DEFAULT 0,
    created_by       uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_cultos_branch ON cultos(tenant_id, branch_id, weekday, start_time);
CREATE INDEX idx_cultos_kind ON cultos(event_kind_id);

ALTER TABLE cultos ENABLE ROW LEVEL SECURITY;
CREATE POLICY cultos_sel ON cultos
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY cultos_ins ON cultos
  FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY cultos_upd ON cultos
  FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
CREATE POLICY cultos_del ON cultos
  FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- Vinculo com a agenda de eventos
-- ---------------------------------------------------------------------------
ALTER TABLE church_events
    ADD COLUMN culto_id uuid REFERENCES cultos(id) ON DELETE SET NULL;

CREATE INDEX idx_church_events_culto ON church_events(culto_id);
