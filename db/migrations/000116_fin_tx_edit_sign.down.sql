-- 000116_fin_tx_edit_sign.down.sql
-- Volta fin_tx_edit a assinatura de 000096 (sem p_sign). O sinal volta a ser
-- definido apenas na criacao do lancamento.

DROP FUNCTION IF EXISTS fin_tx_edit(
    uuid, text, text, numeric, text, text, text, text, text, boolean,
    text, timestamptz, text, text, text, boolean, smallint);

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

    PERFORM fin_tx_rechain(v_row_tenant);
END;
$$;

GRANT EXECUTE ON FUNCTION fin_tx_edit(
    uuid, text, text, numeric, text, text, text, text, text, boolean,
    text, timestamptz, text, text, text, boolean) TO PUBLIC;
