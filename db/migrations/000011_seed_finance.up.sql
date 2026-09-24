-- 000011_seed_finance.up.sql
-- Plano de contas padrao para o tenant de demonstracao.
-- Funcao run-once para nao duplicar se a migracao subir apos um seed manual.

INSERT INTO financial_categories (tenant_id, branch_id, type, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '1.1', 'Dizimos'),
    ('income',  '1.2', 'Ofertas'),
    ('income',  '1.3', 'Doacoes'),
    ('income',  '1.4', 'Eventos'),
    ('expense', '2.1', 'Utilidades'),
    ('expense', '2.2', 'Salarios'),
    ('expense', '2.3', 'Acao Social'),
    ('expense', '2.4', 'Midia'),
    ('expense', '2.5', 'Manutencao')
) AS v(type, code, name)
WHERE t.slug = 'demo'
  AND NOT EXISTS (
      SELECT 1 FROM financial_categories fc
      WHERE fc.tenant_id = t.id AND fc.code = v.code
  );
