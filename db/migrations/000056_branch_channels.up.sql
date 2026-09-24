-- 000056_branch_channels.up.sql
-- Canais de envio POR FILIAL (white-label de comunicação):
--   * WhatsApp via Evolution API — cada filial tem a própria instância, nomeada
--     com o id da filial, conectada por QR Code.
--   * SMTP de e-mail próprio da filial.
--
-- As credenciais ficam em `branches` (RLS já cobre a tabela). A senha do SMTP
-- nunca é devolvida pela API — só um flag `smtp_password_set`.
ALTER TABLE branches
    ADD COLUMN whatsapp_phone    text,
    ADD COLUMN whatsapp_instance text,
    ADD COLUMN whatsapp_status   text NOT NULL DEFAULT 'disconnected',
    ADD COLUMN smtp_host         text,
    ADD COLUMN smtp_port         int,
    ADD COLUMN smtp_user         text,
    ADD COLUMN smtp_password     text,
    ADD COLUMN smtp_from         text,
    ADD COLUMN smtp_from_name    text,
    ADD COLUMN smtp_secure       boolean NOT NULL DEFAULT false;

ALTER TABLE branches
    ADD CONSTRAINT branches_whatsapp_status_check
    CHECK (whatsapp_status IN ('disconnected', 'connecting', 'connected'));
