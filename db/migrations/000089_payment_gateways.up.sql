-- 000089_payment_gateways.up.sql
-- Configuracao de gateways de pagamento (Asaas e afins).
--
-- `tenant_id` NULL  => conta da PLATAFORMA (Fluxo A: a Chosen ERP cobra as igrejas).
-- `tenant_id` preenchido => conta da IGREJA (Fluxo B: a igreja cobra os membros).
--
-- As credenciais (API key / tokens) ficam em `credentials` jsonb e NUNCA sao
-- projetadas pelos handlers. A abstracao de provider (internal/payments) le
-- esta config para instanciar o cliente do gateway correto.

CREATE TABLE payment_gateway_configs (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid REFERENCES tenants(id) ON DELETE CASCADE, -- NULL = plataforma
    provider     text NOT NULL,                       -- asaas | mercadopago | stone | picpay
    environment  text NOT NULL DEFAULT 'sandbox',     -- sandbox | production
    credentials  jsonb NOT NULL DEFAULT '{}'::jsonb,  -- segredos (nunca devolvidos)
    is_active    boolean NOT NULL DEFAULT true,
    is_default   boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, provider)
);

-- Um unico provider default por escopo (plataforma ou igreja).
CREATE UNIQUE INDEX uq_payment_gateway_default
    ON payment_gateway_configs (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE is_default;

CREATE INDEX idx_payment_gateway_tenant ON payment_gateway_configs (tenant_id);

ALTER TABLE payment_gateway_configs ENABLE ROW LEVEL SECURITY;

-- Leitura: sistema, plataforma ou a propria igreja.
CREATE POLICY pgw_sel ON payment_gateway_configs
  FOR SELECT USING (
      is_system() OR is_platform_admin()
      OR (tenant_id IS NOT NULL AND tenant_id = current_tenant())
  );
-- Escrita: sistema/plataforma (qualquer escopo) ou a propria igreja (so o seu).
CREATE POLICY pgw_ins ON payment_gateway_configs
  FOR INSERT WITH CHECK (
      is_system() OR is_platform_admin() OR tenant_id = current_tenant()
  );
CREATE POLICY pgw_upd ON payment_gateway_configs
  FOR UPDATE USING (
      is_system() OR is_platform_admin() OR tenant_id = current_tenant()
  ) WITH CHECK (
      is_system() OR is_platform_admin() OR tenant_id = current_tenant()
  );
CREATE POLICY pgw_del ON payment_gateway_configs
  FOR DELETE USING (
      is_system() OR is_platform_admin() OR tenant_id = current_tenant()
  );
