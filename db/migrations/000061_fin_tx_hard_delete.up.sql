-- 000061_fin_tx_hard_delete.up.sql
-- Exclusao DEFINITIVA de lancamento financeiro (no lugar do estorno).
--
-- Decisao do cliente: ao excluir ou alterar um lancamento o registro sai do
-- banco, nao fica marcado como estornado. Para preservar a trilha de auditoria
-- (hash-chain de financial_transactions) a cadeia do tenant e RECALCULADA na
-- mesma transacao por fin_tx_rechain(), de modo que os lancamentos posteriores
-- voltem a apontar para o hash correto.
--
-- O append-only continua valendo para UPDATE: nenhum campo do lancamento pode
-- ser alterado, exceto quando a propria rotina de recadeamento sinaliza
-- app.fin_rechain = 'on'.

-- ---------------------------------------------------------------------------
-- 1) Guard: permite DELETE; libera UPDATE apenas durante o recadeamento
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        -- Exclusao definitiva: a API valida o escopo (autenticacao + RLS) e a
        -- auditoria fechada e bloqueada no repositorio.
        RETURN OLD;
    END IF;
    -- Recadeamento interno (fin_tx_rechain): so a rotina SECURITY DEFINER pode
    -- reescrever hash/prev_hash. `current_user <> session_user` garante que
    -- estamos dentro da funcao (papel definer) e nao em um SET direto do app.
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

-- ---------------------------------------------------------------------------
-- 2) Recadeamento da hash-chain de um tenant (ordem cronologica created_at, id)
-- ---------------------------------------------------------------------------
-- SECURITY DEFINER: o recadeamento precisa enxergar/atualizar TODOS os
-- lancamentos do tenant, inclusive de filiais fora do escopo do usuario. O
-- parametro e validado contra o tenant da sessao para nao permitir que um
-- usuario force a recadeia de outra igreja.
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

-- A rotina de recadeamento e chamada pelo papel da aplicacao (chosenerp_app).
GRANT EXECUTE ON FUNCTION fin_tx_rechain(uuid) TO PUBLIC;
