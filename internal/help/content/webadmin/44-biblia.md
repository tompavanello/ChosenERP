---
id: conteudo-biblia
app: webadmin
title: Biblia (configuracao)
category: Conteudo
summary: Defina a versao padrao da Biblia exibida no app do membro.
routes: /dashboard/content/bible
permissions: content.read
tags: biblia, versao, api.bible, leitor
order: 44
---

# Biblia (configuracao)

## Configurar

1. Abra **Conteudo > Biblia**.
2. Escolha a versao padrao disponivel.
3. Salve.

O leitor usa a API licenciada `api.bible` com cache local. Sem a chave `API_BIBLE_KEY`, o leitor fica indisponivel.

## Listar versoes

Use a listagem de versoes para conferir o que a licenca permite.

## Boas praticas

- Use uma versao licenciada para a igreja.
- Avise os membros qual versao e a padrao.
- Se o leitor estiver indisponivel, confira a chave da API no servidor.
