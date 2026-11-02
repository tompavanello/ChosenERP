---
id: plataforma-planos
app: webadmin
title: Plataforma — planos
category: Plataforma
summary: Gerencie o catalogo de planos, features e limites usados pelas igrejas.
routes: /dashboard/platform/plans
platform: true
tags: plataforma, plano, feature, limite, catalogo
order: 63
---

# Plataforma — planos

O catalogo de **Planos** define preco, ciclo, features e limites.

## Editar um plano

1. Abra **Plataforma > Planos**.
2. Edite nome, descricao, preco, ciclo e limites (`0` = ilimitado; vazio no limite da igreja usa o do plano).
3. Marque/desmarque as features (financeiro, relatorios, governanca, ministerios, escalas, eventos, kids, oracao, WhatsApp).
4. Salve.

## Features

Chave ausente = habilitada. O override por igreja fica em `tenants.features`. O enforcement e no backend (403 `plan_feature_disabled`).

## Gateway

`plans.gateway_refs` guarda o mapeamento para provedores que tem objeto de plano (ex.: MercadoPago). Para o Asaas, a assinatura e criada a partir de `price_cents`/`cycle`.

## Boas praticas

- Nao remova uma feature sem avisar as igrejas afetadas.
- Use limites coerentes com o suporte.
- Teste a cobranca em sandbox antes de mudar preco.
