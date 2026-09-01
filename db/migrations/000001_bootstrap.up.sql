-- 000001_bootstrap.up.sql
-- Extensões e helpers globais do Chosen ERP

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;

-- UUID v4 padrão
CREATE OR REPLACE FUNCTION uuid_generate() RETURNS uuid
LANGUAGE sql AS $$ SELECT gen_random_uuid() $$;

-- Controla se a session pode "ver tudo" (perfil Sede / sistema)
-- Só é chamável por funções security definer ou por sessões com permissão.
COMMENT ON FUNCTION uuid_generate() IS 'Gera UUID v4 via gen_random_uuid() (core postgres).';
