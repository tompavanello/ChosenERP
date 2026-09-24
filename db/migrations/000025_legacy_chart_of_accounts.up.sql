-- 000025_legacy_chart_of_accounts.up.sql
-- Etapa 4 — plano de contas no formato do cliente (Docs/exemplos/*.jpeg).
-- Importa os MESMOS códigos do sistema legado:
--   * receitas 101–111;
--   * despesas 1–32;
--   * saldos (501 inicial / 502 final) são calculados no Demonstrativo Mensal,
--     não são contas.
--
-- Aplica-se a todos os tenants existentes (instalação nova só tem o `demo`).
-- Os valores genéricos antigos do seed (1.1, 1.2, ... 2.5) são desativados para
-- não poluirem o Demonstrativo; os lançamentos que os usavam são reapontados
-- para a conta equivalente por melhor correspondência.

-- ---------------------------------------------------------------------------
-- 1) Receitas (101–111) e 2) Despesas (1–32) em financial_categories
-- ---------------------------------------------------------------------------
INSERT INTO financial_categories (tenant_id, branch_id, type, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '101', 'Dízimos'),
    ('income',  '102', 'Ofertas'),
    ('income',  '103', 'Oferta Missionária'),
    ('income',  '104', 'Rendimentos de Aplicações Financeiras'),
    ('income',  '105', 'Venda de Equipamentos'),
    ('income',  '106', 'Aluguel de Bens Imóveis'),
    ('income',  '107', 'Outras Receitas'),
    ('income',  '108', 'Resgate de Aplicação Financeira'),
    ('income',  '109', 'Empréstimos'),
    ('income',  '110', 'Receitas Central Kids'),
    ('income',  '111', 'Receitas Diaconia'),
    ('expense', '1',  'Desp. Benfeitorias'),
    ('expense', '2',  'Desp. com Juros'),
    ('expense', '3',  'Desp. com Consumo de Energia Elétrica'),
    ('expense', '4',  'Desp. com Consumo de Água e Esgoto'),
    ('expense', '5',  'Desp. com Comunicação'),
    ('expense', '6',  'Desp. com Impostos, Taxas e Contribuições'),
    ('expense', '7',  'Desp. com Pastores em Congresso (Congressos Pastorais)'),
    ('expense', '8',  'Desp. com Pastores em Exercício (Fundo de Garantia)'),
    ('expense', '9',  'Desp. com Pastores em Exercício (Plano de Saúde)'),
    ('expense', '10', 'Desp. com Pastores em Exercício (Demais Despesas)'),
    ('expense', '11', 'Desp. com Folha de Pagamento da Igreja'),
    ('expense', '12', 'Desp. com Encargos Trabalhistas e Sociais s/ Folha'),
    ('expense', '13', 'Desp. com Ajuda de Custo Permanente p/ Missionários'),
    ('expense', '14', 'Desp. com Ajuda de Custo Eventual p/ Missionários e Pregadores'),
    ('expense', '15', 'Desp. com Viagem e Locomoção'),
    ('expense', '16', 'Desp. com Locação de Bens Imóveis (Ajuda de Custo)'),
    ('expense', '17', 'Desp. com Aquisição de Insumos, Medicamentos e Outros'),
    ('expense', '18', 'Desp. com o Proibitório de Botucatu'),
    ('expense', '19', 'Desp. com a Assembleia Geral da IPI'),
    ('expense', '20', 'Desp. com Serviços de Conservação, Limpeza, Vigilância e Segurança'),
    ('expense', '21', 'Desp. com Demais Serviços de Conservação da Igreja'),
    ('expense', '22', 'Desp. com Materiais e Serviços p/ Reforma dos Prédios'),
    ('expense', '23', 'Desp. com Materiais de Limpeza, Conservação e Manutenção'),
    ('expense', '24', 'Desp. com Reembolso de Gastos do Ministério de Diaconia'),
    ('expense', '25', 'Desp. com Reembolso de Gastos do Ministério de Depoimentos'),
    ('expense', '26', 'Desp. com Reembolso de Gastos do Ministério de Missões'),
    ('expense', '27', 'Desp. com Confraternizações e Eventos Diversos'),
    ('expense', '28', 'Desp. com Aquisição/Manutenção de Instrumentos, Músicas e Som/Projeção'),
    ('expense', '29', 'Desp. com Materiais e Serviços p/ Ensino Bíblico/Teológico'),
    ('expense', '30', 'Despesas com Funeral'),
    ('expense', '31', 'Reserva de Emergência'),
    ('expense', '32', 'Outras Despesas')
) AS v(type, code, name)
WHERE NOT EXISTS (
    SELECT 1 FROM financial_categories fc
    WHERE fc.tenant_id = t.id AND fc.code = v.code AND fc.type = v.type
);

-- ---------------------------------------------------------------------------
-- 3) Mesmo plano em financial_classification_types (usado pelo formulário)
-- ---------------------------------------------------------------------------
INSERT INTO financial_classification_types (tenant_id, branch_id, direction, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '101', 'Dízimos'),
    ('income',  '102', 'Ofertas'),
    ('income',  '103', 'Oferta Missionária'),
    ('income',  '104', 'Rendimentos de Aplicações Financeiras'),
    ('income',  '105', 'Venda de Equipamentos'),
    ('income',  '106', 'Aluguel de Bens Imóveis'),
    ('income',  '107', 'Outras Receitas'),
    ('income',  '108', 'Resgate de Aplicação Financeira'),
    ('income',  '109', 'Empréstimos'),
    ('income',  '110', 'Receitas Central Kids'),
    ('income',  '111', 'Receitas Diaconia'),
    ('expense', '1',  'Desp. Benfeitorias'),
    ('expense', '2',  'Desp. com Juros'),
    ('expense', '3',  'Desp. com Consumo de Energia Elétrica'),
    ('expense', '4',  'Desp. com Consumo de Água e Esgoto'),
    ('expense', '5',  'Desp. com Comunicação'),
    ('expense', '6',  'Desp. com Impostos, Taxas e Contribuições'),
    ('expense', '7',  'Desp. com Pastores em Congresso (Congressos Pastorais)'),
    ('expense', '8',  'Desp. com Pastores em Exercício (Fundo de Garantia)'),
    ('expense', '9',  'Desp. com Pastores em Exercício (Plano de Saúde)'),
    ('expense', '10', 'Desp. com Pastores em Exercício (Demais Despesas)'),
    ('expense', '11', 'Desp. com Folha de Pagamento da Igreja'),
    ('expense', '12', 'Desp. com Encargos Trabalhistas e Sociais s/ Folha'),
    ('expense', '13', 'Desp. com Ajuda de Custo Permanente p/ Missionários'),
    ('expense', '14', 'Desp. com Ajuda de Custo Eventual p/ Missionários e Pregadores'),
    ('expense', '15', 'Desp. com Viagem e Locomoção'),
    ('expense', '16', 'Desp. com Locação de Bens Imóveis (Ajuda de Custo)'),
    ('expense', '17', 'Desp. com Aquisição de Insumos, Medicamentos e Outros'),
    ('expense', '18', 'Desp. com o Proibitório de Botucatu'),
    ('expense', '19', 'Desp. com a Assembleia Geral da IPI'),
    ('expense', '20', 'Desp. com Serviços de Conservação, Limpeza, Vigilância e Segurança'),
    ('expense', '21', 'Desp. com Demais Serviços de Conservação da Igreja'),
    ('expense', '22', 'Desp. com Materiais e Serviços p/ Reforma dos Prédios'),
    ('expense', '23', 'Desp. com Materiais de Limpeza, Conservação e Manutenção'),
    ('expense', '24', 'Desp. com Reembolso de Gastos do Ministério de Diaconia'),
    ('expense', '25', 'Desp. com Reembolso de Gastos do Ministério de Depoimentos'),
    ('expense', '26', 'Desp. com Reembolso de Gastos do Ministério de Missões'),
    ('expense', '27', 'Desp. com Confraternizações e Eventos Diversos'),
    ('expense', '28', 'Desp. com Aquisição/Manutenção de Instrumentos, Músicas e Som/Projeção'),
    ('expense', '29', 'Desp. com Materiais e Serviços p/ Ensino Bíblico/Teológico'),
    ('expense', '30', 'Despesas com Funeral'),
    ('expense', '31', 'Reserva de Emergência'),
    ('expense', '32', 'Outras Despesas')
) AS v(type, code, name)
ON CONFLICT (tenant_id, direction, code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4) Reaponta os lançamentos que usavam as contas genéricas do seed
-- ---------------------------------------------------------------------------
-- financial_transactions é append-only (fin_tx_no_update). Reapontar a conta é
-- uma migração de dados legítima — não altera valor/data nem o hash-chain (o
-- category_id não entra no hash). O guard é suspenso só nesta transação.
-- Em banco sem lançamentos o UPDATE não toca linha alguma, mas desabilitar
-- mantém a migração válida em qualquer base.
ALTER TABLE financial_transactions DISABLE TRIGGER fin_tx_no_update;

WITH mapa(old_code, old_type, new_code, new_type) AS (VALUES
    ('1.1', 'income',  '101', 'income'),
    ('1.2', 'income',  '102', 'income'),
    ('1.3', 'income',  '107', 'income'),
    ('1.4', 'income',  '107', 'income'),
    ('2.1', 'expense', '3',   'expense'),
    ('2.2', 'expense', '11',  'expense'),
    ('2.3', 'expense', '19',  'expense'),
    ('2.4', 'expense', '5',   'expense'),
    ('2.5', 'expense', '20',  'expense')
)
UPDATE financial_transactions t
SET category_id = novo.id
FROM financial_categories antigo
JOIN mapa m ON m.old_code = antigo.code AND m.old_type = antigo.type
JOIN financial_categories novo
  ON novo.tenant_id = antigo.tenant_id AND novo.code = m.new_code AND novo.type = m.new_type
WHERE t.category_id = antigo.id;

ALTER TABLE financial_transactions ENABLE TRIGGER fin_tx_no_update;

-- Desativa as contas genéricas antigas (mantém a linha por integridade).
UPDATE financial_categories SET is_active = false
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');

UPDATE financial_classification_types SET is_active = false
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');
