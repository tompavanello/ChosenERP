-- 000091_payments.up.sql
-- Arrecadacao (Fluxo B): a igreja cobra os membros via gateway (Asaas e afins).
--
-- `payment_products` e o catalogo do que a igreja oferece (dizimo, oferta,
-- evento, campanha...). Cada produto define as formas de pagamento e, o mais
-- importante, a CATEGORIA CONTABIL (financial_categories) - parametrizavel por
-- produto - usada no lancamento gerado quando uma cobranca e paga. A conta
-- bancaria (`account_id`) tambem e opcional por produto.

CREATE TABLE payment_products (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id             uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL = global do tenant
    name                  text NOT NULL,
    description           text,
    amount_cents          integer CHECK (amount_cents IS NULL OR amount_cents >= 0), -- NULL = valor aberto
    allowed_billing_types text[] NOT NULL DEFAULT ARRAY['PIX','BOLETO','CREDIT_CARD']::text[],
    category_id           uuid NOT NULL REFERENCES financial_categories(id) ON DELETE RESTRICT,
    account_id            uuid REFERENCES financial_accounts(id) ON DELETE SET NULL,
    is_recurring          boolean NOT NULL DEFAULT false,
    is_active             boolean NOT NULL DEFAULT true,
    sort_order            integer NOT NULL DEFAULT 0,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_payment_product_tenant ON payment_products (tenant_id, is_active);

-- Cliente do pagador no gateway (cache por CPF/CNPJ dentro da igreja), para nao
-- recriar o customer a cada cobranca.
CREATE TABLE payment_customers (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider    text NOT NULL,
    external_id text NOT NULL,
    cpf_cnpj    text,
    name        text,
    email       text,
    phone       text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, provider, cpf_cnpj)
);
CREATE INDEX idx_payment_customer_tenant ON payment_customers (tenant_id);

CREATE TABLE payment_charges (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id                uuid REFERENCES branches(id) ON DELETE SET NULL,
    product_id               uuid REFERENCES payment_products(id) ON DELETE SET NULL,
    member_id                uuid REFERENCES members(id) ON DELETE SET NULL,
    provider                 text NOT NULL,
    external_id              text,
    billing_type             text NOT NULL DEFAULT 'PIX',
    amount_cents             integer NOT NULL CHECK (amount_cents >= 0),
    status                   text NOT NULL DEFAULT 'pending', -- pending | paid | overdue | canceled | refunded | failed
    due_date                 date,
    paid_at                  timestamptz,
    payer_name               text,
    payer_cpf_cnpj           text,
    payer_email              text,
    description              text,
    external_reference       text,
    invoice_url              text,
    bank_slip_url            text,
    pix_payload              text,
    pix_qr_image             text,
    -- Lancamento gerado na liquidacao (idempotencia da conciliacao).
    financial_transaction_id uuid REFERENCES financial_transactions(id) ON DELETE SET NULL,
    raw                      jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_payment_charge_external
    ON payment_charges (provider, external_id)
    WHERE external_id IS NOT NULL;
CREATE INDEX idx_payment_charge_tenant ON payment_charges (tenant_id, created_at DESC);
CREATE INDEX idx_payment_charge_member ON payment_charges (member_id);

CREATE TABLE payment_subscriptions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid REFERENCES branches(id) ON DELETE SET NULL,
    product_id     uuid REFERENCES payment_products(id) ON DELETE SET NULL,
    member_id      uuid REFERENCES members(id) ON DELETE SET NULL,
    provider       text NOT NULL,
    external_id    text,
    billing_type   text NOT NULL DEFAULT 'PIX',
    cycle          text NOT NULL DEFAULT 'MONTHLY',
    amount_cents   integer NOT NULL DEFAULT 0,
    next_due_date  date,
    status         text NOT NULL DEFAULT 'active', -- active | inactive | canceled | expired
    payer_name     text,
    payer_cpf_cnpj text,
    payer_email    text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_payment_sub_external
    ON payment_subscriptions (provider, external_id)
    WHERE external_id IS NOT NULL;
CREATE INDEX idx_payment_sub_tenant ON payment_subscriptions (tenant_id);
CREATE INDEX idx_payment_sub_member ON payment_subscriptions (member_id);

-- ---------------------------------------------------------------------------
-- RLS tenant/branch (branch nullable = registro global do tenant)
-- ---------------------------------------------------------------------------
ALTER TABLE payment_products      ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_charges       ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_customers     ENABLE ROW LEVEL SECURITY;

CREATE POLICY pcust_sel ON payment_customers
  FOR SELECT USING (is_system() OR tenant_id = current_tenant());
CREATE POLICY pcust_all ON payment_customers
  FOR ALL USING (is_system() OR tenant_id = current_tenant())
  WITH CHECK (is_system() OR tenant_id = current_tenant());

CREATE POLICY pprod_sel ON payment_products
  USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY pprod_all ON payment_products
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

CREATE POLICY pcharge_sel ON payment_charges
  USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY pcharge_all ON payment_charges
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

CREATE POLICY psub_sel ON payment_subscriptions
  USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY psub_all ON payment_subscriptions
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));
