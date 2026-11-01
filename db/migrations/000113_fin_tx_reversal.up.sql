-- 000113_fin_tx_reversal.up.sql
-- Lancamentos de DEVOLUCAO / ESTORNO (contra-lancamento) sem permitir valor
-- negativo na coluna `amount`.
--
-- Regras:
--   * `sign = 1`  -> o lancamento SOMA no total do seu tipo (income/expense).
--   * `sign = -1` -> o lancamento REDUZ o total do seu tipo (contra-receita ou
--                    contra-despesa). Ex.: devolucao de cheque, estorno de
--                    contribuicao, chargeback.
--   * `reversal_of_id` aponta (opcionalmente) para o lancamento original. O
--     vinculo cobre dois modos:
--        - contra-receita: mesma conta de receita, sign = -1;
--        - despesa de devolucao: conta de despesa "Devolucoes/Estornos",
--          sign = 1, tambem podendo referenciar o original.
--   * `bank_direction` e derivado de (type, sign) para a conciliacao bancaria:
--     um debito do extrato casa com um lancamento cuja direcao bancaria e
--     `expense` (inclusive receita com sign = -1).
--
-- `amount` continua positivo (CHECK amount >= 0 e trigger fin_tx_require_account
-- seguem valendo). O sinal e um atributo separado e imutavel.

-- ---------------------------------------------------------------------------
-- 1) Colunas
-- ---------------------------------------------------------------------------
ALTER TABLE financial_transactions
    ADD COLUMN sign smallint NOT NULL DEFAULT 1 CHECK (sign IN (1, -1)),
    ADD COLUMN reversal_of_id uuid REFERENCES financial_transactions(id) ON DELETE RESTRICT,
    ADD COLUMN reversal_reason text;

ALTER TABLE financial_transactions
    ADD COLUMN bank_direction text GENERATED ALWAYS AS (
        CASE WHEN (type = 'income' AND sign = 1) OR (type = 'expense' AND sign = -1)
             THEN 'income' ELSE 'expense' END
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_fin_tx_reversal_of
    ON financial_transactions (tenant_id, reversal_of_id)
    WHERE reversal_of_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_bank_direction
    ON financial_transactions (tenant_id, branch_id, account_id, bank_direction, occurred_at);

-- ---------------------------------------------------------------------------
-- 2) Hash-chain: inclui sign e reversal_of_id no corpo.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            COALESCE(NEW.payment_method,'') || '|' || COALESCE(NEW.account_id::text,'') || '|' ||
            NEW.occurred_at::text || '|' || COALESCE(NEW.description,'') || '|' ||
            COALESCE(NEW.donor_name,'') || '|' ||
            NEW.sign::text || '|' || COALESCE(NEW.reversal_of_id::text,'');
    NEW.prev_hash := (SELECT hash FROM financial_transactions
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION fin_tx_rechain(p_tenant uuid) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    r        record;
    v_prev   text := NULL;
    v_hash   text;
BEGIN
    IF v_tenant IS NULL OR v_tenant <> p_tenant THEN
        RAISE EXCEPTION 'fin_tx_rechain: tenant fora do contexto';
    END IF;

    PERFORM set_config('app.fin_rechain', 'on', true);

    FOR r IN
        SELECT id, tenant_id, branch_id, type, amount, currency,
               donor_member_id, benefactor_id, donor_name, payment_method,
               account_id, occurred_at, description, sign, reversal_of_id
        FROM financial_transactions
        WHERE tenant_id = p_tenant
        ORDER BY created_at, id
    LOOP
        v_hash := encode(digest(
            r.id::text || '|' || r.tenant_id::text || '|' || r.branch_id::text || '|' ||
            r.type || '|' || r.amount::text || '|' || r.currency || '|' ||
            COALESCE(r.donor_member_id::text,'') || '|' || COALESCE(r.benefactor_id::text,'') || '|' ||
            COALESCE(r.payment_method,'') || '|' || COALESCE(r.account_id::text,'') || '|' ||
            r.occurred_at::text || '|' || COALESCE(r.description,'') || '|' ||
            COALESCE(r.donor_name,'') || '|' ||
            r.sign::text || '|' || COALESCE(r.reversal_of_id::text,'') || '|' ||
            COALESCE(v_prev,''), 'sha256'), 'hex');

        UPDATE financial_transactions
        SET prev_hash = v_prev, hash = v_hash
        WHERE id = r.id;

        v_prev := v_hash;
    END LOOP;

    PERFORM set_config('app.fin_rechain', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_rechain(uuid) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- 3) Guard append-only: sign/reversal_of_id/reversal_reason sao imutaveis.
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
       OR NEW.account_transfer_id IS DISTINCT FROM OLD.account_transfer_id
       OR NEW.reconciled_at   IS DISTINCT FROM OLD.reconciled_at
       OR NEW.reconciled_by   IS DISTINCT FROM OLD.reconciled_by
       OR NEW.sign            IS DISTINCT FROM OLD.sign
       OR NEW.reversal_of_id  IS DISTINCT FROM OLD.reversal_of_id
       OR NEW.reversal_reason IS DISTINCT FROM OLD.reversal_reason
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente exclusao e permitida)';
    END IF;
    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- 4) Grupo: devolucao/estorno nao pode ser agrupado (grupos sao depositos).
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

    IF NEW.account_transfer_id IS NOT NULL THEN
        RAISE EXCEPTION 'transferencia entre contas nao pode ser agrupada'
            USING ERRCODE = '23514';
    END IF;

    IF NEW.sign <> 1 OR NEW.reversal_of_id IS NOT NULL THEN
        RAISE EXCEPTION 'devolucao/estorno nao pode ser agrupado'
            USING ERRCODE = '23514';
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

-- Anexar ao grupo tambem recusa devolucao/estorno (defesa em profundidade).
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
               OR t.reconciled_at IS NOT NULL
               OR t.sign <> 1
               OR t.reversal_of_id IS NOT NULL)
    ) THEN
        RAISE EXCEPTION 'lancamento ja agrupado, estornado, conciliado ou e devolucao'
            USING ERRCODE = '23514';
    END IF;

    PERFORM set_config('app.fin_group', 'on', true);
    UPDATE financial_transactions SET group_id = p_group_id WHERE id = ANY(p_tx_ids);
    PERFORM set_config('app.fin_group', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_group_attach(uuid, uuid[]) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- 5) Recadeia a base existente para a nova formula do hash.
--    A migracao roda como dono; desabilita os triggers de usuario e recalcula
--    a cadeia sequencialmente por tenant (created_at, id).
-- ---------------------------------------------------------------------------
ALTER TABLE financial_transactions DISABLE TRIGGER USER;

DO $$
DECLARE
    r        record;
    v_tenant uuid;
    v_prev   text := NULL;
    v_hash   text;
BEGIN
    FOR r IN
        SELECT id, tenant_id, branch_id, type, amount, currency,
               donor_member_id, benefactor_id, donor_name, payment_method,
               account_id, occurred_at, description, sign, reversal_of_id, created_at
        FROM financial_transactions
        ORDER BY tenant_id, created_at, id
    LOOP
        IF v_tenant IS DISTINCT FROM r.tenant_id THEN
            v_tenant := r.tenant_id;
            v_prev := NULL;
        END IF;

        v_hash := encode(digest(
            r.id::text || '|' || r.tenant_id::text || '|' || r.branch_id::text || '|' ||
            r.type || '|' || r.amount::text || '|' || r.currency || '|' ||
            COALESCE(r.donor_member_id::text,'') || '|' || COALESCE(r.benefactor_id::text,'') || '|' ||
            COALESCE(r.payment_method,'') || '|' || COALESCE(r.account_id::text,'') || '|' ||
            r.occurred_at::text || '|' || COALESCE(r.description,'') || '|' ||
            COALESCE(r.donor_name,'') || '|' ||
            r.sign::text || '|' || COALESCE(r.reversal_of_id::text,'') || '|' ||
            COALESCE(v_prev,''), 'sha256'), 'hex');

        UPDATE financial_transactions
        SET prev_hash = v_prev, hash = v_hash
        WHERE id = r.id;

        v_prev := v_hash;
    END LOOP;
END;
$$;

ALTER TABLE financial_transactions ENABLE TRIGGER USER;

-- ---------------------------------------------------------------------------
-- 6) Seed: grupo e contas de despesa para devolucoes/estornos (todos os tenants)
-- ---------------------------------------------------------------------------
INSERT INTO financial_category_groups (tenant_id, branch_id, name, sort_order)
SELECT t.id, NULL, 'Devolucoes e Estornos', 90
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM financial_category_groups g
    WHERE g.tenant_id = t.id AND g.name = 'Devolucoes e Estornos'
);

INSERT INTO financial_categories (tenant_id, branch_id, type, code, name, group_id)
SELECT t.id, NULL, 'expense', v.code, v.name, g.id
FROM tenants t
JOIN financial_category_groups g
  ON g.tenant_id = t.id AND g.name = 'Devolucoes e Estornos'
CROSS JOIN (VALUES
    ('33', 'Devolucao de Cheque'),
    ('34', 'Estorno de Contribuicao'),
    ('35', 'Estorno/Chargeback de Cartao'),
    ('36', 'Outras Devolucoes e Estornos')
) AS v(code, name)
WHERE NOT EXISTS (
    SELECT 1 FROM financial_categories fc
    WHERE fc.tenant_id = t.id AND fc.code = v.code AND fc.type = 'expense'
);
