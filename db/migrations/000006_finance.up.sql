-- 000006_finance.up.sql
-- Plano de contas, transacoes (append-only), repasses e documentos

CREATE TABLE financial_categories (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL, -- NULL => global do tenant
    parent_id     uuid REFERENCES financial_categories(id) ON DELETE SET NULL,
    type          text NOT NULL, -- income | expense
    code          text NOT NULL,
    name          text NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Lancamentos financeiros: APPEND-ONLY (imutavel), com hash-chain
CREATE TABLE financial_transactions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    category_id   uuid REFERENCES financial_categories(id) ON DELETE SET NULL,
    type          text NOT NULL, -- income (dizimo | oferta | doacao | recurrente | crowdfunding) | expense
    amount        numeric(14,2) NOT NULL CHECK (amount >= 0),
    currency      text NOT NULL DEFAULT 'BRL',
    donor_member_id uuid REFERENCES members(id) ON DELETE SET NULL,
    benefactor_id uuid REFERENCES benefactors(id) ON DELETE SET NULL,
    payment_method text, -- pix | card | boleto | cash | transfer
    is_anonymous  boolean NOT NULL DEFAULT false,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    description   text,
    receipt_issued boolean NOT NULL DEFAULT false,
    prev_hash     text,
    hash          text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Repasses entre filiais (split de pagamento)
CREATE TABLE transfers (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    from_branch_id uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    to_branch_id  uuid NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    financial_transaction_id uuid REFERENCES financial_transactions(id) ON DELETE SET NULL,
    amount        numeric(14,2) NOT NULL CHECK (amount >= 0),
    rule_name     text, -- ex.: "10% sede", "5% missoes"
    executed_at   timestamptz NOT NULL DEFAULT now()
);

-- Documentos gerados (carteirinhas, certificados, recibos, cartas)
CREATE TABLE documents (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    branch_id     uuid REFERENCES branches(id) ON DELETE SET NULL,
    kind          text NOT NULL, -- membership_card | certificate | transfer_letter | receipt | recommendation
    title         text NOT NULL,
    member_id     uuid REFERENCES members(id) ON DELETE SET NULL,
    document_ref  text,
    qr_token      text,
    file_url      text,
    content       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_fin_tx_tenant ON financial_transactions(tenant_id, branch_id, occurred_at);
CREATE INDEX idx_fin_cat_tenant ON financial_categories(tenant_id);
CREATE INDEX idx_transfers_tenant ON transfers(tenant_id);
CREATE INDEX idx_documents_tenant ON documents(tenant_id);

-- ---------------------------------------------------------------------------
-- Append-only no financeiro + hash-chain
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION fin_tx_guard() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'financial_transactions is append-only (no UPDATE/DELETE)';
END;
$$;

CREATE TRIGGER fin_tx_no_update
BEFORE UPDATE OR DELETE ON financial_transactions
FOR EACH ROW EXECUTE FUNCTION fin_tx_guard();

CREATE OR REPLACE FUNCTION fin_tx_hash() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    body text;
BEGIN
    body := NEW.id::text || '|' || NEW.tenant_id::text || '|' || NEW.branch_id::text || '|' ||
            NEW.type || '|' || NEW.amount::text || '|' || NEW.currency || '|' ||
            COALESCE(NEW.donor_member_id::text,'') || '|' || COALESCE(NEW.benefactor_id::text,'') || '|' ||
            NEW.payment_method || '|' || NEW.occurred_at::text || '|' || COALESCE(NEW.description,'');
    -- Chain cronologico dentro do TENANT (created_at, id) - evita vazamento cross-tenant
    NEW.prev_hash := (SELECT hash FROM financial_transactions
                      WHERE tenant_id = NEW.tenant_id
                        AND (created_at, id) < (NEW.created_at, NEW.id)
                      ORDER BY created_at DESC, id DESC LIMIT 1);
    NEW.hash := encode(digest(body || '|' || COALESCE(NEW.prev_hash,''), 'sha256'), 'hex');
    RETURN NEW;
END;
$$;

CREATE TRIGGER fin_tx_hash
BEFORE INSERT ON financial_transactions
FOR EACH ROW EXECUTE FUNCTION fin_tx_hash();
