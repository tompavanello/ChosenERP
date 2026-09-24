-- 000041_announcement_scheduling.up.sql
-- Comunicados agendados + segmentacao salva (Fase 2, evolucao de #31/#32).
--
-- Ate aqui o disparo era sempre manual e a segmentacao era escolhida na hora.
-- Agora o comunicado guarda a segmentacao e um agendamento:
--   * manual  -> so dispara pelo botao "Enviar";
--   * once    -> dispara uma vez em `schedule_at`;
--   * daily   -> dispara todo dia em `schedule_time` (fuso do tenant);
--   * event   -> dispara `schedule_offset_minutes` minutos antes (negativo) ou
--                depois (positivo) do inicio do evento `schedule_event_id`.
--
-- `last_run_at` evita repeticao e `run_count` da visibilidade.

ALTER TABLE announcements
    ADD COLUMN audience_filter jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN channel text NOT NULL DEFAULT 'whatsapp',
    ADD COLUMN schedule_type text NOT NULL DEFAULT 'manual',
    ADD COLUMN schedule_at timestamptz,
    ADD COLUMN schedule_time time,
    ADD COLUMN schedule_event_id uuid REFERENCES church_events(id) ON DELETE SET NULL,
    ADD COLUMN schedule_offset_minutes int NOT NULL DEFAULT 0,
    ADD COLUMN last_run_at timestamptz,
    ADD COLUMN run_count int NOT NULL DEFAULT 0;

ALTER TABLE announcements
    ADD CONSTRAINT announcements_schedule_type_check
    CHECK (schedule_type IN ('manual','once','daily','event'));

ALTER TABLE announcements
    ADD CONSTRAINT announcements_channel_check
    CHECK (channel IN ('whatsapp','email'));

-- Agendamento coerente com o tipo escolhido.
ALTER TABLE announcements
    ADD CONSTRAINT announcements_schedule_fields_check
    CHECK (
        (schedule_type = 'manual') OR
        (schedule_type = 'once'  AND schedule_at IS NOT NULL) OR
        (schedule_type = 'daily' AND schedule_time IS NOT NULL) OR
        (schedule_type = 'event' AND schedule_event_id IS NOT NULL)
    );

-- O scheduler varre so os agendados.
CREATE INDEX idx_announcements_scheduled
    ON announcements(schedule_type, last_run_at)
    WHERE schedule_type <> 'manual';

-- Nova origem na outbox (disparo por agendamento).
ALTER TABLE announcement_deliveries
    DROP CONSTRAINT announcement_deliveries_source_check;
ALTER TABLE announcement_deliveries
    ADD CONSTRAINT announcement_deliveries_source_check
    CHECK (source IN ('manual','birthday','roster','visitor_welcome','schedule'));
