---
id: transferencias-contas
app: webadmin
title: Transferencias entre contas
category: Financeiro
summary: Registre depositos e resgates entre conta corrente e aplicacao sem duplicar receita/despesa.
routes: /dashboard/finance/account-transfers
permissions: finance.read
feature: finance
tags: transferencia, aplicacao, resgate, conta corrente, movimento interno
order: 25
---

# Transferencias entre contas

Movimentacoes internas (conta corrente para aplicacao e vice-versa) nao sao receita nem despesa: sao transferencias.

## Registrar

1. Abra **Financeiro > Transferencias entre contas**.
2. Informe a conta de origem, a conta de destino, a data e o valor.
3. Descreva o motivo (ex.: aplicacao automatica, resgate).
4. Salve.

O sistema gera as duas pontas no extrato, mantendo o saldo consolidado correto.

## Excluir

A exclusao desfaz as duas pontas. Use quando o registro estiver errado; nao use para "corrigir" um valor — edite/exclua e recrie.

## Boas praticas

- Nunca lance transferencia como receita/despesa.
- Use contas do tipo aplicacao para separar o saldo.
- Confira a data no extrato bancario.
