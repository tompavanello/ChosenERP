-- 000105_fin_tx_groups.up.sql
-- Agrupamento de lancamentos financeiros.
--
-- Um grupo representa UMA movimentacao bancaria (ex.: um deposito de R$ 1.000)
-- que reune varios lancamentos individuais (dizimos, oferta, benfeitor). O grupo
-- NAO e um lancamento financeiro: os membros continuam sendo a verdade
-- contabil, com categoria, doador, recibo e anexos proprios. Assim os
-- relatorios (DRE, balancete, demonstrativo) continuam somando os membros e
-- nao ha risco de duplicidade.
--
-- A conciliacao e feita no grupo (marca/desmarca todos os membros de uma vez).

CREATE TABLE financial_transaction_groups (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    account_id    uuid NOT NULL REFERENCES financial_accounts(id),
    type          text NOT NULL CHECK (type IN ('income', 'expense')),
    occurred_at   date NOT NULL,
    description   text,
    status        text NOT NULL DEFAULT 'agrupado'
                  CHECK (status IN ('agrupado', 'conciliado')),
    reconciled_at timestamptz,
    reconciled_by uuid,
    bank_entry_id uuid REFERENCES bank_statement_entries(id) ON DELETE SET NULL,
    created_by    uuid,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_fin_grp_tenant
    ON financial_transaction_groups(tenant_id, branch_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_fin_grp_account
    ON financial_transaction_groups(account_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_fin_grp_status
    ON financial_transaction_groups(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_fin_grp_bank_entry
    ON financial_transaction_groups(bank_entry_id)
    WHERE bank_entry_id IS NOT NULL;

ALTER TABLE financial_transaction_groups ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS fintxgrp_sel ON financial_transaction_groups;
CREATE POLICY fintxgrp_sel ON financial_transaction_groups
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));

DROP POLICY IF EXISTS fintxgrp_ins ON financial_transaction_groups;
CREATE POLICY fintxgrp_ins ON financial_transaction_groups
    FOR INSERT WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS fintxgrp_upd ON financial_transaction_groups;
CREATE POLICY fintxgrp_upd ON financial_transaction_groups
    FOR UPDATE USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

DROP POLICY IF EXISTS fintxgrp_del ON financial_transaction_groups;
CREATE POLICY fintxgrp_del ON financial_transaction_groups
    FOR DELETE USING (rls_write(tenant_id, branch_id, false));

-- Vinculo do lancamento com o grupo. ON DELETE SET NULL: excluir o grupo
-- apenas desagrupa os lancamentos (o que a rotina de desagrupar faz).
ALTER TABLE financial_transactions
    ADD COLUMN group_id uuid REFERENCES financial_transaction_groups(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_fin_tx_group
    ON financial_transactions(group_id)
    WHERE group_id IS NOT NULL;

-- Vinculo da entrada do extrato bancario (OFX/CSV) com o grupo.
ALTER TABLE bank_statement_entries
    ADD COLUMN group_id uuid REFERENCES financial_transaction_groups(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bank_entries_group
    ON bank_statement_entries(group_id)
    WHERE group_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 1) Guard do grupo: conciliado vira imutavel.
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

DROP TRIGGER IF EXISTS fin_group_guard ON financial_transaction_groups;
CREATE TRIGGER fin_group_guard
    BEFORE UPDATE OR DELETE ON financial_transaction_groups
    FOR EACH ROW EXECUTE FUNCTION fin_group_guard();

-- ---------------------------------------------------------------------------
-- 2) Consistencia do lancamento com o grupo: mesma filial/conta/tipo e sem
--    estorno. Tambem impede anexar membro a grupo ja conciliado.
-- ---------------------------------------------------------------------------
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

DROP TRIGGER IF EXISTS fin_tx_group_check ON financial_transactions;
CREATE TRIGGER fin_tx_group_check
    BEFORE INSERT OR UPDATE ON financial_transactions
    FOR EACH ROW EXECUTE FUNCTION fin_tx_group_check();

-- ---------------------------------------------------------------------------
-- 3) Guard append-only: group_id so muda pela rotina controlada (app.fin_group).
--    Repete a versao da 000104 acrescentando group_id e o GUC.
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

-- ---------------------------------------------------------------------------
-- 4) Conciliacao individual passa a recusar lancamento agrupado.
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
    v_group uuid;
    v_closed boolean;
BEGIN
    SELECT tenant_id, branch_id, occurred_at::date, voided_at, group_id
    INTO v_row_tenant, v_branch, v_date, v_voided, v_group
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
    IF v_group IS NOT NULL THEN
        RAISE EXCEPTION 'lancamento agrupado: concilie pelo grupo'
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
-- 5) Anexar/desanexar lancamentos do grupo. Precisa ser SECURITY DEFINER para
--    poder setar app.fin_group e passar pelo fin_tx_guard.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_group_attach(
    p_group_id uuid,
    p_tx_ids   uuid[]
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_group_tenant uuid;
    v_group_branch uuid;
    v_status text;
    v_found int;
BEGIN
    SELECT tenant_id, branch_id, status INTO v_group_tenant, v_group_branch, v_status
    FROM financial_transaction_groups WHERE id = p_group_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_group_attach: grupo nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_group_tenant
       OR NOT rls_write(v_group_tenant, v_group_branch, false) THEN
        RAISE EXCEPTION 'fin_group_attach: fora do escopo do usuario';
    END IF;
    IF v_status = 'conciliado' THEN
        RAISE EXCEPTION 'grupo conciliado: nao aceita novos lancamentos'
            USING ERRCODE = '23514';
    END IF;
    IF p_tx_ids IS NULL OR array_length(p_tx_ids, 1) IS NULL THEN
        RETURN;
    END IF;

    SELECT count(*) INTO v_found
    FROM financial_transactions
    WHERE id = ANY(p_tx_ids) AND tenant_id = v_group_tenant;

    IF v_found <> array_length(p_tx_ids, 1) THEN
        RAISE EXCEPTION 'fin_group_attach: lancamento nao encontrado';
    END IF;

    IF EXISTS (
        SELECT 1 FROM financial_transactions t
        WHERE t.id = ANY(p_tx_ids)
          AND (t.group_id IS NOT NULL
               OR t.voided_at IS NOT NULL
               OR t.reconciled_at IS NOT NULL)
    ) THEN
        RAISE EXCEPTION 'lancamento ja agrupado, estornado ou conciliado'
            USING ERRCODE = '23514';
    END IF;

    PERFORM set_config('app.fin_group', 'on', true);
    UPDATE financial_transactions SET group_id = p_group_id WHERE id = ANY(p_tx_ids);
    PERFORM set_config('app.fin_group', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_group_attach(uuid, uuid[]) TO PUBLIC;

CREATE OR REPLACE FUNCTION fin_group_detach(
    p_group_id uuid,
    p_tx_ids   uuid[]
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_group_tenant uuid;
    v_group_branch uuid;
    v_status text;
BEGIN
    SELECT tenant_id, branch_id, status INTO v_group_tenant, v_group_branch, v_status
    FROM financial_transaction_groups WHERE id = p_group_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_group_detach: grupo nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_group_tenant
       OR NOT rls_write(v_group_tenant, v_group_branch, false) THEN
        RAISE EXCEPTION 'fin_group_detach: fora do escopo do usuario';
    END IF;
    IF v_status = 'conciliado' THEN
        RAISE EXCEPTION 'grupo conciliado: nao pode ser alterado'
            USING ERRCODE = '23514';
    END IF;
    IF p_tx_ids IS NULL OR array_length(p_tx_ids, 1) IS NULL THEN
        RETURN;
    END IF;

    PERFORM set_config('app.fin_group', 'on', true);
    UPDATE financial_transactions
    SET group_id = NULL
    WHERE id = ANY(p_tx_ids) AND group_id = p_group_id;
    PERFORM set_config('app.fin_group', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_group_detach(uuid, uuid[]) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- 6) Conciliacao do grupo: marca/desmarca todos os membros atomicamente.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_group_reconcile(
    p_group_id   uuid,
    p_reconciled boolean,
    p_actor      text
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_row_tenant uuid;
    v_row_branch uuid;
    v_status text;
    v_members int;
    v_closed boolean;
    r RECORD;
BEGIN
    SELECT tenant_id, branch_id, status
    INTO v_row_tenant, v_row_branch, v_status
    FROM financial_transaction_groups
    WHERE id = p_group_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_group_reconcile: grupo nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_row_tenant
       OR NOT rls_write(v_row_tenant, v_row_branch, false) THEN
        RAISE EXCEPTION 'fin_group_reconcile: fora do escopo do usuario';
    END IF;

    IF (v_status = 'conciliado') = p_reconciled THEN
        RETURN;
    END IF;

    SELECT count(*) INTO v_members
    FROM financial_transactions
    WHERE group_id = p_group_id AND voided_at IS NULL;

    IF v_members < 2 AND p_reconciled THEN
        RAISE EXCEPTION 'grupo precisa de pelo menos 2 lancamentos'
            USING ERRCODE = '23514';
    END IF;

    FOR r IN
        SELECT branch_id, occurred_at::date AS d
        FROM financial_transactions
        WHERE group_id = p_group_id AND voided_at IS NULL
    LOOP
        SELECT EXISTS (
            SELECT 1 FROM financial_reconciliations fr
            WHERE fr.tenant_id = v_row_tenant
              AND fr.branch_id = r.branch_id
              AND fr.status = 'conciliada'
              AND r.d BETWEEN fr.period_start AND fr.period_end
            UNION ALL
            SELECT 1 FROM financial_audits a
            WHERE a.tenant_id = v_row_tenant
              AND a.branch_id = r.branch_id
              AND a.status = 'fechada'
              AND r.d BETWEEN a.period_start AND a.period_end
        ) INTO v_closed;
        IF v_closed THEN
            RAISE EXCEPTION 'periodo encerrado: conciliacao do grupo nao pode ser alterada'
                USING ERRCODE = '23514';
        END IF;
    END LOOP;

    PERFORM set_config('app.fin_reconcile', 'on', true);
    UPDATE financial_transactions
    SET reconciled_at = CASE WHEN p_reconciled THEN COALESCE(reconciled_at, now()) ELSE NULL END,
        reconciled_by = CASE WHEN p_reconciled THEN COALESCE(reconciled_by, NULLIF(p_actor, '')::uuid) ELSE NULL END
    WHERE group_id = p_group_id AND voided_at IS NULL;
    PERFORM set_config('app.fin_reconcile', 'off', true);

    PERFORM set_config('app.fin_group', 'on', true);
    UPDATE financial_transaction_groups
    SET status = CASE WHEN p_reconciled THEN 'conciliado' ELSE 'agrupado' END,
        reconciled_at = CASE WHEN p_reconciled THEN COALESCE(reconciled_at, now()) ELSE NULL END,
        reconciled_by = CASE WHEN p_reconciled THEN COALESCE(reconciled_by, NULLIF(p_actor, '')::uuid) ELSE NULL END,
        updated_at = now()
    WHERE id = p_group_id;
    PERFORM set_config('app.fin_group', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_group_reconcile(uuid, boolean, text) TO PUBLIC;
