---
id: programacao
app: webadmin
title: Programacao (grade recorrente)
category: Organizacao
summary: Cadastre horarios recorrentes e publique as ocorrencias na agenda.
routes: /dashboard/programacao
permissions: members.read
feature: events
tags: programacao, recorrencia, culto, oracao, celula, ebd, ensaio
order: 35
---

# Programacao (grade recorrente)

A **Programacao** define horarios que se repetem (cultos, oracao, celulas, EBD, ensaios, reunioes) e publica as ocorrencias na agenda.

## Cadastrar horario

1. Abra **Organizacao > Programacao**.
2. Clique em **Novo horario**.
3. Escolha o tipo fixo (culto, oracao, celula, EBD, ensaio, reuniao, outro), dia(s) da semana, horario, local e responsavel.
4. Salve.

## Publicar na agenda

1. Clique em **Gerar** e informe o periodo.
2. O sistema cria os eventos no calendario de forma idempotente (nao duplica).
3. Eventos ja publicados permanecem mesmo se o horario for excluido.

## Boas praticas

- Revise a grade no inicio do ano.
- Gere a agenda mensalmente.
- Use tipos consistentes para os relatorios.
