-- 000108_account_transfers.down.sql

DROP INDEX IF EXISTS idx_fin_tx_account_transfer;
ALTER TABLE financial_transactions DROP COLUMN IF EXISTS account_transfer_id;

DROP TABLE IF EXISTS account_transfers;

ALTER TABLE financial_accounts
    DROP CONSTRAINT IF EXISTS financial_accounts_account_type_check;

-- Restaura o guard append-only da 000105.
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_edit', true) = 'on'
            OR current_setting('app.fin_reconcile', true) = 'on'
            OR current_setting('app.fin_group', true) = 'on') THEN
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
       OR NEW.group_id        IS DISTINCT FROM OLD.group_id
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

-- Restaura o check de grupo da 000106.
CREATE OR REPLACE FUNCTION fin_tx_group_check() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    g_tenant   uuid;
    g_branch   uuid;
    g_account  uuid;
    g_type     text;
    g_status   text;
    g_date     date;
BEGIN
    IF NEW.group_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT tenant_id, branch_id, account_id, type, status, occurred_at
    INTO g_tenant, g_branch, g_account, g_type, g_status, g_date
    FROM financial_transaction_groups
    WHERE id = NEW.group_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'grupo de lancamentos nao encontrado'
            USING ERRCODE = '23503';
    END IF;

    IF NEW.voided_at IS NOT NULL THEN
        RAISE EXCEPTION 'lancamento estornado nao pode ser agrupado'
            USING ERRCODE = '23514';
    END IF;

    IF NEW.tenant_id IS DISTINCT FROM g_tenant
       OR NEW.branch_id IS DISTINCT FROM g_branch
       OR NEW.account_id IS DISTINCT FROM g_account
       OR NEW.type IS DISTINCT FROM g_type
       OR (NEW.occurred_at AT TIME ZONE 'UTC')::date IS DISTINCT FROM g_date THEN
        RAISE EXCEPTION 'lancamento incompativel com o grupo (tenant/filial/conta/tipo/data)'
            USING ERRCODE = '23514';
    END IF;

    IF g_status = 'conciliado' THEN
        IF TG_OP = 'INSERT' THEN
            RAISE EXCEPTION 'grupo conciliado: nao aceita novos lancamentos'
                USING ERRCODE = '23514';
        END IF;
        IF TG_OP = 'UPDATE' AND NEW.group_id IS DISTINCT FROM OLD.group_id THEN
            RAISE EXCEPTION 'grupo conciliado: nao aceita novos lancamentos'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;
