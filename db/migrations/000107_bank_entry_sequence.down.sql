-- 000107_bank_entry_sequence.down.sql

DROP FUNCTION IF EXISTS fin_lines_set_seq(text, uuid, integer);

-- Restaura o guard do lancamento conciliado (000104).
CREATE OR REPLACE FUNCTION fin_tx_reconciled_lock() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_reconcile', true) = 'on') THEN
        RETURN COALESCE(NEW, OLD);
    END IF;
    IF OLD.reconciled_at IS NOT NULL THEN
        IF TG_OP = 'DELETE' THEN
            RAISE EXCEPTION 'lancamento conciliado: nao pode ser excluido'
                USING ERRCODE = '23514';
        END IF;
        RAISE EXCEPTION 'lancamento conciliado: nao pode ser alterado'
            USING ERRCODE = '23514';
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$;

-- Restaura o guard do grupo (000105).
CREATE OR REPLACE FUNCTION fin_group_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'conciliado' THEN
            RAISE EXCEPTION 'grupo conciliado: nao pode ser excluido/desagrupado'
                USING ERRCODE = '23514';
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'conciliado'
           AND current_setting('app.fin_group', true) IS DISTINCT FROM 'on' THEN
            RAISE EXCEPTION 'grupo conciliado: nao pode ser alterado'
                USING ERRCODE = '23514';
        END IF;
        NEW.updated_at := now();
        RETURN NEW;
    END IF;

    RETURN NEW;
END;
$$;

DROP INDEX IF EXISTS uq_bank_entries_line;
DROP INDEX IF EXISTS idx_bank_entries_line;

ALTER TABLE bank_statement_entries DROP COLUMN IF EXISTS line_no;
