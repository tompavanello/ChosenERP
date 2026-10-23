-- 000104_fin_period_item_locks.up.sql
-- Conciliacao por periodo E por lancamento.
--
-- - financial_transactions ganha reconciled_at/reconciled_by: qualquer lancamento
--   pode ser conciliado sozinho, sem depender de um periodo.
-- - Lancamento conciliado fica imutavel (UPDATE/DELETE bloqueados), exceto pelo
--   recadeamento da hash-chain e pela RPC fin_tx_reconcile.
-- - O encerramento do periodo de conciliacao trava o intervalo inteiro e exige
--   que todos os lancamentos estejam conciliados.
-- - Auditoria fechada tambem trava o periodo (nao aceita novos lancamentos).
-- - Periodos de conciliacao e auditoria nao podem ser fechados em conflito.

-- ---------------------------------------------------------------------------
-- 1) Conciliacao individual
-- ---------------------------------------------------------------------------
ALTER TABLE financial_transactions
    ADD COLUMN IF NOT EXISTS reconciled_at timestamptz,
    ADD COLUMN IF NOT EXISTS reconciled_by uuid REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_reconciled
    ON financial_transactions(tenant_id, branch_id, occurred_at)
    WHERE reconciled_at IS NOT NULL;

-- Preserva o comportamento dos periodos ja encerrados: todos os lancamentos
-- vigentes neles passam a estar conciliados.
ALTER TABLE financial_transactions DISABLE TRIGGER fin_tx_period_lock;
UPDATE financial_transactions t
SET reconciled_at = COALESCE(fr.reconciled_at, fr.created_at, now()),
    reconciled_by = fr.reconciled_by
FROM financial_reconciliations fr
WHERE fr.status = 'conciliada'
  AND fr.tenant_id = t.tenant_id
  AND fr.branch_id = t.branch_id
  AND t.voided_at IS NULL
  AND t.reconciled_at IS NULL
  AND t.occurred_at::date BETWEEN fr.period_start AND fr.period_end;
ALTER TABLE financial_transactions ENABLE TRIGGER fin_tx_period_lock;

-- ---------------------------------------------------------------------------
-- 2) Guard append-only: permite a RPC de conciliacao e protege os novos campos
-- ---------------------------------------------------------------------------
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

-- ---------------------------------------------------------------------------
-- 3) Lancamento conciliado: imutavel, salvo recadeamento/RPC de conciliacao
-- ---------------------------------------------------------------------------
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

DROP TRIGGER IF EXISTS fin_tx_reconciled_lock ON financial_transactions;
CREATE TRIGGER fin_tx_reconciled_lock
BEFORE UPDATE OR DELETE ON financial_transactions
FOR EACH ROW EXECUTE FUNCTION fin_tx_reconciled_lock();

-- ---------------------------------------------------------------------------
-- 4) RPC de conciliacao individual
-- ---------------------------------------------------------------------------
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

-- ---------------------------------------------------------------------------
-- 5) Trava de periodo: conciliacao encerrada OU auditoria fechada
--    Em UPDATE valida OLD e NEW para impedir mover lancamento para dentro/fora.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_period_lock() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    v_old_tenant uuid;
    v_old_branch uuid;
    v_old_date   date;
    v_new_tenant uuid;
    v_new_branch uuid;
    v_new_date   date;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_old_tenant := OLD.tenant_id;
        v_old_branch := OLD.branch_id;
        v_old_date   := OLD.occurred_at::date;
    ELSE
        IF TG_OP = 'UPDATE'
           AND current_setting('app.fin_rechain', true) = 'on'
           AND current_user <> session_user THEN
            RETURN NEW;
        END IF;
        v_new_tenant := NEW.tenant_id;
        v_new_branch := NEW.branch_id;
        v_new_date   := NEW.occurred_at::date;
        IF TG_OP = 'UPDATE' THEN
            v_old_tenant := OLD.tenant_id;
            v_old_branch := OLD.branch_id;
            v_old_date   := OLD.occurred_at::date;
        END IF;
    END IF;

    IF TG_OP IN ('DELETE', 'UPDATE') AND EXISTS (
        SELECT 1 FROM financial_reconciliations fr
        WHERE fr.tenant_id = v_old_tenant
          AND fr.branch_id = v_old_branch
          AND fr.status = 'conciliada'
          AND v_old_date BETWEEN fr.period_start AND fr.period_end
        UNION ALL
        SELECT 1 FROM financial_audits a
        WHERE a.tenant_id = v_old_tenant
          AND a.branch_id = v_old_branch
          AND a.status = 'fechada'
          AND v_old_date BETWEEN a.period_start AND a.period_end
    ) THEN
        RAISE EXCEPTION 'periodo encerrado: lancamentos nao podem ser criados, alterados ou excluidos'
            USING ERRCODE = '23514';
    END IF;

    IF TG_OP IN ('INSERT', 'UPDATE') AND EXISTS (
        SELECT 1 FROM financial_reconciliations fr
        WHERE fr.tenant_id = v_new_tenant
          AND fr.branch_id = v_new_branch
          AND fr.status = 'conciliada'
          AND v_new_date BETWEEN fr.period_start AND fr.period_end
        UNION ALL
        SELECT 1 FROM financial_audits a
        WHERE a.tenant_id = v_new_tenant
          AND a.branch_id = v_new_branch
          AND a.status = 'fechada'
          AND v_new_date BETWEEN a.period_start AND a.period_end
    ) THEN
        RAISE EXCEPTION 'periodo encerrado: lancamentos nao podem ser criados, alterados ou excluidos'
            USING ERRCODE = '23514';
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

-- ---------------------------------------------------------------------------
-- 6) Fechamento de auditoria exige que nao exista conciliacao aberta sobreposta
-- ---------------------------------------------------------------------------
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

    IF NEW.status = 'fechada' AND (TG_OP = 'INSERT' OR OLD.status <> 'fechada') THEN
        SELECT count(*) INTO v_count
        FROM financial_reconciliations fr
        WHERE fr.tenant_id = v_tenant
          AND fr.branch_id = v_branch
          AND fr.status = 'aberta'
          AND daterange(fr.period_start, fr.period_end, '[]')
              && daterange(v_start, v_end, '[]');

        IF v_count > 0 THEN
            RAISE EXCEPTION 'existe conciliacao aberta no periodo: encerre-a antes de fechar a auditoria'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- 7) Conciliacao nao pode ser criada/redimensionada sobre auditoria fechada
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

    IF EXISTS (
        SELECT 1 FROM financial_audits a
        WHERE a.tenant_id = v_tenant
          AND a.branch_id = v_branch
          AND a.status = 'fechada'
          AND daterange(a.period_start, a.period_end, '[]')
              && daterange(v_start, v_end, '[]')
    ) THEN
        RAISE EXCEPTION 'periodo ja encerrado por auditoria'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$$;
