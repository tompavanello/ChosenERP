-- 000014_recurring_donations.up.sql
-- Agendamentos de doacoes recorrentes (dizimo/oferta) + worker que gera
-- lancamentos automaticos com recibo a cada periodo.

CREATE TABLE recurring_donations (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    member_id      uuid REFERENCES members(id) ON DELETE SET NULL,
    benefactor_id  uuid REFERENCES benefactors(id) ON DELETE SET NULL,
    category_id    uuid REFERENCES financial_categories(id) ON DELETE SET NULL,
    subtype        text NOT NULL DEFAULT 'doacao',   -- dizimo | oferta | doacao
    amount         numeric(14,2) NOT NULL CHECK (amount > 0),
    frequency      text NOT NULL DEFAULT 'monthly',  -- weekly | monthly | yearly
    period_start   timestamptz NOT NULL DEFAULT now(),
    next_run_at    timestamptz NOT NULL,
    last_run_at    timestamptz,
    times_run      integer NOT NULL DEFAULT 0,
    is_active      boolean NOT NULL DEFAULT true,
    payment_method text,
    description    text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_recurring_next ON recurring_donations(tenant_id, branch_id, is_active, next_run_at);
CREATE INDEX idx_recurring_tenant ON recurring_donations(tenant_id);

ALTER TABLE recurring_donations ENABLE ROW LEVEL SECURITY;

CREATE POLICY recurring_sel ON recurring_donations
  FOR SELECT USING (branch_id = current_branch() OR is_headquarters());
CREATE POLICY recurring_ins ON recurring_donations
  FOR INSERT WITH CHECK (branch_id = current_branch());
CREATE POLICY recurring_upd ON recurring_donations
  FOR UPDATE USING (branch_id = current_branch()) WITH CHECK (branch_id = current_branch());
