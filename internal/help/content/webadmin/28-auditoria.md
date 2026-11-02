---
id: auditoria-financeira
app: webadmin
title: Auditoria financeira
category: Financeiro
summary: Marque lancamentos como auditados, feche o periodo com assinatura e exporte o documento imutavel.
routes: /dashboard/reports/audit
permissions: finance.read
feature: finance
tags: auditoria, fechamento, assinatura, documento, periodo
order: 28
---

# Auditoria financeira

A **Auditoria** registra a conferencia formal de um periodo. Depois de fechada, a auditoria trava o intervalo e vira um documento imutavel.

## Criar a auditoria

1. Abra **Relatorios > Auditoria financeira**.
2. Crie a auditoria informando o periodo.
3. O sistema lista os lancamentos do intervalo na ordem oficial.

## Marcar e fechar

1. Marque os lancamentos conferidos (ou use a lista vazia para marcar todos).
2. Clique em **Fechar**. O documento e assinado (hash) e passa a ser imutavel.
3. Exporte em PDF/CSV/XLSX quando necessario.

## Travas

- Auditoria fechada trava INSERT/UPDATE/DELETE no periodo.
- Quando conciliacao e auditoria se sobrepoem, encerre a conciliacao **antes** da auditoria.
- Grupos aparecem como uma linha; a auditoria marca apenas os membros do periodo.

## Excluir

A exclusao da auditoria remove os itens e destrava o periodo (quando aplicavel). Use com cautela e registre o motivo.

## Boas praticas

- Feche a conciliacao antes da auditoria.
- Guarde o PDF assinado com os comprovantes.
- Nao audite periodos ainda em movimento.
