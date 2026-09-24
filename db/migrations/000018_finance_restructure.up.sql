-- 000018_finance_restructure.up.sql
-- Reestruturacao do modulo financeiro:
--   1) financial_classification_types: tipos de classificacao parametrizaveis
--      para entradas (income) e saidas (expense), por tenant/filial.
--   2) financial_accounts: cadastro de contas bancarias do tenant.
--   3) Associacao de lancamentos (financial_transactions) e recorrencias
--      a uma conta bancaria.
--   4) Atualizacao da funcao de hash-chain para incluir account_id.

-- ---------------------------------------------------------------------------
-- 1) Tipos de classificacao parametrizaveis (entrada | saida)
-- ---------------------------------------------------------------------------
CREATE TABLE financial_classification_types (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id   uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => global do tenant
    direction   text NOT NULL,  -- income | expense
    code        text NOT NULL,
    name        text NOT NULL,
    is_active   boolean NOT NULL DEFAULT true,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, direction, code)
);

CREATE INDEX idx_fin_class_tenant ON financial_classification_types(tenant_id);
CREATE INDEX idx_fin_class_dir    ON financial_classification_types(tenant_id, direction);

ALTER TABLE financial_classification_types ENABLE ROW LEVEL SECURITY;

-- Leitura: sistema, ou tenant do contexto + (branch | global | Sede).
CREATE POLICY fintype_sel ON financial_classification_types
  USING (rls_read(tenant_id, branch_id, true));
-- Escrita: sistema, ou tenant do contexto + branch exato (ou global).
CREATE POLICY fintype_all ON financial_classification_types
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- 2) Contas bancarias
-- ---------------------------------------------------------------------------
CREATE TABLE financial_accounts (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id       uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => global do tenant
    name            text NOT NULL,               -- ex.: "Conta Corrente Principal"
    bank            text,                        -- ex.: "Banco do Brasil"
    bank_code       text,                        -- ex.: "1"
    agency          text,                        -- ex.: "1234"
    account_number  text,                        -- ex.: "56789-0"
    account_type    text NOT NULL DEFAULT 'checking', -- checking | savings | cash
    initial_balance numeric(14,2) NOT NULL DEFAULT 0,
    is_active       boolean NOT NULL DEFAULT true,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_fin_acct_tenant ON financial_accounts(tenant_id);
CREATE INDEX idx_fin_acct_branch ON financial_accounts(tenant_id, branch_id);

ALTER TABLE financial_accounts ENABLE ROW LEVEL SECURITY;

CREATE POLICY finacct_sel ON financial_accounts
  USING (rls_read(tenant_id, branch_id, true));
CREATE POLICY finacct_all ON financial_accounts
  FOR ALL USING (rls_write(tenant_id, branch_id, true))
  WITH CHECK (rls_write(tenant_id, branch_id, true));

-- ---------------------------------------------------------------------------
-- 3) Associacao de lancamentos e recorrencias a contas bancarias
-- ---------------------------------------------------------------------------

-- Lancamentos: associar a conta de origem/destino da movimentacao.
ALTER TABLE financial_transactions
    ADD COLUMN account_id uuid REFERENCES financial_accounts(id) ON DELETE SET NULL;

CREATE INDEX idx_fin_tx_account ON financial_transactions(account_id);

-- Recorrencias: associar conta para os lancamentos automaticos.
ALTER TABLE recurring_donations
    ADD COLUMN account_id uuid REFERENCES financial_accounts(id) ON DELETE SET NULL;

CREATE INDEX idx_recurring_account ON recurring_donations(account_id);

-- ---------------------------------------------------------------------------
-- 4) Atualizacao da funcao de hash-chain (inclui account_id)
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            NEW.payment_method || '|' || COALESCE(NEW.account_id::text,'') || '|' ||
            NEW.occurred_at::text || '|' || COALESCE(NEW.description,'');
    -- Chain cronologico dentro do TENANT (created_at, id) - UUID ordering
    -- produzia cadeia nao-temporal e podia atravessar tenants.
    NEW.prev_hash := (SELECT hash FROM financial_transactions
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

-- ---------------------------------------------------------------------------
-- 5) Seed: tipos de classificacao e contas padrao para o tenant demo
-- ---------------------------------------------------------------------------
INSERT INTO financial_classification_types (tenant_id, branch_id, direction, code, name)
SELECT t.id, NULL, v.direction, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '1.1', 'Dizimo'),
    ('income',  '1.2', 'Oferta'),
    ('income',  '1.3', 'Doacao'),
    ('income',  '1.4', 'Evento'),
    ('expense', '2.1', 'Utilidades'),
    ('expense', '2.2', 'Salario'),
    ('expense', '2.3', 'Acao Social'),
    ('expense', '2.4', 'Midia'),
    ('expense', '2.5', 'Manutencao')
) AS v(direction, code, name)
WHERE t.slug = 'demo'
  AND NOT EXISTS (
      SELECT 1 FROM financial_classification_types fct
      WHERE fct.tenant_id = t.id AND fct.direction = v.direction AND fct.code = v.code
  );

INSERT INTO financial_accounts (tenant_id, branch_id, name, bank, bank_code, agency, account_number, account_type, initial_balance)
SELECT t.id, NULL, v.name, v.bank, v.bank_code, v.agency, v.account_number, v.account_type, v.initial_balance
FROM tenants t
CROSS JOIN (VALUES
    ('Conta Corrente Principal', 'Banco do Brasil', '1', '1234', '56789-0', 'checking', 0),
    ('Caixa', NULL, NULL, NULL, NULL, 'cash', 0)
) AS v(name, bank, bank_code, agency, account_number, account_type, initial_balance)
WHERE t.slug = 'demo'
  AND NOT EXISTS (
      SELECT 1 FROM financial_accounts fa
      WHERE fa.tenant_id = t.id AND fa.name = v.name
  );
