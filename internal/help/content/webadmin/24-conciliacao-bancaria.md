---
id: conciliacao-bancaria
app: webadmin
title: Conciliacao bancaria (extrato OFX/CSV)
category: Financeiro
summary: Importe o extrato, case as linhas com os lancamentos, gere pendentes e agrupe depositos.
routes: /dashboard/finance/reconciliation, /dashboard/finance/bank-reconciliation
permissions: finance.read
feature: finance
tags: extrato, ofx, csv, banco, conciliacao, importar
order: 24
---

# Conciliacao bancaria (extrato OFX/CSV)

A aba **Bancaria** importa o extrato do banco e ajuda a casar cada linha com o financeiro.

## Importar o extrato

1. Abra **Conciliacao > Bancaria**.
2. Selecione a conta e envie o arquivo OFX ou CSV.
3. O sistema le data, valor, descricao e identificador (FITID), evitando duplicidade.

## Conciliar

- Cada linha mostra o status (pendente, casada, ignorada).
- Use **Sugestoes de sequencia** para ordenar as linhas como no extrato.
- **Gerar lancamento** cria o lancamento a partir da linha.
- **Gerar pendentes** cria em lote para linhas sem correspondencia.
- **Ignorar** marca a linha como fora do controle (ex.: tarifa ja lancada).

## Agrupar deposito

Use **Agrupar** quando a linha do banco for um deposito que reune varios lancamentos. Regras:

- a soma dos membros deve bater com a linha;
- os lancamentos precisam estar na **data do deposito**;
- se a data divergir, o sistema recusa e pede o ajuste manual.

## Boas praticas

- Importe o extrato do mes inteiro antes de fechar o periodo.
- Trate todas as linhas (conciliadas ou ignoradas).
- Guarde o arquivo original para auditoria.

## Erros comuns

- **Linha duplicada**: o FITID ja foi importado.
- **Conta errada**: confira a conta selecionada.
- **Soma nao bate**: ajuste os lancamentos ou a data.
