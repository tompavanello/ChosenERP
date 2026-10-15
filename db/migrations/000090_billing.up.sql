-- 000090_billing.up.sql
-- Cobranca SaaS (Fluxo A): a Chosen ERP cobra a IGREJA pelos planos.
--
-- As linhas sao gerenciadas pela plataforma (system/plataforma admin), mas a
-- igreja pode LER a propria assinatura e as proprias faturas (area da igreja).
-- A criacao da assinatura no gateway e automatica a partir do `plans.price_cents`
-- (o Asaas nao tem objeto de plano; para provedores que tem, `plans.gateway_refs`
-- guarda o mapeamento).

-- Cliente no gateway (mapeamento tenant <-> customer externo).
CREATE TABLE billing_customers (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider    text NOT NULL,
    external_id text NOT NULL,
    name        text,
    email       text,
    phone       text,
    cpf_cnpj    text,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, provider)
);
CREATE INDEX idx_billing_customer_tenant ON billing_customers (tenant_id);

CREATE TABLE billing_subscriptions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_key      text REFERENCES plans(key) ON DELETE SET NULL,
    provider      text NOT NULL,
    external_id   text,
    billing_type  text NOT NULL DEFAULT 'PIX',       -- PIX | BOLETO | CREDIT_CARD
    cycle         text NOT NULL DEFAULT 'MONTHLY',   -- MONTHLY | YEARLY
    value_cents   integer NOT NULL DEFAULT 0,
    next_due_date date,
    status        text NOT NULL DEFAULT 'active',    -- active | inactive | canceled | expired
    started_at    timestamptz NOT NULL DEFAULT now(),
    canceled_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- Uma assinatura "viva" (active/inactive) por igreja.
CREATE UNIQUE INDEX uq_billing_subscription_active
    ON billing_subscriptions (tenant_id)
    WHERE status IN ('active', 'inactive');

CREATE INDEX idx_billing_sub_tenant ON billing_subscriptions (tenant_id);

CREATE TABLE billing_invoices (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id    uuid REFERENCES billing_subscriptions(id) ON DELETE SET NULL,
    provider           text NOT NULL,
    external_id        text,
    billing_type       text,
    status             text NOT NULL DEFAULT 'pending', -- pending | paid | overdue | canceled | refunded | failed
    value_cents        integer NOT NULL DEFAULT 0,
    net_value_cents    integer,
    due_date           date,
    paid_at            timestamptz,
    invoice_url        text,
    bank_slip_url      text,
    pix_payload        text,
    pix_qr_image       text,
    description        text,
    external_reference text,
    raw                jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_billing_invoice_external
    ON billing_invoices (provider, external_id)
    WHERE external_id IS NOT NULL;
CREATE INDEX idx_billing_invoice_tenant ON billing_invoices (tenant_id, due_date DESC);
CREATE INDEX idx_billing_invoice_status ON billing_invoices (status, due_date);

-- Idempotencia dos webhooks: um evento do gateway so e processado uma vez.
CREATE TABLE billing_webhook_events (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider     text NOT NULL,
    event_id     text NOT NULL,
    event_type   text,
    payload      jsonb NOT NULL DEFAULT '{}'::jsonb,
    scope        text NOT NULL DEFAULT 'platform', -- platform | tenant
    tenant_id    uuid REFERENCES tenants(id) ON DELETE SET NULL,
    received_at  timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    error        text,
    UNIQUE (provider, event_id)
);

-- Estado de cobranca no proprio tenant.
ALTER TABLE tenants ADD COLUMN billing_status text NOT NULL DEFAULT 'active';
ALTER TABLE tenants ADD COLUMN trial_ends_at timestamptz;
ALTER TABLE tenants ADD COLUMN billing_exempt boolean NOT NULL DEFAULT false;
ALTER TABLE tenants ADD COLUMN past_due_since timestamptz;
ALTER TABLE tenants ADD COLUMN suspended_at timestamptz;

-- Mapeamento do plano para o gateway (provedores com objeto de plano) + ciclo.
ALTER TABLE plans ADD COLUMN cycle text NOT NULL DEFAULT 'MONTHLY';
ALTER TABLE plans ADD COLUMN gateway_refs jsonb NOT NULL DEFAULT '{}'::jsonb;

-- ---------------------------------------------------------------------------
-- RLS: plataforma/system escrevem; a igreja le os proprios dados.
-- ---------------------------------------------------------------------------
ALTER TABLE billing_customers    ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_invoices      ENABLE ROW LEVEL SECURITY;
ALTER TABLE billing_webhook_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY bcust_sel ON billing_customers
  FOR SELECT USING (is_system() OR is_platform_admin() OR tenant_id = current_tenant());
CREATE POLICY bsub_sel ON billing_subscriptions
  FOR SELECT USING (is_system() OR is_platform_admin() OR tenant_id = current_tenant());
CREATE POLICY binv_sel ON billing_invoices
  FOR SELECT USING (is_system() OR is_platform_admin() OR tenant_id = current_tenant());

-- Plataforma/system gerenciam a cobranca.
CREATE POLICY bcust_all ON billing_customers
  FOR ALL USING (is_system() OR is_platform_admin())
  WITH CHECK (is_system() OR is_platform_admin());
CREATE POLICY bsub_all ON billing_subscriptions
  FOR ALL USING (is_system() OR is_platform_admin())
  WITH CHECK (is_system() OR is_platform_admin());
CREATE POLICY binv_all ON billing_invoices
  FOR ALL USING (is_system() OR is_platform_admin())
  WITH CHECK (is_system() OR is_platform_admin());

-- Eventos de webhook: somente sistema/plataforma.
CREATE POLICY bwh_sel ON billing_webhook_events
  FOR SELECT USING (is_system() OR is_platform_admin());
CREATE POLICY bwh_all ON billing_webhook_events
  FOR ALL USING (is_system() OR is_platform_admin())
  WITH CHECK (is_system() OR is_platform_admin());
