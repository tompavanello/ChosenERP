-- 000026_drop_classification_types.up.sql
-- Remove a tabela REDUNDANTE com o plano de contas.
--
-- `financial_transactions.category_id` sempre apontou para
-- `financial_categories` (o plano de contas). `financial_classification_types`
-- era um catálogo paralelo (entrada/saída) que NÃO era usado nos lançamentos —
-- apenas exibido numa aba própria. Mantê-lo criava duas fontes de verdade para a
-- mesma classificação. Fica só o plano de contas.

DROP TABLE IF EXISTS financial_classification_types;
