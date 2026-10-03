-- 000084_member_requests.up.sql
-- Solicitacoes/formularios do app do membro (atualizacao cadastral, carta,
-- visita pastoral, batismo, profissao de fe, transferencia, casamento,
-- inscricao em evento...). O membro cria e acompanha; a secretaria responde.
CREATE TABLE member_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    member_id    uuid REFERENCES members(id) ON DELETE SET NULL,
    user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
    kind         text NOT NULL DEFAULT 'outro',
    subject      text NOT NULL,
    message      text,
    status       text NOT NULL DEFAULT 'pendente',
    response     text,
    responded_by uuid REFERENCES users(id) ON DELETE SET NULL,
    responded_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT member_requests_status_check CHECK (status IN ('pendente', 'em_andamento', 'concluida', 'cancelada'))
);

CREATE INDEX idx_member_requests_scope ON member_requests (tenant_id, branch_id, status, created_at);
CREATE INDEX idx_member_requests_member ON member_requests (member_id);

ALTER TABLE member_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_requests FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS requests_sel ON member_requests;
CREATE POLICY requests_sel ON member_requests FOR SELECT
  USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS requests_all ON member_requests;
CREATE POLICY requests_all ON member_requests
  USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
