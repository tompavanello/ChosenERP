---
id: plataforma-visao-geral
app: webadmin
title: Plataforma — visao geral
category: Plataforma
summary: Painel do operador do SaaS: estatisticas, igrejas, planos, cobranca e backups.
routes: /dashboard/platform
platform: true
tags: plataforma, saas, admin, estatisticas, igrejas
order: 61
---

# Plataforma — visao geral

O **admin da plataforma** e o operador do SaaS. Ele nao tem vinculo com uma igreja e enxerga apenas o console da Plataforma.

## O que aparece

- **Visao geral**: estatisticas gerais (igrejas, planos, totais e recentes).
- **Igrejas**: criar/editar igreja, plano, limites, branding e situacao.
- **Planos**: catalogo de planos e features.
- **Cobranca**: assinaturas e faturas das igrejas.
- **Backups**: pacotes por igreja e restauracao.
- **Suporte**: chamados abertos pela ajuda do produto.

## Responsabilidades

- Onboarding de novas igrejas (tenant + Matriz + papeis).
- Suporte de acesso (criar usuario, redefinir senha, definir administradores).
- Ajuste de plano/limites quando a igreja cresce.
- Acompanhar inadimplencia e suspensao.
- Gerar/restaurar backups.

## Cuidados

- O backup contem hashes de senha e segredos MFA: restrinja o download.
- Restaurar um backup substitui todo o conteudo da igreja.
- Suspender uma igreja bloqueia login/refresh com `tenant_suspended`.
- Nao use a plataforma para operar dados pastorais/financeiros da igreja.

## Ajuda

Use o botao **Ajuda** no topo para o manual da plataforma, IA e chamados.
