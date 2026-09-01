-- 000012_document_deliveries.up.sql
-- Fila/outbox de envio de documentos (recibos, carteirinhas) por e-mail/WhatsApp.

CREATE TABLE document_deliveries (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    document_id   uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    channel       text NOT NULL, -- email | whatsapp
    recipient     text NOT NULL,
    status        text NOT NULL DEFAULT 'pending', -- pending | sent | failed
    error         text,
    attempts      int NOT NULL DEFAULT 0,
    sent_at       timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_deliveries_document ON document_deliveries(document_id, created_at);

ALTER TABLE document_deliveries ENABLE ROW LEVEL SECURITY;

CREATE POLICY deliveries_sel ON document_deliveries
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY deliveries_all ON document_deliveries FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL)
  WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);
