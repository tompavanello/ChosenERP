-- 000106_fin_group_same_date_seq.up.sql
-- Agrupamentos passam a exigir que TODOS os membros tenham a mesma data do
-- grupo (a data da movimentacao bancaria). Tambem ganham entry_seq proprio,
-- para aparecerem como uma linha reordenavel junto dos lancamentos avulsos.
--
-- O entry_seq do grupo nasce da posicao do primeiro membro (MIN) no escopo
-- filial+conta+data; sem membro na data, vai para o fim do escopo.

ALTER TABLE financial_transaction_groups
    ADD COLUMN IF NOT EXISTS entry_seq integer;

CREATE INDEX IF NOT EXISTS idx_fin_grp_entry_seq
    ON financial_transaction_groups(tenant_id, branch_id, account_id, occurred_at, entry_seq);

-- Backfill dos grupos existentes (feature recem-criada). Usa a data do grupo
-- em UTC, igual ao escopo do entry_seq dos lancamentos.
UPDATE financial_transaction_groups g
SET entry_seq = COALESCE(
    (SELECT MIN(t.entry_seq)
       FROM financial_transactions t
      WHERE t.group_id = g.id
        AND t.voided_at IS NULL
        AND (t.occurred_at AT TIME ZONE 'UTC')::date = g.occurred_at),
    (SELECT MAX(x.seq) + 1
       FROM (
            SELECT t.entry_seq AS seq
              FROM financial_transactions t
             WHERE t.tenant_id = g.tenant_id
               AND t.branch_id = g.branch_id
               AND t.account_id = g.account_id
               AND t.group_id IS NULL
               AND (t.occurred_at AT TIME ZONE 'UTC')::date = g.occurred_at
            UNION ALL
            SELECT other.entry_seq AS seq
              FROM financial_transaction_groups other
             WHERE other.tenant_id = g.tenant_id
               AND other.branch_id = g.branch_id
               AND other.account_id = g.account_id
               AND other.occurred_at = g.occurred_at
               AND other.id <> g.id
       ) x),
    1
);

-- ---------------------------------------------------------------------------
-- Limpeza defensiva: grupos legados com membros em datas diferentes nao
-- satisfazem o novo invariante. Desagrupa e remove o grupo (preserva os
-- lancamentos, valores, datas e a conciliacao individual de cada um).
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    PERFORM set_config('app.fin_group', 'on', true);
    PERFORM set_config('app.fin_reconcile', 'on', true);

    CREATE TEMP TABLE _fin_mixed_groups ON COMMIT DROP AS
        SELECT g.id
          FROM financial_transaction_groups g
         WHERE EXISTS (
               SELECT 1
                 FROM financial_transactions t
                WHERE t.group_id = g.id
                  AND (t.occurred_at AT TIME ZONE 'UTC')::date <> g.occurred_at
         );

    -- O cleanup precisa mexer em group_id de lancamentos (coluna protegida pelo
    -- fin_tx_guard) e remover grupos conciliados. Roda como o papel migrador,
    -- entao desliga os triggers apenas nesta transacao.
    PERFORM set_config('session_replication_role', 'replica', true);

    UPDATE financial_transactions t
       SET group_id = NULL
      FROM _fin_mixed_groups m
     WHERE t.group_id = m.id;

    UPDATE bank_statement_entries b
       SET group_id = NULL
      FROM _fin_mixed_groups m
     WHERE b.group_id = m.id;

    UPDATE financial_transaction_groups g
       SET status = 'agrupado',
           reconciled_at = NULL,
           reconciled_by = NULL
      FROM _fin_mixed_groups m
     WHERE g.id = m.id;

    DELETE FROM financial_transaction_groups g
     USING _fin_mixed_groups m
     WHERE g.id = m.id;

    PERFORM set_config('session_replication_role', 'origin', true);
    DROP TABLE _fin_mixed_groups;

    PERFORM set_config('app.fin_reconcile', 'off', true);
    PERFORM set_config('app.fin_group', 'off', true);
END;
$$;

-- ---------------------------------------------------------------------------
-- Move a data do grupo e de todos os membros de uma vez. Precisa ser
-- SECURITY DEFINER para poder atualizar occurred_at (hash-chain) e setar o GUC
-- que o fin_tx_guard exige. O entry_seq novo ja vem calculado pelo Go.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_group_move_date(
    p_group_id   uuid,
    p_date       date,
    p_entry_seq  integer
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant    uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_row_tenant uuid;
    v_branch    uuid;
    v_status    text;
    v_bank      uuid;
BEGIN
    SELECT tenant_id, branch_id, status, bank_entry_id
    INTO v_row_tenant, v_branch, v_status, v_bank
    FROM financial_transaction_groups
    WHERE id = p_group_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_group_move_date: grupo nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_row_tenant
       OR NOT rls_write(v_row_tenant, v_branch, false) THEN
        RAISE EXCEPTION 'fin_group_move_date: fora do escopo do usuario';
    END IF;
    IF v_status = 'conciliado' THEN
        RAISE EXCEPTION 'grupo conciliado: nao pode ser alterado'
            USING ERRCODE = '23514';
    END IF;
    IF v_bank IS NOT NULL THEN
        RAISE EXCEPTION 'grupo vinculado ao extrato: a data do deposito nao pode ser alterada'
            USING ERRCODE = '23514';
    END IF;

    UPDATE financial_transaction_groups
    SET occurred_at = p_date,
        entry_seq = p_entry_seq
    WHERE id = p_group_id;

    PERFORM set_config('app.fin_group', 'on', true);
    UPDATE financial_transactions
    SET occurred_at = p_date::timestamp AT TIME ZONE 'UTC'
    WHERE group_id = p_group_id;
    PERFORM set_config('app.fin_group', 'off', true);

    -- occurred_at entra no hash: recalcula a cadeia do tenant.
    PERFORM fin_tx_rechain(v_row_tenant);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_group_move_date(uuid, date, integer) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- Invariante no banco: ao final da transacao, todo membro nao estornado tem a
-- mesma data do grupo. E deferido para permitir criar/mover o grupo e os
-- membros na mesma transacao.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_group_same_date_check() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM financial_transactions t
         WHERE t.group_id = NEW.id
           AND t.voided_at IS NULL
           AND (t.occurred_at AT TIME ZONE 'UTC')::date <> NEW.occurred_at
    ) THEN
        RAISE EXCEPTION 'grupo e lancamentos precisam ter a mesma data'
            USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS fin_group_same_date_check ON financial_transaction_groups;
CREATE CONSTRAINT TRIGGER fin_group_same_date_check
    AFTER INSERT OR UPDATE OF occurred_at ON financial_transaction_groups
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION fin_group_same_date_check();

-- ---------------------------------------------------------------------------
-- Consistencia do lancamento com o grupo: mesma filial/conta/tipo/DATA e sem
-- estorno. Tambem impede anexar membro a grupo ja conciliado.
-- ---------------------------------------------------------------------------
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
