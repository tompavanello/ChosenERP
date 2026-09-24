-- 000026_drop_classification_types.up.sql
-- Remove a tabela REDUNDANTE com o plano de contas.
--
-- `financial_transactions.category_id` sempre apontou para
-- `financial_categories` (o plano de contas). `financial_classification_types`
-- era um catalogo paralelo (entrada/saida) que NAO era usado nos lancamentos -
-- apenas exibido numa aba propria. Mante-lo criava duas fontes de verdade para a
-- mesma classificacao. Fica so o plano de contas.

DROP TABLE IF EXISTS financial_classification_types;
