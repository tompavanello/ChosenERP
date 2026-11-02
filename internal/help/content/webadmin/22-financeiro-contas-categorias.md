---
id: financeiro-contas-categorias
app: webadmin
title: Financeiro — contas, plano de contas e grupos
category: Financeiro
summary: Configure contas bancarias, categorias e grupos de contas.
routes: /dashboard/finance
permissions: finance.read
feature: finance
tags: conta bancaria, plano de contas, categoria, grupo de contas, split
order: 22
---

# Financeiro — contas, plano de contas e grupos

## Contas bancarias

Cadastre cada conta (banco, agencia, numero, tipo). O saldo e calculado pelos lancamentos. Contas em uso nao podem ser excluidas; desative.

## Plano de contas

Categorias de entrada e saida organizam o DRE e os relatorios. Cada categoria pertence a um grupo e pode ter uma conta padrao.

- Crie categorias com codigo e nome claros.
- Nao exclua categoria em uso (409); desative.
- Mantenha a mesma nomenclatura ao longo do ano para comparar periodos.

## Grupos de contas

Agrupam categorias para relatorios gerenciais (ex.: "Receitas correntes", "Despesas administrativas"). Ao excluir um grupo, as contas ficam sem grupo.

## Split / repasses

Em **Repasses**, defina regras de distribuicao entre filiais. O split e aplicado quando habilitado, respeitando o percentual por filial de destino.

## Boas praticas

- Comece com um plano de contas enxuto; detalhe depois.
- Nao renomeie categorias frequentemente.
- Revise contas ativas antes de lancar.
