-- 000015_visitors_whatsapp.up.sql
-- WhatsApp do visitante (necessário para as boas-vindas automatizadas).

ALTER TABLE visitors ADD COLUMN IF NOT EXISTS whatsapp text;
