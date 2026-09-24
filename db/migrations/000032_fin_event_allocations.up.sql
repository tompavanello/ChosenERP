-- 000032_fin_event_allocations.up.sql
-- Rateio de um lançamento entre eventos: quanto de cada lançamento pertence a
-- cada evento. Permite saber o custo REAL de um evento (soma do rateio) além do
-- custo estimado já cadastrado.

CREATE TABLE financial_event_allocations (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    transaction_id uuid NOT NULL REFERENCES financial_transactions(id) ON DELETE CASCADE,
    event_id       uuid NOT NULL REFERENCES church_events(id) ON DELETE CASCADE,
    amount         numeric(14,2) NOT NULL CHECK (amount >= 0),
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_fin_evt_alloc_tx ON financial_event_allocations(transaction_id);
CREATE INDEX idx_fin_evt_alloc_event ON financial_event_allocations(event_id);

ALTER TABLE financial_event_allocations ENABLE ROW LEVEL SECURITY;
CREATE POLICY fin_evt_alloc_sel ON financial_event_allocations
  FOR SELECT USING (rls_read(tenant_id, branch_id, false));
CREATE POLICY fin_evt_alloc_all ON financial_event_allocations
  FOR ALL USING (rls_write(tenant_id, branch_id, false))
  WITH CHECK (rls_write(tenant_id, branch_id, false));
