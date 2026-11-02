---
id: plataforma-suporte
app: webadmin
title: Plataforma — suporte (chamados)
category: Plataforma
summary: Acompanhe os chamados abertos pela ajuda, responda e altere status/prioridade.
routes: /dashboard/platform/support
platform: true
tags: plataforma, suporte, chamado, ticket, inbox, atendimento
order: 66
---

# Plataforma — suporte (chamados)

O **Suporte** e o inbox dos chamados abertos pelo botao de Ajuda do webadmin e do app do membro.

## Atender

1. Abra **Plataforma > Suporte**.
2. Filtre por status.
3. Abra o chamado: veja tela, app, igreja, usuario, navegador, contexto e transcricao da IA.
4. Responda na thread.
5. Altere status (`aberto`, `em_andamento`, `aguardando`, `resolvido`, `fechado`) e prioridade.

## Diagnostico automatico

O chamado ja chega com:

- rota/tela e app;
- papel do usuario e igreja/filial;
- plano e navegador;
- ultimo erro (quando enviado pelo cliente);
- conversa com a IA.

Isso reduz a ida e volta. Nunca inclua dados sensiveis desnecessarios.

## Notificacoes

Ao abrir um chamado, a API pode notificar `SUPPORT_EMAIL`/`SUPPORT_WHATSAPP` (quando configurados), alem do inbox.

## Boas praticas

- Responda mesmo que seja para pedir mais informacoes.
- Use o status `aguardando` quando depender do usuario.
- Marque como `resolvido` e depois `fechado`.
