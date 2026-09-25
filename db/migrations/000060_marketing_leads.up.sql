-- 000060_marketing_leads.up.sql
-- Leads do site institucional (formulario "solicitar demonstracao").
--
-- Nao e dado de igreja: a tabela NAO tem tenant_id/branch_id e NAO usa RLS.
-- A API grava com o papel chosenerp_app, que recebe apenas INSERT (sem SELECT),
-- para nao vazar a lista de contatos. A leitura fica para o super_admin/console.

CREATE TABLE IF NOT EXISTS marketing_leads (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text        NOT NULL,
    email       text        NOT NULL,
    phone       text,
    church_name text,
    church_size text,
    message     text,
    source      text        NOT NULL DEFAULT 'site',
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS marketing_leads_created_at_idx
    ON marketing_leads (created_at DESC);

-- db/init/setup.sql concede SELECT/INSERT/UPDATE/DELETE por padrao a
-- chosenerp_app (ALTER DEFAULT PRIVILEGES). Aqui reduzimos a superficie ao
-- necessario: o site apenas INSERE. Sem SELECT, uma eventual exposicao do
-- papel da aplicacao nao varre a lista de leads.
REVOKE ALL ON marketing_leads FROM chosenerp_app;
GRANT INSERT ON marketing_leads TO chosenerp_app;
