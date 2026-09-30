-- 000071_fin_tx_donor_name.down.sql
-- Reverte o doador avulso: restaura as funcoes de hash/guard/recadeamento ao
-- estado anterior (000027 para o hash, 000061 para rechain/guard) e remove a
-- coluna donor_name.

CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            COALESCE(NEW.payment_method,'') || '|' || COALESCE(NEW.account_id::text,'') || '|' ||
            NEW.occurred_at::text || '|' || COALESCE(NEW.description,'');
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
BEGIN
    IF v_tenant IS NULL OR v_tenant <> p_tenant THEN
        RAISE EXCEPTION 'fin_tx_rechain: tenant fora do contexto';
    END IF;

    PERFORM set_config('app.fin_rechain', 'on', true);

    WITH ordered AS (
        SELECT id,
               lag(hash) OVER (ORDER BY created_at, id) AS prev
        FROM financial_transactions
        WHERE tenant_id = p_tenant
    )
    UPDATE financial_transactions t
    SET prev_hash = o.prev,
        hash = encode(digest(
            t.id::text || '|' || t.tenant_id::text || '|' || t.branch_id::text || '|' ||
            t.type || '|' || t.amount::text || '|' || t.currency || '|' ||
            COALESCE(t.donor_member_id::text,'') || '|' || COALESCE(t.benefactor_id::text,'') || '|' ||
            COALESCE(t.payment_method,'') || '|' || COALESCE(t.account_id::text,'') || '|' ||
            t.occurred_at::text || '|' || COALESCE(t.description,'') || '|' ||
            COALESCE(o.prev,''), 'sha256'), 'hex')
    FROM ordered o
    WHERE t.id = o.id;

    PERFORM set_config('app.fin_rechain', 'off', true);
END;
$$;

CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    IF current_setting('app.fin_rechain', true) = 'on'
       AND current_user <> session_user THEN
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

ALTER TABLE financial_transactions DROP COLUMN donor_name;
