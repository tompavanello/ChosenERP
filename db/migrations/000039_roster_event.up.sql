-- 000039_roster_event.up.sql
-- Escala ligada a evento OU a tipo de evento, com geracao automatica de evento.
--   * event_kind_id: tipo de evento escolhido quando NAO se vincula um evento
--     concreto (a escala pode ser "somente escala" ou "escala de um tipo").
--   * generated_event: marca que o evento vinculado foi CRIADO pela escala
--     (grade de eventos + convocados/responsabilidades sincronizados a partir
--     dos escalados). Sem a marca, o evento e externo e a escala nao o altera.

ALTER TABLE rosters
    ADD COLUMN event_kind_id uuid REFERENCES event_kinds(id) ON DELETE SET NULL,
    ADD COLUMN generated_event boolean NOT NULL DEFAULT false;

CREATE INDEX idx_rosters_event_kind ON rosters(event_kind_id);
