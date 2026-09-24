-- 000048_txn_supplier.up.sql
-- Permite associar um FORNECEDOR a um lançamento financeiro (tipicamente uma
-- despesa). O vínculo é opcional e como o livro é append-only, a correção
-- continua sendo estorno + relançamento (a coluna não entra no corpo do
-- hash-chain, então a integridade das linhas antigas permanece).
ALTER TABLE financial_transactions
    ADD COLUMN supplier_id uuid REFERENCES suppliers(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_fin_tx_supplier ON financial_transactions(supplier_id);
