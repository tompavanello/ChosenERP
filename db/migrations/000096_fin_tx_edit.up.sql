-- 000096_fin_tx_edit.up.sql
-- Edicao de lancamento financeiro EM VIGOR (sem excluir/recriar).
--
-- Antes, "editar" era excluir + recriar: o lancamento perdia o id, o recibo, os
-- anexos, a data de criacao e, sobretudo, a posicao (entry_seq) - podendo ir
-- para o fim do escopo. Agora o update acontece na propria linha, preservando
-- id/entry_seq/recibo/anexos. A hash-chain e recalculada por fin_tx_rechain,
-- o mesmo mecanismo ja usado na exclusao definitiva.
--
-- O guard append-only continua valendo para UPDATE direto do papel da aplicacao.
-- So a rotina SECURITY DEFINER fin_tx_edit (abaixo) pode reescrever os campos do
-- lancamento. A trava de periodo conciliado (fin_tx_period_lock) NAO e
-- afrouxada: editar lancamento de periodo conciliado continua bloqueado.

-- ---------------------------------------------------------------------------
-- 1) Guard append-only: libera UPDATE somente durante o recadeamento ou dentro
--    da rotina de edicao (app.fin_edit), ambas SECURITY DEFINER.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    IF current_user <> session_user
       AND (current_setting('app.fin_rechain', true) = 'on'
            OR current_setting('app.fin_edit', true) = 'on') THEN
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
       OR NEW.hash            IS DISTINCT FROM OLD.hash
       OR NEW.prev_hash       IS DISTINCT FROM OLD.prev_hash
    THEN
        RAISE EXCEPTION 'financial_transactions is append-only (somente exclusao e permitida)';
    END IF;
    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- 2) Edicao em vigor. Atualiza os campos do lancamento e reencadeia o tenant.
--    entry_seq NAO muda (posicao preservada); reposicionar e feito por
--    POST /finance/transactions/reorder.
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_edit(
    p_id              uuid,
    p_category_id     text,
    p_type            text,
    p_amount          numeric,
    p_payment_method  text,
    p_donor_member_id text,
    p_benefactor_id   text,
    p_donor_name      text,
    p_supplier_id     text,
    p_is_anonymous    boolean,
    p_description     text,
    p_occurred_at     timestamptz,
    p_account_id      text,
    p_due_date        text,
    p_cost_center     text,
    p_paid            boolean
) RETURNS void
LANGUAGE plpgsql SECURITY DEFINER SET search_path = public AS $$
DECLARE
    v_tenant      uuid := NULLIF(current_setting('app.tenant_id', true), '')::uuid;
    v_row_tenant  uuid;
BEGIN
    SELECT tenant_id INTO v_row_tenant
    FROM financial_transactions WHERE id = p_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'fin_tx_edit: lancamento nao encontrado';
    END IF;
    IF v_tenant IS NULL OR v_tenant <> v_row_tenant THEN
        RAISE EXCEPTION 'fin_tx_edit: tenant fora do contexto';
    END IF;

    PERFORM set_config('app.fin_edit', 'on', true);

    UPDATE financial_transactions SET
        category_id     = NULLIF(p_category_id, '')::uuid,
        type            = p_type,
        amount          = p_amount,
        payment_method  = NULLIF(p_payment_method, ''),
        donor_member_id = NULLIF(p_donor_member_id, '')::uuid,
        benefactor_id   = NULLIF(p_benefactor_id, '')::uuid,
        donor_name      = NULLIF(p_donor_name, ''),
        supplier_id     = NULLIF(p_supplier_id, '')::uuid,
        is_anonymous    = p_is_anonymous,
        description     = NULLIF(p_description, ''),
        occurred_at     = p_occurred_at,
        account_id      = NULLIF(p_account_id, '')::uuid,
        due_date        = NULLIF(p_due_date, '')::date,
        cost_center     = NULLIF(p_cost_center, ''),
        paid_at         = CASE WHEN p_paid THEN COALESCE(paid_at, now()) ELSE NULL END
    WHERE id = p_id;

    PERFORM set_config('app.fin_edit', 'off', true);

    -- O corpo do hash mudou: recalcula a cadeia do tenant inteiro.
    PERFORM fin_tx_rechain(v_row_tenant);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_edit(
    uuid, text, text, numeric, text, text, text, text, text, boolean,
    text, timestamptz, text, text, text, boolean) TO PUBLIC;

-- ---------------------------------------------------------------------------
-- 3) Recálculo do split de repasses exige apagar os repasses antigos da
--    transacao (transfers so tinha policy de SELECT/INSERT).
-- ---------------------------------------------------------------------------
DROP POLICY IF EXISTS transfers_del ON transfers;
CREATE POLICY transfers_del ON transfers FOR DELETE
  USING (rls_write(tenant_id, from_branch_id, false) OR rls_hq(tenant_id));
