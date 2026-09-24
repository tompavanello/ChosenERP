-- 000017_announcement_deliveries.up.sql
-- Fila/outbox de envio de comunicados por WhatsApp (e futuro e-mail).
-- Cada envio real e feito pelo worker de anuncios, operando dentro do
-- contexto RLS de cada filial/tenant para preservar o isolamento multi-tenant.

CREATE TABLE announcement_deliveries (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    announcement_id uuid NOT NULL REFERENCES announcements(id) ON DELETE CASCADE,
    channel       text NOT NULL, -- whatsapp | email
    recipient     text NOT NULL, -- telefone no formato E.164 para whatsapp
    provider      text NOT NULL DEFAULT 'evolution', -- evolution | meta
    provider_message_id text, -- id retornado pelo provedor (para deduplicar/status)
    status        text NOT NULL DEFAULT 'pending', -- pending | sent | failed
    error         text,
    attempts      int NOT NULL DEFAULT 0,
    sent_at       timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ann_dlv_announcement ON announcement_deliveries(announcement_id, created_at);
CREATE INDEX idx_ann_dlv_pending ON announcement_deliveries(status, attempts)
    WHERE status IN ('pending','failed');

ALTER TABLE announcement_deliveries ENABLE ROW LEVEL SECURITY;

CREATE POLICY ann_dlv_sel ON announcement_deliveries FOR SELECT
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY ann_dlv_all ON announcement_deliveries FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL)
  WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);
