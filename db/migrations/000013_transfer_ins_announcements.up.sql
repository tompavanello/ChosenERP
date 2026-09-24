-- 000013_transfer_ins_announcements.up.sql
-- 1) Repasses: política de INSERT (oficializar repasse entre filiais)
-- 2) Avisos (feed do app do membro), escopados por filial/tenant

-- Repasses: a tabela transfers só tinha policy de SELECT; sem INSERT a RLS
-- bloqueia a criação. Autoriza quem é origem ou destino (ou sede).
CREATE POLICY transfers_ins ON transfers FOR INSERT
  WITH CHECK (
    from_branch_id = current_branch() OR
    to_branch_id   = current_branch() OR
    is_headquarters()
  );

-- Avisos do app do membro
CREATE TABLE announcements (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id    uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => sede/todos
    title        text NOT NULL,
    body         text NOT NULL,
    audience     text NOT NULL DEFAULT 'everyone', -- everyone | members | leaders
    is_active    boolean NOT NULL DEFAULT true,
    published_at timestamptz NOT NULL DEFAULT now(),
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_announcements_branch ON announcements(tenant_id, branch_id, published_at DESC);

ALTER TABLE announcements ENABLE ROW LEVEL SECURITY;

CREATE POLICY announcements_sel ON announcements FOR SELECT
  USING (branch_id = current_branch() OR branch_id IS NULL OR is_headquarters());
CREATE POLICY announcements_all ON announcements FOR ALL
  USING (branch_id = current_branch() OR branch_id IS NULL)
  WITH CHECK (branch_id = current_branch() OR branch_id IS NULL);
