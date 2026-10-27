-- 000105_fin_tx_groups.down.sql

DROP TRIGGER IF EXISTS fin_tx_group_check ON financial_transactions;
DROP FUNCTION IF EXISTS fin_tx_group_check();

DROP TRIGGER IF EXISTS fin_group_guard ON financial_transaction_groups;
DROP FUNCTION IF EXISTS fin_group_guard();
DROP FUNCTION IF EXISTS fin_group_attach(uuid, uuid[]);
DROP FUNCTION IF EXISTS fin_group_detach(uuid, uuid[]);
DROP FUNCTION IF EXISTS fin_group_reconcile(uuid, boolean, text);

DROP INDEX IF EXISTS idx_bank_entries_group;
ALTER TABLE bank_statement_entries DROP COLUMN IF EXISTS group_id;

DROP INDEX IF EXISTS idx_fin_tx_group;
ALTER TABLE financial_transactions DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS financial_transaction_groups;

-- Restaura o guard append-only sem group_id (versao da 000104).
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_edit', true) = 'on'
            OR current_setting('app.fin_reconcile', true) = 'on') THEN
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
       OR NEW.reconciled_at   IS DISTINCT FROM OLD.reconciled_at
       OR NEW.reconciled_by   IS DISTINCT FROM OLD.reconciled_by
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente exclusao e permitida)';
    END IF;
    RETURN NEW;
END;
$$;

-- Restaura a conciliacao individual sem a trava de grupo (versao da 000104).
CREATE OR REPLACE FUNCTION fin_tx_reconcile(
    p_id         uuid,
    p_reconciled boolean,
    p_actor      text
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_row_tenant uuid;
    v_branch uuid;
    v_date date;
    v_voided timestamptz;
    v_closed boolean;
BEGIN
    SELECT tenant_id, branch_id, occurred_at::date, voided_at
    INTO v_row_tenant, v_branch, v_date, v_voided
    FROM financial_transactions
    WHERE id = p_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_tx_reconcile: lancamento nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_row_tenant THEN
        RAISE EXCEPTION 'fin_tx_reconcile: tenant fora do contexto';
    END IF;
    IF v_voided IS NOT NULL THEN
        RAISE EXCEPTION 'lancamento estornado: nao pode ser conciliado'
            USING ERRCODE = '23514';
    END IF;

    SELECT EXISTS (
        SELECT 1 FROM financial_reconciliations fr
        WHERE fr.tenant_id = v_row_tenant
          AND fr.branch_id = v_branch
          AND fr.status = 'conciliada'
          AND v_date BETWEEN fr.period_start AND fr.period_end
        UNION ALL
        SELECT 1 FROM financial_audits a
        WHERE a.tenant_id = v_row_tenant
          AND a.branch_id = v_branch
          AND a.status = 'fechada'
          AND v_date BETWEEN a.period_start AND a.period_end
    ) INTO v_closed;

    IF v_closed THEN
        RAISE EXCEPTION 'periodo encerrado: conciliacao nao pode ser alterada'
            USING ERRCODE = '23514';
    END IF;

    PERFORM set_config('app.fin_reconcile', 'on', true);
    UPDATE financial_transactions
    SET reconciled_at = CASE WHEN p_reconciled THEN COALESCE(reconciled_at, now()) ELSE NULL END,
        reconciled_by = CASE WHEN p_reconciled THEN COALESCE(reconciled_by, NULLIF(p_actor, '')::uuid) ELSE NULL END
    WHERE id = p_id;
    PERFORM set_config('app.fin_reconcile', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_reconcile(uuid, boolean, text) TO PUBLIC;
