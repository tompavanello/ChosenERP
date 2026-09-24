-- 000025_legacy_chart_of_accounts.up.sql
-- Etapa 4 - plano de contas no formato do cliente (Docs/exemplos/*.jpeg).
-- Importa os MESMOS codigos do sistema legado:
--   * receitas 101-111;
--   * despesas 1-32;
--   * saldos (501 inicial / 502 final) sao calculados no Demonstrativo Mensal,
--     nao sao contas.
--
-- Aplica-se a todos os tenants existentes (instalacao nova so tem o `demo`).
-- Os valores genericos antigos do seed (1.1, 1.2, ... 2.5) sao desativados para
-- nao poluirem o Demonstrativo; os lancamentos que os usavam sao reapontados
-- para a conta equivalente por melhor correspondencia.

-- ---------------------------------------------------------------------------
-- 1) Receitas (101-111) e 2) Despesas (1-32) em financial_categories
-- ---------------------------------------------------------------------------
INSERT INTO financial_categories (tenant_id, branch_id, type, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '101', 'Dizimos'),
    ('income',  '102', 'Ofertas'),
    ('income',  '103', 'Oferta Missionaria'),
    ('income',  '104', 'Rendimentos de Aplicacoes Financeiras'),
    ('income',  '105', 'Venda de Equipamentos'),
    ('income',  '106', 'Aluguel de Bens Imoveis'),
    ('income',  '107', 'Outras Receitas'),
    ('income',  '108', 'Resgate de Aplicacao Financeira'),
    ('income',  '109', 'Emprestimos'),
    ('income',  '110', 'Receitas Central Kids'),
    ('income',  '111', 'Receitas Diaconia'),
    ('expense', '1',  'Desp. Benfeitorias'),
    ('expense', '2',  'Desp. com Juros'),
    ('expense', '3',  'Desp. com Consumo de Energia Eletrica'),
    ('expense', '4',  'Desp. com Consumo de Agua e Esgoto'),
    ('expense', '5',  'Desp. com Comunicacao'),
    ('expense', '6',  'Desp. com Impostos, Taxas e Contribuicoes'),
    ('expense', '7',  'Desp. com Pastores em Congresso (Congressos Pastorais)'),
    ('expense', '8',  'Desp. com Pastores em Exercicio (Fundo de Garantia)'),
    ('expense', '9',  'Desp. com Pastores em Exercicio (Plano de Saude)'),
    ('expense', '10', 'Desp. com Pastores em Exercicio (Demais Despesas)'),
    ('expense', '11', 'Desp. com Folha de Pagamento da Igreja'),
    ('expense', '12', 'Desp. com Encargos Trabalhistas e Sociais s/ Folha'),
    ('expense', '13', 'Desp. com Ajuda de Custo Permanente p/ Missionarios'),
    ('expense', '14', 'Desp. com Ajuda de Custo Eventual p/ Missionarios e Pregadores'),
    ('expense', '15', 'Desp. com Viagem e Locomocao'),
    ('expense', '16', 'Desp. com Locacao de Bens Imoveis (Ajuda de Custo)'),
    ('expense', '17', 'Desp. com Aquisicao de Insumos, Medicamentos e Outros'),
    ('expense', '18', 'Desp. com o Proibitorio de Botucatu'),
    ('expense', '19', 'Desp. com a Assembleia Geral da IPI'),
    ('expense', '20', 'Desp. com Servicos de Conservacao, Limpeza, Vigilancia e Seguranca'),
    ('expense', '21', 'Desp. com Demais Servicos de Conservacao da Igreja'),
    ('expense', '22', 'Desp. com Materiais e Servicos p/ Reforma dos Predios'),
    ('expense', '23', 'Desp. com Materiais de Limpeza, Conservacao e Manutencao'),
    ('expense', '24', 'Desp. com Reembolso de Gastos do Ministerio de Diaconia'),
    ('expense', '25', 'Desp. com Reembolso de Gastos do Ministerio de Depoimentos'),
    ('expense', '26', 'Desp. com Reembolso de Gastos do Ministerio de Missoes'),
    ('expense', '27', 'Desp. com Confraternizacoes e Eventos Diversos'),
    ('expense', '28', 'Desp. com Aquisicao/Manutencao de Instrumentos, Musicas e Som/Projecao'),
    ('expense', '29', 'Desp. com Materiais e Servicos p/ Ensino Biblico/Teologico'),
    ('expense', '30', 'Despesas com Funeral'),
    ('expense', '31', 'Reserva de Emergencia'),
    ('expense', '32', 'Outras Despesas')
) AS v(type, code, name)
WHERE NOT EXISTS (
    SELECT 1 FROM financial_categories fc
    WHERE fc.tenant_id = t.id AND fc.code = v.code AND fc.type = v.type
);

-- ---------------------------------------------------------------------------
-- 3) Mesmo plano em financial_classification_types (usado pelo formulario)
-- ---------------------------------------------------------------------------
INSERT INTO financial_classification_types (tenant_id, branch_id, direction, code, name)
SELECT t.id, NULL, v.type, v.code, v.name
FROM tenants t
CROSS JOIN (VALUES
    ('income',  '101', 'Dizimos'),
    ('income',  '102', 'Ofertas'),
    ('income',  '103', 'Oferta Missionaria'),
    ('income',  '104', 'Rendimentos de Aplicacoes Financeiras'),
    ('income',  '105', 'Venda de Equipamentos'),
    ('income',  '106', 'Aluguel de Bens Imoveis'),
    ('income',  '107', 'Outras Receitas'),
    ('income',  '108', 'Resgate de Aplicacao Financeira'),
    ('income',  '109', 'Emprestimos'),
    ('income',  '110', 'Receitas Central Kids'),
    ('income',  '111', 'Receitas Diaconia'),
    ('expense', '1',  'Desp. Benfeitorias'),
    ('expense', '2',  'Desp. com Juros'),
    ('expense', '3',  'Desp. com Consumo de Energia Eletrica'),
    ('expense', '4',  'Desp. com Consumo de Agua e Esgoto'),
    ('expense', '5',  'Desp. com Comunicacao'),
    ('expense', '6',  'Desp. com Impostos, Taxas e Contribuicoes'),
    ('expense', '7',  'Desp. com Pastores em Congresso (Congressos Pastorais)'),
    ('expense', '8',  'Desp. com Pastores em Exercicio (Fundo de Garantia)'),
    ('expense', '9',  'Desp. com Pastores em Exercicio (Plano de Saude)'),
    ('expense', '10', 'Desp. com Pastores em Exercicio (Demais Despesas)'),
    ('expense', '11', 'Desp. com Folha de Pagamento da Igreja'),
    ('expense', '12', 'Desp. com Encargos Trabalhistas e Sociais s/ Folha'),
    ('expense', '13', 'Desp. com Ajuda de Custo Permanente p/ Missionarios'),
    ('expense', '14', 'Desp. com Ajuda de Custo Eventual p/ Missionarios e Pregadores'),
    ('expense', '15', 'Desp. com Viagem e Locomocao'),
    ('expense', '16', 'Desp. com Locacao de Bens Imoveis (Ajuda de Custo)'),
    ('expense', '17', 'Desp. com Aquisicao de Insumos, Medicamentos e Outros'),
    ('expense', '18', 'Desp. com o Proibitorio de Botucatu'),
    ('expense', '19', 'Desp. com a Assembleia Geral da IPI'),
    ('expense', '20', 'Desp. com Servicos de Conservacao, Limpeza, Vigilancia e Seguranca'),
    ('expense', '21', 'Desp. com Demais Servicos de Conservacao da Igreja'),
    ('expense', '22', 'Desp. com Materiais e Servicos p/ Reforma dos Predios'),
    ('expense', '23', 'Desp. com Materiais de Limpeza, Conservacao e Manutencao'),
    ('expense', '24', 'Desp. com Reembolso de Gastos do Ministerio de Diaconia'),
    ('expense', '25', 'Desp. com Reembolso de Gastos do Ministerio de Depoimentos'),
    ('expense', '26', 'Desp. com Reembolso de Gastos do Ministerio de Missoes'),
    ('expense', '27', 'Desp. com Confraternizacoes e Eventos Diversos'),
    ('expense', '28', 'Desp. com Aquisicao/Manutencao de Instrumentos, Musicas e Som/Projecao'),
    ('expense', '29', 'Desp. com Materiais e Servicos p/ Ensino Biblico/Teologico'),
    ('expense', '30', 'Despesas com Funeral'),
    ('expense', '31', 'Reserva de Emergencia'),
    ('expense', '32', 'Outras Despesas')
) AS v(type, code, name)
ON CONFLICT (tenant_id, direction, code) DO NOTHING;

-- ---------------------------------------------------------------------------
-- 4) Reaponta os lancamentos que usavam as contas genericas do seed
-- ---------------------------------------------------------------------------
-- financial_transactions e append-only (fin_tx_no_update). Reapontar a conta e
-- uma migracao de dados legitima - nao altera valor/data nem o hash-chain (o
-- category_id nao entra no hash). O guard e suspenso so nesta transacao.
-- Em banco sem lancamentos o UPDATE nao toca linha alguma, mas desabilitar
-- mantem a migracao valida em qualquer base.
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

-- Desativa as contas genericas antigas (mantem a linha por integridade).
UPDATE financial_categories SET is_active = false
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');

UPDATE financial_classification_types SET is_active = false
WHERE code IN ('1.1','1.2','1.3','1.4','2.1','2.2','2.3','2.4','2.5');
