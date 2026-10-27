-- 000106_fin_group_same_date_seq.down.sql

DROP TRIGGER IF EXISTS fin_group_same_date_check ON financial_transaction_groups;
DROP FUNCTION IF EXISTS fin_group_same_date_check();
DROP FUNCTION IF EXISTS fin_group_move_date(uuid, date, integer);

CREATE OR REPLACE FUNCTION fin_tx_group_check() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    g_tenant  uuid;
    g_branch  uuid;
    g_account uuid;
    g_type    text;
    g_status  text;
BEGIN
    IF NEW.group_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT tenant_id, branch_id, account_id, type, status
    INTO g_tenant, g_branch, g_account, g_type, g_status
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
       OR NEW.type IS DISTINCT FROM g_type THEN
        RAISE EXCEPTION 'lancamento incompativel com o grupo (tenant/filial/conta/tipo)'
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

DROP INDEX IF EXISTS idx_fin_grp_entry_seq;

ALTER TABLE financial_transaction_groups
    DROP COLUMN IF EXISTS entry_seq;
