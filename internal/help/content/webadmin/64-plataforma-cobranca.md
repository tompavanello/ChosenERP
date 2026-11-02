---
id: plataforma-cobranca
app: webadmin
title: Plataforma — cobranca (SaaS)
category: Plataforma
summary: Acompanhe assinaturas, faturas, inadimplencia e a suspensao automatica das igrejas.
routes: /dashboard/platform/billing
platform: true
tags: plataforma, cobranca, assinatura, fatura, asaas, inadimplencia, suspensao
order: 64
---

# Plataforma — cobranca (SaaS)

A tela **Cobranca** mostra a saude financeira do SaaS.

## Indicadores

- MRR.
- Faturas vencidas.
- Igrejas suspensas.

## Assinaturas

Lista por igreja, com plano, status e proxima cobranca. Use a ficha da igreja para criar/atualizar/cancelar a assinatura, gerar cobranca avulsa e sincronizar.

## Faturas

Filtre por status, veja o detalhe e estorne quando necessario.

## Suspensao automatica

O worker reconcilia as assinaturas e suspende igrejas inadimplentes apos `PAYMENTS_SUSPEND_AFTER_DAYS`. Igrejas com plano gratuito ou `billing_exempt` nao sao suspensas. A reativacao ocorre ao regularizar.

## Boas praticas

- Confira o webhook e o token do provedor.
- Use o ambiente sandbox para testes.
- Avise a igreja antes de suspender (quando possivel).
