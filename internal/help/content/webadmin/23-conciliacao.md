---
id: conciliacao
app: webadmin
title: Conciliacao financeira
category: Financeiro
summary: Concilie lancamentos individualmente, feche o periodo e entenda as travas de edicao.
routes: /dashboard/finance/reconciliation
permissions: finance.read
feature: finance
tags: conciliacao, periodo, fechamento, auditoria, trava
order: 23
tour: fechar-mes
---

# Conciliacao financeira

A tela **Conciliacao** possui tres abas: **Lancamentos** (conciliacao individual), **Periodos** (fechamento) e **Bancaria** (extrato).

## Conciliacao individual

1. Abra a aba **Lancamentos**.
2. Marque os lancamentos conferidos (individualmente ou em lote).
3. O lancamento conciliado fica **imutavel**: nao pode ser editado/excluido, inclusive `paid_at` e sequencia. Anexos continuam permitidos.

Use o botao **Conciliar** no lancamento ou no lote. Para desfazer, use **Desconciliar**.

## Encerrar o periodo

1. Abra a aba **Periodos** e crie o periodo (data inicial/final).
2. Concilie 100% dos lancamentos nao estornados do intervalo.
3. Clique em **Encerrar/Conciliar periodo**. O sistema sela o hash e trava INSERT/UPDATE/DELETE no intervalo.

A auditoria fechada tambem trava o periodo. Quando os periodos se sobrepoem, encerre a conciliacao **antes** da auditoria.

## Excluir uma conciliacao

A Sede pode excluir a conciliacao de periodo, destravando o intervalo. As conciliacoes individuais ja marcadas sao preservadas.

## Ordem das linhas

Conciliacao, auditoria e ledger listam os lancamentos na mesma ordem: `occurred_at`, `entry_seq`, `created_at`, `id`. O grupo aparece como uma linha com expansao dos membros.

## Erros comuns

- **Periodo encerrado**: nao e possivel lancar/editar no intervalo; reabra excluindo a conciliacao (Sede).
- **Lancamento agrupado**: concilie pelo grupo.
- **Nao fecha 100%**: verifique lancamentos estornados/pendentes e a data.
