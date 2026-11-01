-- 000114_fin_reversal_simplify.up.sql
-- O estorno passou a ser um flag no proprio lancamento (sign = -1), sem conta
-- contabil dedicada. Remove o seed de "Devolucoes e Estornos" criado pela
-- 000113, preservando qualquer categoria que ja esteja em uso.

DELETE FROM financial_categories fc
WHERE fc.type = 'expense'
  AND fc.code IN ('33', '34', '35', '36')
  AND NOT EXISTS (
      SELECT 1 FROM financial_transactions t WHERE t.category_id = fc.id
  );

DELETE FROM financial_category_groups g
WHERE g.name = 'Devolucoes e Estornos'
  AND NOT EXISTS (
      SELECT 1 FROM financial_categories c WHERE c.group_id = g.id
  );
