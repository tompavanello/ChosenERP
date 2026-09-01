-- infra/postgres/initdb/01-setup.sql
-- Executado na primeira criação do volume do Postgres (empty data dir).
-- Cria o papel da aplicação e privilégios padrão para objetos criados pelo migrador.

DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='chosenerp_app') THEN
    CREATE ROLE chosenerp_app LOGIN PASSWORD 'chosenapp' NOSUPERUSER NOCREATEDB NOCREATEROLE;
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO chosenerp_app;

ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO chosenerp_app;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT USAGE ON SEQUENCES TO chosenerp_app;
