-- 000062_financial_reconciliation.up.sql
-- CONCILIACAO FINANCEIRA por filial + periodo.
--
-- Fluxo: uma pessoa lanca, outra concilia e outra audita. Ao CONCILIAR um
-- periodo, ele fica TRAVADO no banco: um trigger em financial_transactions
-- recusa INSERT/UPDATE/DELETE de lancamentos cujo occurred_at caia dentro de um
-- periodo conciliado da mesma filial - nem a API nem acesso direto escapam.
--
-- Tambem cria as permissoes finance.reconcile / finance.audit e os papeis
-- conciliador / auditor (separacao de funcoes).

-- ---------------------------------------------------------------------------
-- 1) Permissoes novas
-- ---------------------------------------------------------------------------
INSERT INTO permissions (key, module, name) VALUES
    ('finance.reconcile', 'finance', 'Conciliar financeiro'),
    ('finance.audit',     'finance', 'Auditar financeiro')
ON CONFLICT (key) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 2) Papeis novos por tenant (conciliador / auditor)
-- ---------------------------------------------------------------------------
INSERT INTO roles (tenant_id, key, name, is_system)
SELECT t.id, v.key, v.name, true
FROM tenants t
CROSS JOIN (VALUES
    ('conciliador', 'Conciliador Financeiro'),
    ('auditor',     'Auditor Financeiro')
) AS v(key, name)
ON CONFLICT (tenant_id, key) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 3) Backfill das permissoes
-- ---------------------------------------------------------------------------
-- super_admin / admin_sede: todas as permissoes (inclui as novas).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.key IN ('super_admin', 'admin_sede')
  AND p.key IN ('finance.reconcile', 'finance.audit')
ON CONFLICT DO NOTHING;

-- conciliador: le + concilia; auditor: le + audita + relatorios.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
    ('conciliador', ARRAY['finance.read', 'finance.reconcile']),
    ('auditor',     ARRAY['finance.read', 'finance.audit', 'reports.read'])
) AS v(role_key, perms)
JOIN roles r ON r.key = v.role_key
JOIN permissions p ON p.key = ANY(v.perms)
ON CONFLICT DO NOTHING;

-- Dá ao tesoureiro a conciliacao (ele ja lanca) e ao contador a auditoria,
-- mantendo a possibilidade de separar por conciliador/auditor quando quiser.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM (VALUES
    ('tesoureiro', ARRAY['finance.reconcile']),
    ('contador',   ARRAY['finance.audit'])
) AS v(role_key, perms)
JOIN roles r ON r.key = v.role_key
JOIN permissions p ON p.key = ANY(v.perms)
ON CONFLICT DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4) Tabela de conciliacao
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS financial_reconciliations (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id      uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    title          text NOT NULL DEFAULT 'Conciliacao financeira',
    period_start   date NOT NULL,
    period_end     date NOT NULL,
    status         text NOT NULL DEFAULT 'aberta' CHECK (status IN ('aberta','conciliada')),
    notes          text,
    reconciled_at  timestamptz,
    reconciled_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    signature_hash text,
    created_by     uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT financial_reconciliations_period_check CHECK (period_end >= period_start)
);

CREATE INDEX IF NOT EXISTS idx_fin_recon_tenant
    ON financial_reconciliations(tenant_id, branch_id, period_start DESC);

-- ---------------------------------------------------------------------------
-- 5) Guard da conciliacao: sem periodos sobrepostos na mesma filial e
--    imutabilidade depois de conciliada.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_reconciliations_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_id     uuid := COALESCE(NEW.id, OLD.id);
    v_tenant uuid := COALESCE(NEW.tenant_id, OLD.tenant_id);
    v_branch uuid := COALESCE(NEW.branch_id, OLD.branch_id);
    v_start  date := COALESCE(NEW.period_start, OLD.period_start);
    v_end    date := COALESCE(NEW.period_end, OLD.period_end);
    v_count  int;
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.status = 'conciliada' THEN
        RAISE EXCEPTION 'conciliacao concluida: periodo travado (documento imutavel)';
    END IF;

    SELECT count(*) INTO v_count
    FROM financial_reconciliations fr
    WHERE fr.tenant_id = v_tenant
      AND fr.branch_id = v_branch
      AND fr.id <> v_id
      AND daterange(fr.period_start, fr.period_end, '[]')
          && daterange(v_start, v_end, '[]');

    IF v_count > 0 THEN
        RAISE EXCEPTION 'ja existe conciliacao para periodo sobreposto nesta filial'
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS fin_reconciliations_guard ON financial_reconciliations;
CREATE TRIGGER fin_reconciliations_guard
BEFORE INSERT OR UPDATE ON financial_reconciliations
FOR EACH ROW EXECUTE FUNCTION fin_reconciliations_guard();

-- ---------------------------------------------------------------------------
-- 6) TRAVA do periodo: bloqueia lancamentos dentro de periodo conciliado.
--    O recadeamento interno (fin_tx_rechain) e liberado para nao quebrar a
--    exclusao definitiva nem a recalculo da hash-chain.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_period_lock() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_tenant uuid;
    v_branch uuid;
    v_date   date;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_tenant := OLD.tenant_id;
        v_branch := OLD.branch_id;
        v_date   := OLD.occurred_at::date;
    ELSE
        IF TG_OP = 'UPDATE'
           AND current_setting('app.fin_rechain', true) = 'on'
           AND current_user <> session_user THEN
            RETURN NEW;
        END IF;
        v_tenant := NEW.tenant_id;
        v_branch := NEW.branch_id;
        v_date   := NEW.occurred_at::date;
    END IF;

    IF EXISTS (
        SELECT 1 FROM financial_reconciliations fr
        WHERE fr.tenant_id = v_tenant
          AND fr.branch_id = v_branch
          AND fr.status = 'conciliada'
          AND v_date BETWEEN fr.period_start AND fr.period_end
    ) THEN
        RAISE EXCEPTION 'periodo conciliado: lancamentos nao podem ser criados, alterados ou excluidos'
            USING ERRCODE = '23514';
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

DROP TRIGGER IF EXISTS fin_tx_period_lock ON financial_transactions;
CREATE TRIGGER fin_tx_period_lock
BEFORE INSERT OR UPDATE OR DELETE ON financial_transactions
FOR EACH ROW EXECUTE FUNCTION fin_tx_period_lock();

-- ---------------------------------------------------------------------------
-- 7) RLS
-- ---------------------------------------------------------------------------
ALTER TABLE financial_reconciliations ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS fin_recon_sel ON financial_reconciliations;
CREATE POLICY fin_recon_sel ON financial_reconciliations
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_recon_ins ON financial_reconciliations;
CREATE POLICY fin_recon_ins ON financial_reconciliations
    FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_recon_upd ON financial_reconciliations;
CREATE POLICY fin_recon_upd ON financial_reconciliations
    FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));
DROP POLICY IF EXISTS fin_recon_del ON financial_reconciliations;
CREATE POLICY fin_recon_del ON financial_reconciliations
    FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- 8) create_tenant passa a criar tambem conciliador/auditor (novas igrejas)
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
        (v_tenant, 'conciliador',   'Conciliador Financeiro',     true),
        (v_tenant, 'auditor',       'Auditor Financeiro',         true),
        (v_tenant, 'visitante',     'Visitante',                  true);

    -- super_admin e admin_sede: todas as permissoes.
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.tenant_id = v_tenant AND r.key IN ('super_admin', 'admin_sede');

    -- Demais papeis: conjunto curado por funcao (mesma matriz do seed).
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM (VALUES
        ('pastor_filial', ARRAY['families.read','finance.read','governance.read','members.read','members.write','ministries.read','ministries.write','reports.read','visitors.read']),
        ('tesoureiro',    ARRAY['finance.read','finance.write','finance.reconcile','members.read','reports.read']),
        ('secretario',    ARRAY['families.read','members.read','members.write','reports.read','visitors.read']),
        ('lider',         ARRAY['families.read','members.read','ministries.read','ministries.write','reports.read']),
        ('pastor',        ARRAY['families.read','governance.read','members.read','ministries.read','reports.read']),
        ('contador',      ARRAY['finance.read','finance.audit','reports.read']),
        ('conciliador',   ARRAY['finance.read','finance.reconcile']),
        ('auditor',       ARRAY['finance.read','finance.audit','reports.read'])
    ) AS v(role_key, perms)
    JOIN roles r ON r.tenant_id = v_tenant AND r.key = v.role_key
    CROSS JOIN LATERAL unnest(v.perms) AS perm(key)
    JOIN permissions p ON p.key = perm.key
    ON CONFLICT DO NOTHING;

    RETURN v_tenant;
END;
$$;

GRANT EXECUTE ON FUNCTION create_tenant(text, text, text) TO chosenerp_app;
