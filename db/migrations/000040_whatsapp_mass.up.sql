-- 000040_whatsapp_mass.up.sql
-- WhatsApp em massa (Fase 2 / itens #31 e #32):
--   * #32 disparo segmentado: os filtros demográficos viajam no SendInput e são
--     resolvidos em internal/announcements (não exigem coluna nova).
--   * #31 automações (aniversário, lembrete de escala, boas-vindas ao visitante):
--     precisam de um outbox que aceite mensagens SEM `announcements` (o texto é
--     renderizado por destinatário) e de configuração por tenant.
--
-- Por isso a fila de envio ganha snapshot de título/corpo, origem e uma chave de
-- deduplicação (evita mandar o mesmo "feliz aniversário" duas vezes no dia).

-- ---------------------------------------------------------------------------
-- 1) Outbox: mensagens avulsas (automações) + snapshot + dedupe
-- ---------------------------------------------------------------------------
ALTER TABLE announcement_deliveries
    ADD COLUMN title          text,
    ADD COLUMN body           text,
    ADD COLUMN recipient_name text,
    ADD COLUMN source         text NOT NULL DEFAULT 'manual',
    ADD COLUMN dedupe_key     text;

ALTER TABLE announcement_deliveries
    ADD CONSTRAINT announcement_deliveries_source_check
    CHECK (source IN ('manual','birthday','roster','visitor_welcome'));

-- Uma automação pode não estar atrelada a um aviso do feed.
ALTER TABLE announcement_deliveries ALTER COLUMN announcement_id DROP NOT NULL;

-- Dedupe por tenant: o worker usa ON CONFLICT nesta chave para não repetir.
CREATE UNIQUE INDEX idx_ann_dlv_dedupe
    ON announcement_deliveries (tenant_id, dedupe_key)
    WHERE dedupe_key IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 2) Corrige a RLS do outbox: as políticas originais (000017) comparavam o
--    branch sem checar o tenant e `branch_id IS NULL` vazava entre tenants.
--    Passa a usar os helpers rls_read/rls_write (branch nullable = global).
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS ann_dlv_sel ON announcement_deliveries;
CREATE POLICY ann_dlv_sel ON announcement_deliveries
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));
DROP POLICY IF EXISTS ann_dlv_all ON announcement_deliveries;
CREATE POLICY ann_dlv_all ON announcement_deliveries
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- 3) Configuração das automações (uma linha por tenant)
-- ---------------------------------------------------------------------------
CREATE TABLE notification_settings (
    tenant_id                  uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    birthdays_enabled          boolean NOT NULL DEFAULT false,
    birthday_template          text NOT NULL DEFAULT 'Feliz aniversário, {primeiro_nome}! A {igreja} celebra a sua vida hoje. Que Deus o(a) abençoe ricamente!',
    roster_reminders_enabled   boolean NOT NULL DEFAULT false,
    roster_reminder_hours      int NOT NULL DEFAULT 24
        CHECK (roster_reminder_hours BETWEEN 1 AND 168),
    roster_template            text NOT NULL DEFAULT 'Olá, {primeiro_nome}! Lembrete: você está escalado(a) em "{titulo}" no dia {data}. Conte com você!',
    visitor_welcome_enabled    boolean NOT NULL DEFAULT false,
    visitor_welcome_delay_hours int NOT NULL DEFAULT 24
        CHECK (visitor_welcome_delay_hours BETWEEN 1 AND 720),
    visitor_welcome_template   text NOT NULL DEFAULT 'Olá, {primeiro_nome}! Foi uma alegria receber você na {igreja}. Esperamos ver você novamente em breve!',
    updated_at                 timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE notification_settings ENABLE ROW LEVEL SECURITY;
CREATE POLICY notification_settings_all ON notification_settings
  FOR ALL USING (rls_hq(tenant_id)) WITH CHECK (rls_hq(tenant_id));
