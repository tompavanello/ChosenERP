-- 000100_payments_rls_hardening.up.sql
-- Correcoes de RLS e integridade do modulo de pagamentos + restauracao das
-- permissoes de cobranca no `create_tenant`.
--
-- 1) As policies `pprod_sel`/`pcharge_sel`/`psub_sel` (000091) nasceram como
--    `CREATE POLICY ... USING(...)` sem `FOR SELECT`, ou seja, default `FOR ALL`
--    - o mesmo defeito de "escrita pela leitura" corrigido globalmente na 000037.
--    Recriadas explicitamente como `FOR SELECT`.
-- 2) CHECKs de dominio nos status/provider/environment (como a 000088 fez).
-- 3) Indices faltantes nas FKs e colunas de filtro RLS.
-- 4) `create_tenant` volta a conceder `payments.read/write` (a 000095 redefiniu a
--    funcao e deixou de fora as permissoes criadas na 000092).

-- ---------------------------------------------------------------------------
-- 1) RLS: as policies de leitura devem ser somente SELECT.
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS pprod_sel ON payment_products;
CREATE POLICY pprod_sel ON payment_products
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));

DROP POLICY IF EXISTS pcharge_sel ON payment_charges;
CREATE POLICY pcharge_sel ON payment_charges
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));

DROP POLICY IF EXISTS psub_sel ON payment_subscriptions;
CREATE POLICY psub_sel ON payment_subscriptions
  FOR SELECT USING (rls_read(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- 2) CHECKs de dominio.
-- ---------------------------------------------------------------------------
ALTER TABLE payment_charges
  ADD CONSTRAINT payment_charges_status_check
  CHECK (status IN ('pending','paid','overdue','canceled','refunded','failed'));

ALTER TABLE payment_subscriptions
  ADD CONSTRAINT payment_subscriptions_status_check
  CHECK (status IN ('active','inactive','canceled','expired'));

ALTER TABLE billing_subscriptions
  ADD CONSTRAINT billing_subscriptions_status_check
  CHECK (status IN ('active','inactive','canceled','expired'));

ALTER TABLE billing_invoices
  ADD CONSTRAINT billing_invoices_status_check
  CHECK (status IN ('pending','paid','overdue','canceled','refunded','failed'));

ALTER TABLE payment_gateway_configs
  ADD CONSTRAINT payment_gateway_environment_check
  CHECK (environment IN ('sandbox','production'));

ALTER TABLE payment_gateway_configs
  ADD CONSTRAINT payment_gateway_provider_check
  CHECK (provider IN ('asaas','mercadopago','stone','picpay'));

-- ---------------------------------------------------------------------------
-- 3) Indices faltantes (FKs / colunas de escopo).
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_payment_charge_branch    ON payment_charges (branch_id);
CREATE INDEX IF NOT EXISTS idx_payment_charge_product   ON payment_charges (product_id);
CREATE INDEX IF NOT EXISTS idx_payment_charge_fin_tx    ON payment_charges (financial_transaction_id);
CREATE INDEX IF NOT EXISTS idx_payment_sub_product      ON payment_subscriptions (product_id);
CREATE INDEX IF NOT EXISTS idx_payment_sub_branch       ON payment_subscriptions (branch_id);
CREATE INDEX IF NOT EXISTS idx_billing_invoice_sub      ON billing_invoices (subscription_id);
CREATE INDEX IF NOT EXISTS idx_billing_webhook_tenant   ON billing_webhook_events (tenant_id);
CREATE INDEX IF NOT EXISTS idx_event_rsvps_tenant       ON event_rsvps (tenant_id, branch_id);
CREATE INDEX IF NOT EXISTS idx_devotional_days_branch   ON devotional_plan_days (branch_id);
CREATE INDEX IF NOT EXISTS idx_devotional_progress_mem  ON devotional_progress (member_id);
CREATE INDEX IF NOT EXISTS idx_content_videos_created_by ON content_videos (created_by);

-- ---------------------------------------------------------------------------
-- 4) create_tenant: inclui content.* (000095) E payments.* (000092).
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION create_tenant(
    p_name text,
    p_slug text,
    p_plan text DEFAULT 'starter'
)
RETURNS uuid
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
    v_tenant uuid;
    v_slug   text := lower(trim(p_slug));
    reserved text[] := ARRAY['app', 'www', 'api', 'admin', 'localhost'];
BEGIN
    IF p_name IS NULL OR trim(p_name) = '' THEN
        RAISE EXCEPTION 'nome da igreja obrigatorio' USING ERRCODE = '22023';
    END IF;
    IF v_slug !~ '^[a-z0-9][a-z0-9-]{1,38}$' THEN
        RAISE EXCEPTION 'slug invalido (minusculas, numeros e hifen; 2 a 39 chars)' USING ERRCODE = '22023';
    END IF;
    IF v_slug = ANY (reserved) THEN
        RAISE EXCEPTION 'slug reservado' USING ERRCODE = '22023';
    END IF;
    IF EXISTS (SELECT 1 FROM tenants WHERE slug = v_slug) THEN
        RAISE EXCEPTION 'slug ja existe' USING ERRCODE = '23505';
    END IF;

    INSERT INTO tenants (name, slug, plan)
    VALUES (trim(p_name), v_slug, COALESCE(NULLIF(trim(p_plan), ''), 'starter'))
    RETURNING id INTO v_tenant;

    INSERT INTO branches (tenant_id, name, slug, kind)
    VALUES (v_tenant, 'Sede Matriz', 'matriz', 'matriz');

    INSERT INTO roles (tenant_id, key, name, is_system) VALUES
        (v_tenant, 'super_admin',   'Super Admin',                true),
        (v_tenant, 'admin_sede',    'Admin da Sede',              true),
        (v_tenant, 'pastor_filial', 'Pastor da Filial',           true),
        (v_tenant, 'tesoureiro',    'Tesoureiro',                 true),
        (v_tenant, 'secretario',    'Secretario(a)',              true),
        (v_tenant, 'membro',        'Membro',                     true),
        (v_tenant, 'lider',         'Lider de Ministerio/Celula', true),
        (v_tenant, 'pastor',        'Pastor/Conselheiro',         true),
        (v_tenant, 'contador',      'Contador Externo',           true),
        (v_tenant, 'visitante',     'Visitante',                  true);

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM (VALUES
        ('pastor_filial', ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','payments.read','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','payments.read','payments.write','reports.read']),
        ('secretario',    ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','members.read','members.write','payments.read','reports.read','visitors.read']),
        ('lider',         ARRAY['announcements.read','announcements.write','content.read','families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','governance.read','members.read','ministries.read','payments.read','reports.read']),
        ('contador',      ARRAY['finance.read','payments.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;

GRANT EXECUTE ON FUNCTION create_tenant(text, text, text) TO chosenerp_app;
