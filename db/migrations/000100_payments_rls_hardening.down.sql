-- 000100_payments_rls_hardening.down.sql

-- 1) Politicas de volta ao estado da 000091 (FOR ALL - default).
DROP POLICY IF EXISTS pprod_sel ON payment_products;
CREATE POLICY pprod_sel ON payment_products
  USING (rls_read(tenant_id, branch_id, true));

DROP POLICY IF EXISTS pcharge_sel ON payment_charges;
CREATE POLICY pcharge_sel ON payment_charges
  USING (rls_read(tenant_id, branch_id, true));

DROP POLICY IF EXISTS psub_sel ON payment_subscriptions;
CREATE POLICY psub_sel ON payment_subscriptions
  USING (rls_read(tenant_id, branch_id, true));

-- 2) Remove CHECKs.
ALTER TABLE payment_charges        DROP CONSTRAINT IF EXISTS payment_charges_status_check;
ALTER TABLE payment_subscriptions  DROP CONSTRAINT IF EXISTS payment_subscriptions_status_check;
ALTER TABLE billing_subscriptions  DROP CONSTRAINT IF EXISTS billing_subscriptions_status_check;
ALTER TABLE billing_invoices       DROP CONSTRAINT IF EXISTS billing_invoices_status_check;
ALTER TABLE payment_gateway_configs DROP CONSTRAINT IF EXISTS payment_gateway_environment_check;
ALTER TABLE payment_gateway_configs DROP CONSTRAINT IF EXISTS payment_gateway_provider_check;

-- 3) Remove indices.
DROP INDEX IF EXISTS idx_payment_charge_branch;
DROP INDEX IF EXISTS idx_payment_charge_product;
DROP INDEX IF EXISTS idx_payment_charge_fin_tx;
DROP INDEX IF EXISTS idx_payment_sub_product;
DROP INDEX IF EXISTS idx_payment_sub_branch;
DROP INDEX IF EXISTS idx_billing_invoice_sub;
DROP INDEX IF EXISTS idx_billing_webhook_tenant;
DROP INDEX IF EXISTS idx_event_rsvps_tenant;
DROP INDEX IF EXISTS idx_devotional_days_branch;
DROP INDEX IF EXISTS idx_devotional_progress_mem;
DROP INDEX IF EXISTS idx_content_videos_created_by;

-- 4) create_tenant volta ao estado da 000095 (sem payments.*).
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
        ('pastor_filial', ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','members.read','reports.read']),
        ('secretario',    ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['announcements.read','announcements.write','content.read','families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['announcements.read','announcements.write','content.read','content.write','families.read','governance.read','members.read','ministries.read','reports.read']),
        ('contador',      ARRAY['finance.read','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;
