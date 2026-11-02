---
id: relatorio-balancete
app: webadmin
title: Balancete mensal
category: Relatorios
summary: Entradas, saidas e saldo por mes, com exportacao.
routes: /dashboard/reports/balance
permissions: reports.read
feature: reports
tags: balancete, mensal, saldo, entradas, saidas, exportar
order: 51
---

# Balancete mensal

O **Balancete** mostra entradas, saidas e saldo por mes.

## Usar

1. Abra **Relatorios > Balancete mensal**.
2. Escolha o periodo.
3. Confira a serie mensal e os totais.
4. Exporte em CSV, XLSX ou PDF.

## Regras

- Respeita o contexto de filial.
- Lancamentos estornados nao entram.
- A conciliacao nao bloqueia a leitura, mas relatorios definitivos devem ser gerados apos o fechamento.

## Boas praticas

- Compare com o extrato bancario.
- Feche a conciliacao antes de apresentar o balancete.
- Guarde o PDF do mes aprovado.
