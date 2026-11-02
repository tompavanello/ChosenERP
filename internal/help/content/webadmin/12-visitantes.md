---
id: visitantes
app: webadmin
title: Visitantes — acolhimento e conversao
category: Pessoas
summary: Acompanhe a trilha de acolhimento, avance etapas e converta visitantes em membros.
routes: /dashboard/visitors
permissions: visitors.read
tags: visitante, acolhimento, trilha, conversao, membro
order: 12
tour: cadastrar-visitante
---

# Visitantes — acolhimento e conversao

A tela **Visitantes** registra quem chegou e acompanha o acolhimento ate a decisao de membresia.

## Registrar um visitante

1. Abra **Pessoas > Visitantes**.
2. Clique em **Novo visitante** e informe nome, telefone e origem (quando houver).
3. Salve. O visitante entra na trilha de acolhimento.

O proprio visitante tambem pode entrar pelo app do membro usando o codigo de 6 digitos enviado por WhatsApp. Nesse caso, o cadastro nasce pelo app.

## Trilha de acolhimento

- Cada mudanca de etapa gera um registro imutavel em `visitor_journey` (hash-chain).
- Use **Trilha** para ver o historico completo.
- Etapas comuns: primeiro contato, visita, acompanhamento, discipulado, decisao.

## Converter em membro

1. Abra o visitante.
2. Clique em **Converter** e complete os dados necessarios.
3. O sistema cria o membro e, quando houver pedido "quero ser membro", promove o mesmo acesso de visitante para membro.

Se a conversao foi feita por engano, use **Reverter** (quando disponivel) para voltar ao estado anterior.

## Boas praticas

- Registre o contato no mesmo dia da visita.
- Use a trilha para nao perder o acompanhamento.
- Ao converter, revise e-mail/telefone para o acesso ao app.
