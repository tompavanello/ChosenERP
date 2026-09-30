-- 000071_fin_tx_donor_name.up.sql
-- Doador avulso: identificacao livre (texto) de quem doou sem ser membro nem
-- benfeitor cadastrado. Ex.: "Visitante - oferta de missoes". E o terceiro
-- caminho da identificacao de uma entrada, ao lado de donor_member_id (membro)
-- e benefactor_id (benfeitor); quando nenhum deles existe e/ou o lancamento e
-- anonimo, donor_name fica nulo.
--
-- A coluna entra no corpo do hash-chain (fin_tx_hash/fin_tx_rechain) como as
-- demais, e fica coberta pelo guard append-only (nao pode ser alterada depois).

ALTER TABLE financial_transactions ADD COLUMN donor_name text;

-- ---------------------------------------------------------------------------
-- Recadeamento corrigido: agora encadeia o NOVO hash do antecessor (e nao o
-- hash antigo lido do snapshot). Isso e necessario porque o corpo do hash
-- passou a incluir donor_name; a versao anterior (000061) so funcionava com a
-- formula inalterada. Faz o recalculo sequencial preservando a cadeia.
-- SECURITY DEFINER + validacao do tenant contra a sessao (como em 000061).
-- ---------------------------------------------------------------------------
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
-- Hash do INSERT: inclui donor_name no corpo.
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

-- ---------------------------------------------------------------------------
-- Guard append-only: donor_name tambem nao pode ser alterado.
-- ---------------------------------------------------------------------------
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
