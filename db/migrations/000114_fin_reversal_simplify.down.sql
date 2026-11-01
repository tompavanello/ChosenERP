-- 000114_fin_reversal_simplify.down.sql
-- Restaura o seed de categorias de devolucao/estorno removido pela 000113.
-- (O fluxo simplificado por flag nao depende delas.)

INSERT INTO financial_category_groups (tenant_id, branch_id, name, sort_order)
SELECT t.id, NULL, 'Devolucoes e Estornos', 90
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM financial_category_groups g
    WHERE g.tenant_id = t.id AND g.name = 'Devolucoes e Estornos'
);

INSERT INTO financial_categories (tenant_id, branch_id, type, code, name, group_id)
SELECT t.id, NULL, 'expense', v.code, v.name, g.id
FROM tenants t
JOIN financial_category_groups g
  ON g.tenant_id = t.id AND g.name = 'Devolucoes e Estornos'
CROSS JOIN (VALUES
    ('33', 'Devolucao de Cheque'),
    ('34', 'Estorno de Contribuicao'),
    ('35', 'Estorno/Chargeback de Cartao'),
    ('36', 'Outras Devolucoes e Estornos')
) AS v(code, name)
WHERE NOT EXISTS (
    SELECT 1 FROM financial_categories fc
    WHERE fc.tenant_id = t.id AND fc.code = v.code AND fc.type = 'expense'
);
