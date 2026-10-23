-- 000104_fin_period_item_locks.down.sql
-- Reverte a conciliacao por lancamento e as travas cruzadas de periodo.

DROP TRIGGER IF EXISTS fin_tx_reconciled_lock ON financial_transactions;
DROP FUNCTION IF EXISTS fin_tx_reconciled_lock();
DROP FUNCTION IF EXISTS fin_tx_reconcile(uuid, boolean, text);

DROP INDEX IF EXISTS idx_fin_tx_reconciled;
ALTER TABLE financial_transactions
    DROP COLUMN IF EXISTS reconciled_by,
    DROP COLUMN IF EXISTS reconciled_at;

-- Restaura o guard append-only da migracao 000096.
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_edit', true) = 'on') THEN
        RETURN NEW;
    END IF;
    IF NEW.tenant_id       IS DISTINCT FROM OLD.tenant_id
       OR NEW.branch_id    IS DISTINCT FROM OLD.branch_id
       OR NEW.category_id  IS DISTINCT FROM OLD.category_id
       OR NEW.type         IS DISTINCT FROM OLD.type
       OR NEW.amount       IS DISTINCT FROM OLD.amount
       OR NEW.currency     IS DISTINCT FROM OLD.currency
       OR NEW.donor_member_id IS DISTINCT FROM OLD.donor_member_id
       OR NEW.benefactor_id   IS DISTINCT FROM OLD.benefactor_id
       OR NEW.donor_name      IS DISTINCT FROM OLD.donor_name
       OR NEW.payment_method  IS DISTINCT FROM OLD.payment_method
       OR NEW.is_anonymous    IS DISTINCT FROM OLD.is_anonymous
       OR NEW.occurred_at     IS DISTINCT FROM OLD.occurred_at
       OR NEW.description     IS DISTINCT FROM OLD.description
       OR NEW.account_id      IS DISTINCT FROM OLD.account_id
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente exclusao e permitida)';
    END IF;
    RETURN NEW;
END;
$$;

-- Restaura a trava de periodo da migracao 000062 (somente conciliacoes).
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

-- Restaura o guard de auditoria da migracao 000063.
CREATE OR REPLACE FUNCTION fin_audits_period_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_id     uuid := COALESCE(NEW.id, OLD.id);
    v_tenant uuid := COALESCE(NEW.tenant_id, OLD.tenant_id);
    v_branch uuid := COALESCE(NEW.branch_id, OLD.branch_id);
    v_start  date := COALESCE(NEW.period_start, OLD.period_start);
    v_end    date := COALESCE(NEW.period_end, OLD.period_end);
    v_count  int;
BEGIN
    SELECT count(*) INTO v_count
    FROM financial_audits a
    WHERE a.tenant_id = v_tenant
      AND a.branch_id = v_branch
      AND a.id <> v_id
      AND daterange(a.period_start, a.period_end, '[]')
          && daterange(v_start, v_end, '[]');

    IF v_count > 0 THEN
        RAISE EXCEPTION 'ja existe auditoria para periodo sobreposto nesta filial'
            USING ERRCODE = '23505';
    END IF;

    RETURN NEW;
END;
$$;

-- Restaura o guard de conciliacao da migracao 000062.
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
