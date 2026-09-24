-- 000057_branch_whatsapp_number.up.sql
-- Numero efetivamente CONECTADO na instancia Evolution da filial (ownerJid).
-- Distinto de whatsapp_phone (o telefone que o admin associa manualmente): este
-- e lido da Evolution quando a conexao abre e persistido para aparecer no grid.
ALTER TABLE branches ADD COLUMN whatsapp_number text;
