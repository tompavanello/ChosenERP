---
id: usuarios
app: webadmin
title: Usuarios e acessos
category: Organizacao
summary: Crie acessos da equipe, defina perfis, filial, ative/desative e redefina senhas.
routes: /dashboard/users
permissions: users.read
tags: usuario, acesso, perfil, papel, senha, filial, mfa
order: 37
tour: criar-usuario
---

# Usuarios e acessos

A tela **Usuarios e Acessos** gerencia quem entra no painel e em qual igreja/filial.

## Criar usuario

1. Abra **Organizacao > Usuarios**.
2. Clique em **Novo usuario**.
3. Informe nome, e-mail, perfil (papel) e filial.
4. Defina uma senha provisoria. No primeiro acesso, o usuario troca a senha.

## Editar

- Alterar nome, perfil e filial.
- Ativar/desativar o acesso (prefira desativar a excluir).
- Redefinir a senha (provisoria).

## Perfis

Os perfis definem as permissoes. A Sede (`super_admin`/`admin_sede`) tem todas. Para os demais, revise as permissoes em **Perfis** quando disponivel.

## Seguranca

- Cada pessoa deve ter o proprio acesso.
- Ative **MFA** no perfil pessoal quando disponivel.
- Desative acessos de pessoas que sairam da equipe.
- Nunca compartilhe senhas por WhatsApp/e-mail em texto aberto.

## Erros comuns

- **Limite de usuarios do plano**: remova/desative acessos ou fale com a plataforma.
- **E-mail ja cadastrado**: a identidade e global; use o vinculo com a igreja.
- **Sem permissao**: seu perfil precisa de `users.read`/`users.write`.
