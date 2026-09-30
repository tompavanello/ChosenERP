-- 000068_cultos.down.sql
-- Remove o vinculo com a agenda e o modulo de Cultos. Os eventos ja publicados
-- permanecem (apenas perdem a referencia ao culto).
ALTER TABLE church_events DROP COLUMN IF EXISTS culto_id;
DROP TABLE IF EXISTS cultos;
