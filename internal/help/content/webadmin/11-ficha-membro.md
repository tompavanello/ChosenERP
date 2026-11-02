---
id: ficha-membro
app: webadmin
title: Ficha do membro — abas e rotinas
category: Pessoas
summary: Entenda cada aba da ficha: dados, contato, cargos, familia, vinculos, frequencia, historico, documentos, LGPD e acesso.
routes: /dashboard/members/*
permissions: members.read
tags: ficha, membro, cargos, familia, historico, documentos, lgpd, acesso, frequencia
order: 11
---

# Ficha do membro — abas e rotinas

A ficha reune tudo sobre a pessoa. As abas variam conforme as permissoes do seu perfil.

## Dados

Nome, nome de preferencia, situacao (`Ativo Professo`, `Ativo Nao Professo`, `Inativo`), datas de nascimento, batismo e entrada, sexo, estado civil, CPF e observacoes. Ao inativar, informe o **motivo da saida**.

## Contato

E-mail, telefone, WhatsApp e endereco (com busca por CEP). O e-mail/telefone podem ser usados para o acesso ao app do membro.

## Cargos

Mostra os mandatos do membro (cargo, inicio, fim, situacao). Use **Atribuir cargo** para criar um novo mandato. O historico e preservado; para encerrar, preencha a data final.

## Familia

Cria/edita a familia, define o chefe e vincula parentescos. A familia tambem pode ser gerenciada por aqui, sem tela separada.

## Vinculos

Relacionamentos de discipulado, conjuge, filho, lider etc. A arvore genealogica e de discipulado aparece no detalhe.

## Frequencia

Registra a frequencia do membro. O historico e mantido; o app do membro mostra a aba **Minha frequencia**.

## Historico

Vida eclesiastica (batismo, profissao de fe, transferencia, casamento, etc.). O catalogo de tipos fica em **Configuracoes > Eventos eclesiasticos**. O registro e append-only (nao pode ser apagado), com hash de integridade.

## Documentos

Emite certificados, cartas e declaracoes. O documento pode ser visualizado/impresso e, quando aplicavel, enviado por e-mail/WhatsApp.

## LGPD

Consentimentos, exportacao dos dados do titular e anonimizacao (restrita a Sede).

## Acesso

Cria o acesso ao app do membro:

1. Informe e-mail **ou** telefone e uma senha provisoria.
2. O sistema cria a identidade e vincula `memberships.member_id`.
3. No primeiro acesso, o membro e obrigado a trocar a senha.

Se o membro nao tiver e-mail/telefone, cadastre um contato antes.

## Carteirinha

A carteirinha e idempotente: emitir novamente devolve o mesmo `card_ref`. O QR aponta para uma pagina publica de validacao.
