-- 000113_fin_tx_reversal.down.sql
-- Remove contra-lancamentos (sign/reversal_of_id/reversal_reason) e restaura as
-- funcoes de hash/guard/grupo da versao anterior (000108).

-- ---------------------------------------------------------------------------
-- 1) Restaura hash/rechain sem sign/reversal.
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
            COALESCE(NEW.donor_name,'');
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
               account_id, occurred_at, description
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
            COALESCE(r.donor_name,'') || '|' || COALESCE(v_prev,''), 'sha256'), 'hex');
        UPDATE financial_transactions SET prev_hash = v_prev, hash = v_hash WHERE id = r.id;
        v_prev := v_hash;
    END LOOP;
    PERFORM set_config('app.fin_rechain', 'off', true);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_rechain(uuid) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- 2) Recadeia a base na formula antiga antes de remover as colunas.
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
               account_id, occurred_at, description, created_at
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
            COALESCE(r.donor_name,'') || '|' || COALESCE(v_prev,''), 'sha256'), 'hex');
        UPDATE financial_transactions SET prev_hash = v_prev, hash = v_hash WHERE id = r.id;
        v_prev := v_hash;
    END LOOP;
END;
$$;

ALTER TABLE financial_transactions ENABLE TRIGGER USER;

-- ---------------------------------------------------------------------------
-- 3) Guard/grupo da versao anterior (000108).
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
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente exclusao e permitida)';
    END IF;
    RETURN NEW;
END;
$$;

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

-- Restaura fin_group_attach sem as colunas novas.
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

-- ---------------------------------------------------------------------------
-- 4) Remove colunas/indices e o seed.
-- ---------------------------------------------------------------------------
DROP INDEX IF EXISTS idx_fin_tx_bank_direction;
DROP INDEX IF EXISTS idx_fin_tx_reversal_of;

ALTER TABLE financial_transactions
    DROP COLUMN IF EXISTS bank_direction,
    DROP COLUMN IF EXISTS reversal_reason,
    DROP COLUMN IF EXISTS reversal_of_id,
    DROP COLUMN IF EXISTS sign;

DELETE FROM financial_categories
WHERE type = 'expense'
  AND code IN ('33','34','35','36')
  AND NOT EXISTS (
      SELECT 1 FROM financial_transactions t WHERE t.category_id = financial_categories.id
  );

DELETE FROM financial_category_groups g
WHERE g.name = 'Devolucoes e Estornos'
  AND NOT EXISTS (
      SELECT 1 FROM financial_categories c WHERE c.group_id = g.id
  );
