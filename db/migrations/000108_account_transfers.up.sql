-- 000108_account_transfers.up.sql
-- Conta do tipo APLICACAO + transferencias entre contas bancarias.
--
-- Uma transferencia entre contas gera DUAS pernas em financial_transactions:
-- uma saida na conta de origem e uma entrada na conta de destino, ambas ligadas
-- ao mesmo account_transfer_id. Assim cada conta tem extrato/saldo corretos e o
-- total geral nao muda (movimentacao interna). As pernas NAO sao receita/despesa
-- de resultado e sao ignoradas pelo DRE/balancete/demonstrativo.

-- ---------------------------------------------------------------------------
-- 1) Tipo de conta: adiciona 'investment' (Aplicacao)
-- ---------------------------------------------------------------------------
UPDATE financial_accounts
SET account_type = 'checking'
WHERE account_type NOT IN ('checking', 'savings', 'cash', 'investment');

ALTER TABLE financial_accounts
    DROP CONSTRAINT IF EXISTS financial_accounts_account_type_check;

ALTER TABLE financial_accounts
    ADD CONSTRAINT financial_accounts_account_type_check
    CHECK (account_type IN ('checking', 'savings', 'cash', 'investment'));

-- ---------------------------------------------------------------------------
-- 2) Transferencias entre contas
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS account_transfers (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    from_account_id uuid NOT NULL REFERENCES financial_accounts(id) ON DELETE RESTRICT,
    to_account_id   uuid NOT NULL REFERENCES financial_accounts(id) ON DELETE RESTRICT,
    kind            text NOT NULL CHECK (kind IN ('deposit', 'redeem', 'transfer')),
    amount          numeric(14,2) NOT NULL CHECK (amount > 0),
    occurred_at     date NOT NULL,
    description     text,
    created_by      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CHECK (from_account_id <> to_account_id)
);

CREATE INDEX IF NOT EXISTS idx_acct_transfer_tenant
    ON account_transfers(tenant_id, branch_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_acct_transfer_from ON account_transfers(from_account_id);
CREATE INDEX IF NOT EXISTS idx_acct_transfer_to ON account_transfers(to_account_id);

ALTER TABLE account_transfers ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS acct_transfer_sel ON account_transfers;
CREATE POLICY acct_transfer_sel ON account_transfers
    FOR SELECT USING (rls_read(tenant_id, branch_id, false));

DROP POLICY IF EXISTS acct_transfer_all ON account_transfers;
CREATE POLICY acct_transfer_all ON account_transfers
    FOR ALL USING (rls_write(tenant_id, branch_id, false))
    WITH CHECK (rls_write(tenant_id, branch_id, false));

-- ---------------------------------------------------------------------------
-- 3) Vinculo das pernas com a transferencia
-- ---------------------------------------------------------------------------
ALTER TABLE financial_transactions
    ADD COLUMN IF NOT EXISTS account_transfer_id uuid
        REFERENCES account_transfers(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_fin_tx_account_transfer
    ON financial_transactions(account_transfer_id)
    WHERE account_transfer_id IS NOT NULL;

-- ---------------------------------------------------------------------------
-- 4) Guard append-only: account_transfer_id nao pode ser alterado por fora.
--    (mesma versao da 000105 acrescentando a coluna)
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

-- ---------------------------------------------------------------------------
-- 5) Consistencia do grupo: perna de transferencia nao pode ser agrupada.
--    (mesma versao da 000106 acrescentando a regra)
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
