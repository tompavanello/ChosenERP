-- 000001_bootstrap.down.sql
DROP EXTENSION IF EXISTS "uuid-ossp";
DROP EXTENSION IF EXISTS pgcrypto;
DROP FUNCTION IF EXISTS uuid_generate();
