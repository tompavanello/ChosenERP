-- 000001_bootstrap.up.sql
-- Extensoes e helpers globais do Chosen ERP

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;

-- UUID v4 padrao
CREATE OR REPLACE FUNCTION uuid_generate() RETURNS uuid
LANGUAGE sql AS $$ SELECT gen_random_uuid() $$;

-- Controla se a session pode "ver tudo" (perfil Sede / sistema)
-- So e chamavel por funcoes security definer ou por sessoes com permissao.
COMMENT ON FUNCTION uuid_generate() IS 'Gera UUID v4 via gen_random_uuid() (core postgres).';
