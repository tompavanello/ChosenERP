---
id: membros
app: webadmin
title: Membros — cadastro, busca e importacao
category: Pessoas
summary: Como cadastrar, localizar, importar e manter o rol de membros da igreja.
routes: /dashboard/members
permissions: members.read
tags: membros, cadastro, rol, importar, planilha, busca, foto
order: 10
tour: cadastrar-membro
---

# Membros — cadastro, busca e importacao

A tela **Membros** e a porta de entrada do rol da igreja. Cada membro tem uma ficha completa com dados pessoais, contato, cargos, familia, historico, documentos, frequencia e acesso ao app.

## Cadastrar um membro

1. Abra **Pessoas > Membros**.
2. Clique em **Novo membro** (ou no atalho equivalente).
3. Preencha os dados obrigatorios (nome e situacao) e, se possivel, e-mail/telefone.
4. Salve. A ficha do membro sera aberta para completar as demais abas.

## Buscar e filtrar

- Use a busca por nome, e-mail, telefone ou CPF.
- Os filtros respeitam o contexto de filial: a Sede ve todas as unidades; a filial ve apenas a propria e as sub-congregacoes.
- A lista mostra situacao, foto, contato e acoes rapidas.

## Importar em lote (CSV/XLSX)

1. Clique em **Importar**.
2. Envie a planilha (CSV ou XLSX).
3. O sistema mostra as primeiras linhas para **mapear as colunas**.
4. Escolha a linha inicial e associe cada coluna ao campo correspondente.
5. Confirme a importacao.

Regras e cuidados:

- A importacao respeita a **quota de membros** do plano.
- Linhas invalidas sao reportadas; corrija a planilha e reimporte apenas o necessario.
- Evite duplicidade: confira e-mail/CPF/telefone antes de importar.

## Acoes da lista

- Abrir a ficha completa.
- Emitir carteirinha digital (QR).
- Transferir para outra filial do mesmo tenant.
- Exportar dados do titular (LGPD), quando permitido.

## Excluir um membro

A exclusao definitiva e restrita a Sede. Prefira **inativar** e registrar o motivo da saida no historico, preservando o rastro pastoral e financeiro.

## Erros comuns

- **Limite do plano atingido**: a quota de membros acabou; fale com o administrador da plataforma.
- **Sem permissao**: o perfil precisa de `members.read`/`members.write`.
- **Membro nao aparece**: confira a filial ativa e se ele esta em uma sub-congregacao.
