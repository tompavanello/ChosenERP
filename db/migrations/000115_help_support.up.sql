-- 000115_help_support.up.sql
-- Central de ajuda: chamados de suporte e feedback dos artigos.
--
-- O chamado nasce dentro do ERP (inbox da plataforma). A notificacao paralela
-- por e-mail/WhatsApp e feita pela API, sem bloquear a criacao. O RLS garante
-- que o usuario so ve os proprios chamados; a Sede ve os da igreja; a
-- plataforma le tudo rodando como `system` (WithSystem).

CREATE TABLE support_tickets (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind          text NOT NULL DEFAULT 'duvida',
    subject       text NOT NULL,
    description   text NOT NULL,
    app           text NOT NULL DEFAULT 'webadmin',
    page_path     text NOT NULL DEFAULT '',
    status        text NOT NULL DEFAULT 'aberto',
    priority      text NOT NULL DEFAULT 'normal',
    context       jsonb NOT NULL DEFAULT '{}'::jsonb,
    ai_summary    text,
    ai_transcript jsonb,
    assigned_to   uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    resolved_at   timestamptz,
    CONSTRAINT support_tickets_kind_check CHECK (kind IN ('duvida','problema','sugestao','acesso','outro')),
    CONSTRAINT support_tickets_status_check CHECK (status IN ('aberto','em_andamento','aguardando','resolvido','fechado')),
    CONSTRAINT support_tickets_priority_check CHECK (priority IN ('baixa','normal','alta','urgente')),
    CONSTRAINT support_tickets_app_check CHECK (app IN ('webadmin','member'))
);

CREATE INDEX idx_support_tickets_tenant ON support_tickets (tenant_id, status, created_at DESC);
CREATE INDEX idx_support_tickets_user ON support_tickets (user_id, created_at DESC);
CREATE INDEX idx_support_tickets_branch ON support_tickets (branch_id);

ALTER TABLE support_tickets ENABLE ROW LEVEL SECURITY;
ALTER TABLE support_tickets FORCE ROW LEVEL SECURITY;

-- Leitura: usuario ve os proprios chamados; Sede ve os da igreja; system ve tudo.
DROP POLICY IF EXISTS support_tickets_sel ON support_tickets;
CREATE POLICY support_tickets_sel ON support_tickets FOR SELECT
  USING (rls_read(tenant_id, branch_id, true)
         AND (user_id = current_user_id() OR is_headquarters() OR is_system()));

-- Criacao: sempre no proprio nome (o gateway usa o user_id das claims).
DROP POLICY IF EXISTS support_tickets_ins ON support_tickets;
CREATE POLICY support_tickets_ins ON support_tickets FOR INSERT
  WITH CHECK (rls_write(tenant_id, branch_id, true)
              AND (user_id = current_user_id() OR is_system()));

-- Atualizacao: o autor pode complementar; Sede/system fazem a triagem.
DROP POLICY IF EXISTS support_tickets_upd ON support_tickets;
CREATE POLICY support_tickets_upd ON support_tickets FOR UPDATE
  USING (rls_write(tenant_id, branch_id, true)
         AND (user_id = current_user_id() OR is_headquarters() OR is_system()))
  WITH CHECK (rls_write(tenant_id, branch_id, true)
              AND (user_id = current_user_id() OR is_headquarters() OR is_system()));

CREATE TABLE support_ticket_messages (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id      uuid NOT NULL REFERENCES support_tickets(id) ON DELETE CASCADE,
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid REFERENCES branches(id) ON DELETE SET NULL,
    author_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    author_kind    text NOT NULL DEFAULT 'user',
    body           text NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT support_ticket_messages_author_check CHECK (author_kind IN ('user','platform','ai'))
);

CREATE INDEX idx_support_ticket_messages_ticket ON support_ticket_messages (ticket_id, created_at);
CREATE INDEX idx_support_ticket_messages_tenant ON support_ticket_messages (tenant_id, created_at DESC);

ALTER TABLE support_ticket_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE support_ticket_messages FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS support_ticket_messages_sel ON support_ticket_messages;
CREATE POLICY support_ticket_messages_sel ON support_ticket_messages FOR SELECT
  USING (rls_read(tenant_id, branch_id, true)
         AND (is_system() OR EXISTS (
             SELECT 1 FROM support_tickets t
             WHERE t.id = support_ticket_messages.ticket_id)));

DROP POLICY IF EXISTS support_ticket_messages_ins ON support_ticket_messages;
CREATE POLICY support_ticket_messages_ins ON support_ticket_messages FOR INSERT
  WITH CHECK (rls_write(tenant_id, branch_id, true)
              AND (is_system() OR EXISTS (
                  SELECT 1 FROM support_tickets t
                  WHERE t.id = support_ticket_messages.ticket_id)));

-- Feedback dos artigos de ajuda (backlog de conteudo).
CREATE TABLE help_feedback (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid REFERENCES branches(id) ON DELETE SET NULL,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app          text NOT NULL DEFAULT 'webadmin',
    article_slug text NOT NULL,
    helpful      boolean NOT NULL DEFAULT true,
    comment      text,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_help_feedback_article ON help_feedback (article_slug, created_at DESC);
CREATE INDEX idx_help_feedback_tenant ON help_feedback (tenant_id, created_at DESC);

ALTER TABLE help_feedback ENABLE ROW LEVEL SECURITY;
ALTER TABLE help_feedback FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS help_feedback_ins ON help_feedback;
CREATE POLICY help_feedback_ins ON help_feedback FOR INSERT
  WITH CHECK (is_system()
              OR (rls_write(tenant_id, branch_id, true) AND user_id = current_user_id()));

DROP POLICY IF EXISTS help_feedback_sel ON help_feedback;
CREATE POLICY help_feedback_sel ON help_feedback FOR SELECT
  USING (is_system()
         OR (rls_read(tenant_id, branch_id, true)
             AND (user_id = current_user_id() OR is_headquarters())));
