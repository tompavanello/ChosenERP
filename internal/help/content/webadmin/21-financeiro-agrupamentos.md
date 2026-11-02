---
id: financeiro-agrupamentos
app: webadmin
title: Financeiro — agrupamentos
category: Financeiro
summary: Reuna varios lancamentos de um mesmo deposito em uma unica linha bancaria sem perder a verdade contabil.
routes: /dashboard/finance
permissions: finance.read
feature: finance
tags: agrupamento, grupo, deposito, conciliacao, extrato
order: 21
---

# Financeiro — agrupamentos

Um **grupo** representa uma unica movimentacao bancaria (ex.: um deposito) que reune varios lancamentos (dizimos, ofertas, benfeitores).

O grupo **nao e um lancamento financeiro**: os membros continuam sendo a verdade contabil (categoria, doador, recibo e anexos). DRE, balancete e demonstrativos somam os lancamentos individuais.

## Criar um grupo

1. Abra **Financeiro > Agrupamentos**.
2. Clique em **Novo grupo** e selecione os lancamentos.
3. Regras: minimo de 2 membros, mesma filial, conta, tipo e data; nenhum estornado, conciliado ou ja agrupado.
4. A data do grupo e a mesma de todos os membros.
5. Salve.

## Editar e desagrupar

- Alterar a data move todos os membros.
- **Desagrupar** devolve os lancamentos ao fluxo normal.
- Grupo conciliado e imutavel.

## Conciliacao no grupo

A conciliacao e feita no grupo: marca/desmarca todos os membros atomicamente, respeitando periodo encerrado/auditoria fechada. Um lancamento agrupado nao pode ser conciliado isoladamente.

## Extrato

No extrato, a visao **Banco** colapsa o grupo em uma linha (na data do deposito). A visao **Detalhe** mostra um lancamento por linha. O export aceita as duas visoes.

## Erros comuns

- **Soma nao fecha**: confira os valores e a data dos membros.
- **Lancamento conciliado**: desconcilie antes de agrupar.
- **Data diferente**: ajuste manualmente a data dos lancamentos antes de agrupar.
