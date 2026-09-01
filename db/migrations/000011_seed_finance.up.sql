-- 000011_seed_finance.up.sql
-- Plano de contas padrão para o tenant de demonstração.
-- Função run-once para não duplicar se a migração subir após um seed manual.

INSERT INTO financial_categories (tenant_id, branch_id, type, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '1.1', 'Dízimos'),
    ('income',  '1.2', 'Ofertas'),
    ('income',  '1.3', 'Doações'),
    ('income',  '1.4', 'Eventos'),
    ('expense', '2.1', 'Utilidades'),
    ('expense', '2.2', 'Salários'),
    ('expense', '2.3', 'Ação Social'),
    ('expense', '2.4', 'Mídia'),
    ('expense', '2.5', 'Manutenção')
) AS v(type, code, name)
WHERE t.slug = 'demo'
  AND NOT EXISTS (
      SELECT 1 FROM financial_categories fc
      WHERE fc.tenant_id = t.id AND fc.code = v.code
  );
