-- 000116_fin_tx_edit_sign.up.sql
-- Permite inverter o sinal (estorno) na edicao de um lancamento.
--
-- Antes, o flag de estorno (sign) so podia ser definido na criacao: a edicao em
-- vigor (000096) nao recebia o sinal e recusava lancamentos ja estornados. Agora
-- fin_tx_edit recebe p_sign e grava o sinal junto com os demais campos, entao o
-- usuario pode marcar/desmarcar "Inverter o sinal (estorno)" ao editar.
--
-- O guard append-only continua bloqueando UPDATE direto do papel da aplicacao;
-- a alteracao so acontece dentro da rotina SECURITY DEFINER (app.fin_edit). A
-- hash-chain e recalculada por fin_tx_rechain, que ja inclui sign no corpo, e
-- bank_direction e coluna gerada (acompanha o novo sinal automaticamente).

DROP FUNCTION IF EXISTS fin_tx_edit(
    uuid, text, text, numeric, text, text, text, text, text, boolean,
    text, timestamptz, text, text, text, boolean);

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
    p_paid            boolean,
    p_sign            smallint
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
    IF p_sign IS NULL OR p_sign NOT IN (1, -1) THEN
        RAISE EXCEPTION 'fin_tx_edit: sinal invalido (use 1 ou -1)'
            USING ERRCODE = '23514';
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
        paid_at         = CASE WHEN p_paid THEN COALESCE(paid_at, now()) ELSE NULL END,
        sign            = p_sign
    WHERE id = p_id;

    PERFORM set_config('app.fin_edit', 'off', true);

    -- O corpo do hash mudou (inclusive o sinal): recalcula a cadeia do tenant.
    PERFORM fin_tx_rechain(v_row_tenant);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_edit(
    uuid, text, text, numeric, text, text, text, text, text, boolean,
    text, timestamptz, text, text, text, boolean, smallint) TO PUBLIC;
