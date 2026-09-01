-- db/init/setup.sql
-- Configura os papéis do Chosen ERP no database chosenerp.
-- - migrator (postgres): dono do schema; executa as migrações.
-- - chosenerp_app: papel da aplicação (NÃO é dono nem superuser) => sujeito a RLS.

DO $$ BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='chosenerp_app') THEN
    CREATE ROLE chosenerp_app LOGIN PASSWORD 'chosenapp' NOSUPERUSER NOCREATEDB NOCREATEROLE;
  END IF;
END $$;

GRANT USAGE ON SCHEMA public TO chosenerp_app;

-- Objetos futuros criados pelo migrador (postgres) herdam os privilégios abaixo.
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO chosenerp_app;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT USAGE ON SEQUENCES TO chosenerp_app;
