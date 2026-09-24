-- 000048_txn_supplier.up.sql
-- Permite associar um FORNECEDOR a um lancamento financeiro (tipicamente uma
-- despesa). O vinculo e opcional e como o livro e append-only, a correcao
-- continua sendo estorno + relancamento (a coluna nao entra no corpo do
-- hash-chain, entao a integridade das linhas antigas permanece).
ALTER TABLE financial_transactions
    ADD COLUMN supplier_id uuid REFERENCES suppliers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_supplier ON financial_transactions(supplier_id);
