-- 000072_fin_tx_entry_seq.up.sql
-- Sequencia do lancamento: numero de ordem explicito para alinhar os
-- lancamentos com o extrato bancario.
--
-- Escopo da sequencia: (tenant, filial, conta bancaria, data de competencia).
-- Assim cada conta tem sua propria numeracao, que reinicia a cada dia, o que
-- permite inserir um lancamento no meio (esquecido) e renumerar os demais.
--
-- IMPORTANTE: entry_seq e um dado de ORDENACAO, nao financeiro. Fica FORA da
-- hash-chain (fin_tx_hash/fin_tx_rechain) e fora do guard append-only, de modo
-- que pode ser alterado depois (o guard so reclama de colunas listadas). Isso e
-- intencional: reordenar nao muda valor/data/tipo.

ALTER TABLE financial_transactions ADD COLUMN entry_seq integer;

-- Backfill dos lancamentos existentes, numerando por escopo na ordem em que
-- foram criados. O guard append-only nao reclama porque so entry_seq muda, mas
-- a trava de periodo conciliado precisa ser desligada durante o backfill.
ALTER TABLE financial_transactions DISABLE TRIGGER fin_tx_period_lock;

WITH numbered AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY tenant_id, branch_id,
                            COALESCE(account_id, '00000000-0000-0000-0000-000000000000'::uuid),
                            (occurred_at AT TIME ZONE 'UTC')::date
               ORDER BY created_at, id
           ) AS rn
    FROM financial_transactions
)
UPDATE financial_transactions t
SET entry_seq = n.rn
FROM numbered n
WHERE t.id = n.id;

ALTER TABLE financial_transactions ENABLE TRIGGER fin_tx_period_lock;

-- Apoia a ordenacao por escopo. Nao e UNIQUE: um indice funcional em
-- occurred_at::date nao e IMMUTABLE no PostgreSQL; a unicidade e garantida na
-- aplicacao (lock por escopo + renumeracao transacional).
CREATE INDEX idx_fin_tx_entry_seq
    ON financial_transactions (tenant_id, branch_id, account_id, occurred_at, entry_seq);
