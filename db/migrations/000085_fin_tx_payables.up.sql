-- 000085_fin_tx_payables.up.sql
-- Contas a pagar vs. pagas + centro de custo.
--
-- `due_date` e o vencimento, `paid_at` marca a quitacao (NULL = em aberto) e
-- `cost_center` identifica o ministerio/celula/projeto. Essas colunas ficam
-- FORA da hash-chain e o trigger fin_tx_guard ja permite UPDATE nelas.
ALTER TABLE financial_transactions
    ADD COLUMN due_date     date,
    ADD COLUMN paid_at      timestamptz,
    ADD COLUMN cost_center  text;

-- Apoio ao filtro/relatorio de contas a pagar em aberto.
CREATE INDEX idx_fin_tx_payables ON financial_transactions (tenant_id, branch_id, due_date)
    WHERE paid_at IS NULL AND type = 'expense';
