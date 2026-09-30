-- 000070_programacao_title.down.sql
ALTER TABLE church_events DROP COLUMN IF EXISTS title;
ALTER TABLE programacoes DROP COLUMN IF EXISTS title;
