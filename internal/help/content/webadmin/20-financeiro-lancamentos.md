---
id: financeiro-lancamentos
app: webadmin
title: Financeiro — lancamentos
category: Financeiro
summary: Lance dizimos, ofertas, despesas, lote e importacao, com categoria, conta, doador e anexos.
routes: /dashboard/finance
permissions: finance.read
feature: finance
tags: financeiro, lancamento, dizimo, oferta, despesa, lote, importar, recibo
order: 20
tour: lancar-financeiro
---

# Financeiro — lancamentos

A aba **Lancamentos** e o coracao do financeiro. Cada linha e uma movimentacao com categoria, conta, valor, data e, quando aplicavel, doador/fornecedor.

## Conta de trabalho

No topo da aba, escolha a **conta de trabalho**. Ela define:

- os lancamentos exibidos na lista;
- os cards de Entradas, Saidas e Saldo atual;
- a conta usada no lancamento, no lote, na importacao e na edicao.

Para consultar todas as contas, escolha **Todas as contas**. Nesse modo os cards e a lista ficam gerais, mas para lancar/lote/importar e preciso selecionar uma conta. A escolha fica lembrada no navegador.

## Novo lancamento

1. Abra **Financeiro**.
2. Escolha a **conta de trabalho** no topo.
3. Clique em **Novo lancamento**.
4. Escolha o tipo (entrada/saida), a categoria, a data e o valor. A conta bancaria fica travada na conta de trabalho.
5. Para entradas, selecione o membro/benfeitor quando houver.
6. Anexe o comprovante, se necessario.
7. Salve. O recibo e gerado automaticamente para entradas com doador.

## Lancar em lote

Use **Lote** para varias linhas de uma vez. A conta vem do seletor do topo e vale para todas as linhas. Cada linha e validada individualmente; erros nao impedem as demais.

## Importar planilha

1. Escolha a **conta de trabalho** no topo.
2. Clique em **Importar**.
3. Envie CSV/XLSX.
4. Mapeie as colunas e escolha a linha inicial. A coluna de conta bancaria da planilha e ignorada: tudo entra na conta de trabalho.
5. Confirme. O sistema valida categoria/conta e reporta inconsistencias.

## Editar, estornar e excluir

- **Editar**: preserva id, sequencia, recibo e anexos; recalcula o hash. Marque **Inverter o sinal (estorno)** para transformar o lancamento em estorno (ou desmarque para voltar ao normal).
- **Estornar**: o lancamento passa a abater a propria categoria (entrada reduz a receita, saida reduz a despesa) e continua no historico.
- **Excluir**: remocao definitiva, restrita a quem tem permissao. Prefira estornar.

Lancamentos conciliados, auditados ou agrupados possuem travas: nao podem ser editados/excluidos isoladamente.

## Recibo e envio

Abra o lancamento e use **Recibo** para visualizar/imprimir ou **Enviar** por e-mail/WhatsApp. O historico de envios fica disponivel.

## Anexos

Aceita documentos ate 50 MB. O arquivo fica com nome opaco e download autenticado.

## Erros comuns

- **Categoria/conta obrigatoria**: confira o plano de contas e as contas bancarias ativas.
- **Periodo encerrado**: a conciliacao/auditoria travou o intervalo.
- **Lancamento agrupado**: edite pelo grupo ou desagrupe.
- **Limite do plano**: o modulo financeiro pode estar fora do plano.
