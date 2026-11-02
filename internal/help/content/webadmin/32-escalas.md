---
id: escalas
app: webadmin
title: Escalas de voluntarios
category: Organizacao
summary: Crie escalas, sugira voluntarios, trate conflitos e acompanhe confirmacoes.
routes: /dashboard/rosters
permissions: ministries.read
feature: rosters
tags: escala, voluntario, conflito, confirmacao, evento
order: 32
tour: criar-escala
---

# Escalas de voluntarios

A tela **Escalas** organiza quem serve em cada culto/evento.

## Criar uma escala

1. Abra **Organizacao > Escalas**.
2. Clique em **Nova escala**.
3. Informe ministerio, data/hora, local e, se quiser, crie o evento correspondente (`create_event`).
4. Defina os escalados.

## Sugestao de voluntarios

Use **Sugestoes** informando ministerio e horario. O sistema lista membros e marca conflitos de agenda.

## Confirmacao

- O membro confirma/recusa no app em **Minhas escalas**.
- A tela mostra o status de cada escalado.
- Lembretes podem ser enviados pelas automacoes de comunicacao.

## Conflitos

A aba **Conflitos** mostra choques de horario entre escalas/eventos. Resolva antes de publicar.

## Excluir

Ao excluir uma escala, escolha se o evento gerado tambem deve ser removido (`delete_event=true`). As confirmacoes dos membros sao preservadas quando a escala e editada.

## Boas praticas

- Publique com antecedencia.
- Use sugestoes para distribuir a carga.
- Revise conflitos antes de avisar o time.
