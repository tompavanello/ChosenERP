-- 000107_bank_entry_sequence.up.sql
-- Sequencia do arquivo bancario + reordenacao que preserva a conciliacao.
--
-- 1) bank_statement_entries.line_no guarda a ordem da linha dentro do arquivo
--    (OFX/CSV). Ela permite gerar os lancamentos na mesma sequencia do extrato.
-- 2) fin_lines_set_seq muda SOMENTE o entry_seq (que nao entra no hash) de um
--    lancamento ou grupo, inclusive de linhas ja conciliadas. O periodo
--    encerrado continua bloqueando a alteracao.
--
-- O GUC app.fin_reorder e usado apenas pela RPC; ele NAO libera o
-- fin_tx_period_lock (diferente de app.fin_rechain), portanto um periodo
-- encerrado continua imutavel.

ALTER TABLE bank_statement_entries
    ADD COLUMN IF NOT EXISTS line_no integer NOT NULL DEFAULT 0;

-- Backfill: ordem cronologica dentro de cada importacao (as importacoes antigas
-- nao tinham a ordem original do arquivo).
WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY import_id
               ORDER BY posted_at, created_at, id
           ) AS rn
    FROM bank_statement_entries
)
UPDATE bank_statement_entries e
SET line_no = ranked.rn
FROM ranked
WHERE e.id = ranked.id
  AND e.line_no = 0;

CREATE INDEX IF NOT EXISTS idx_bank_entries_line
    ON bank_statement_entries(import_id, posted_at, line_no);

CREATE UNIQUE INDEX IF NOT EXISTS uq_bank_entries_line
    ON bank_statement_entries(import_id, line_no)
    WHERE line_no > 0;

-- ---------------------------------------------------------------------------
-- Guard do lancamento conciliado: permite a reordenacao controlada.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_reconciled_lock() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_reconcile', true) = 'on'
            OR current_setting('app.fin_reorder', true) = 'on') THEN
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

-- ---------------------------------------------------------------------------
-- Guard do grupo: permite a reordenacao controlada do grupo conciliado.
-- ---------------------------------------------------------------------------
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
           AND current_setting('app.fin_group', true) IS DISTINCT FROM 'on'
           AND current_setting('app.fin_reorder', true) IS DISTINCT FROM 'on' THEN
            RAISE EXCEPTION 'grupo conciliado: nao pode ser alterado'
                USING ERRCODE = '23514';
        END IF;
        NEW.updated_at := now();
        RETURN NEW;
    END IF;

    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- Muda apenas a posicao (entry_seq) de uma linha visivel da sequencia.
-- kind: transaction | group. Respeita RLS e periodo encerrado.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_lines_set_seq(
    p_kind text,
    p_id   uuid,
    p_seq  integer
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant  uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_kind    text := lower(coalesce(p_kind, ''));
    v_row_tenant uuid;
    v_branch  uuid;
    v_date    date;
    v_closed  boolean;
BEGIN
    IF p_seq IS NULL OR p_seq < 0 THEN
        RAISE EXCEPTION 'fin_lines_set_seq: posicao invalida';
    END IF;

    IF v_kind IN ('transaction', 'tx', 'lancamento') THEN
        v_kind := 'transaction';
        SELECT tenant_id, branch_id, (occurred_at AT TIME ZONE 'UTC')::date
        INTO v_row_tenant, v_branch, v_date
        FROM financial_transactions
        WHERE id = p_id;
    ELSIF v_kind IN ('group', 'grupo', 'agrupamento') THEN
        v_kind := 'group';
        SELECT tenant_id, branch_id, occurred_at
        INTO v_row_tenant, v_branch, v_date
        FROM financial_transaction_groups
        WHERE id = p_id;
    ELSE
        RAISE EXCEPTION 'fin_lines_set_seq: tipo de linha invalido';
    END IF;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_lines_set_seq: linha nao encontrada';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_row_tenant
       OR NOT rls_write(v_row_tenant, v_branch, false) THEN
        RAISE EXCEPTION 'fin_lines_set_seq: fora do escopo do usuario';
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
        RAISE EXCEPTION 'periodo encerrado: nao pode reordenar'
            USING ERRCODE = '23514';
    END IF;

    PERFORM set_config('app.fin_reorder', 'on', true);
    IF v_kind = 'transaction' THEN
        UPDATE financial_transactions SET entry_seq = p_seq WHERE id = p_id;
    ELSE
        UPDATE financial_transaction_groups SET entry_seq = p_seq WHERE id = p_id;
    END IF;
    PERFORM set_config('app.fin_reorder', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_lines_set_seq(text, uuid, integer) TO PUBLIC;
