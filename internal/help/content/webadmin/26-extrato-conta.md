---
id: extrato-conta
app: webadmin
title: Extrato de conta corrente
category: Financeiro
summary: Consulte o extrato por conta, nas visoes detalhe e banco, respeitando conciliacao e agrupamentos.
routes: /dashboard/reports/account-statement
permissions: finance.read
feature: finance
tags: extrato, conta corrente, banco, detalhe, agrupamento
order: 26
---

# Extrato de conta corrente

O **Extrato** mostra a movimentacao de uma conta no periodo.

## Visoes

- **Detalhe**: um lancamento por linha (padrao contabil).
- **Banco**: um grupo por linha, na data do deposito (visao da conciliacao bancaria).

A visao Banco pode divergir do mes dos lancamentos quando o deposito cai em outro mes. O export aceita a mesma visao selecionada.

## Filtros

- Conta bancaria.
- Periodo.
- Visao (detalhe/banco).

## Exportar

Use **Exportar** para CSV, XLSX ou PDF. O arquivo respeita os filtros e a visao atual.

## Dicas

- Confira o saldo inicial/final antes de fechar o mes.
- Use a visao Banco para conversar com o extrato do banco.
- Use a visao Detalhe para auditoria contabil.
