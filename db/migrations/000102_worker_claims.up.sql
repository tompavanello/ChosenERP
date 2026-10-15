-- 000102_worker_claims.up.sql
-- Claim/lock + backoff para as filas de entrega (documentos e comunicados).
--
-- ListPending fazia SELECT ... WHERE status IN ('pending','failed') sem lock nem
-- transicao de estado. Com duas instancias da API (ou overlap de ticks) o mesmo
-- item era enviado duas vezes (WhatsApp/e-mail duplicado). Adicionamos:
--   * status 'sending' (claim atomico via FOR UPDATE SKIP LOCKED);
--   * locked_at para recuperar itens presos (crash no meio do envio);
--   * next_attempt_at para backoff exponencial quando o envio falha.

ALTER TABLE document_deliveries
    ADD COLUMN locked_at       timestamptz,
    ADD COLUMN next_attempt_at timestamptz;

ALTER TABLE announcement_deliveries
    ADD COLUMN locked_at       timestamptz,
    ADD COLUMN next_attempt_at timestamptz;

CREATE INDEX idx_deliveries_claim
    ON document_deliveries (status, next_attempt_at, created_at);
CREATE INDEX idx_ann_dlv_claim
    ON announcement_deliveries (status, next_attempt_at, created_at);
