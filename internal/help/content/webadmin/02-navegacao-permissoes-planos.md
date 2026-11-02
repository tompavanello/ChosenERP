---
id: navegacao-permissoes-planos
app: webadmin
title: Navegacao, permissoes e planos
category: Primeiros passos
summary: Como funciona o menu, a troca de igreja/filial, as permissoes por perfil e os modulos do plano.
tags: menu, permissao, perfil, plano, filial, igreja, 403
order: 2
---

# Navegacao, permissoes e planos

## Contexto de igreja e filial

- O **seletor de igreja** aparece quando sua identidade tem vinculo com mais de uma igreja.
- O **seletor de filial** aparece para a Sede (`super_admin`/`admin_sede`) e permite trabalhar como:
  - **Sede (todas as filiais)**: leitura ampla e gravacao pela Sede;
  - **uma filial especifica**: leitura e gravacao naquela unidade.
- A escolha da filial fica salva no navegador e acompanha todas as telas.

## Permissoes

Cada rota exige uma permissao, por exemplo:

| Modulo | Permissoes |
|---|---|
| Membros/pessoas | `members.read`, `members.write`, `members.delete` |
| Financeiro | `finance.read`, `finance.write`, `finance.reconcile`, `finance.audit` |
| Relatorios | `reports.read` |
| Ministerios/escalas | `ministries.read`, `ministries.write` |
| Comunicados | `announcements.read`, `announcements.write` |
| Usuarios | `users.read`, `users.write` |
| Configuracoes | `settings.read`, `settings.write` |
| Governanca | `governance.read`, `governance.write` |
| Conteudo | `content.read`, `content.write` |

A Sede (`super_admin`/`admin_sede`) tem todas as permissoes da igreja. O gate de plano continua valendo.

## Modulos do plano

O plano define quais modulos a igreja pode usar (financeiro, relatorios, governanca, ministerios, escalas, eventos, kids, oracao, WhatsApp/comunicados). Quando um modulo esta fora do plano:

- a tela nao aparece no menu;
- a API responde **403 `plan_feature_disabled`**;
- o ideal e falar com o administrador da plataforma para ajustar o plano.

## Erros comuns

- **403 forbidden**: o seu perfil nao tem a permissao. Peca ao administrador da igreja.
- **403 plan_feature_disabled**: o modulo nao esta no plano da igreja.
- **403 tenant_suspended**: a igreja esta suspensa por pendencia financeira.
- **Menu vazio**: confirme a igreja ativa e o perfil do seu acesso.

## Boas praticas

- Trabalhe na filial correta antes de lancar financeiro ou cadastrar membros.
- Nao compartilhe acessos: cada pessoa deve ter o proprio usuario.
- Revise periodicamente **Usuarios e Acessos** para desativar quem saiu da equipe.
