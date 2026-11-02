---
id: meios-pagamento
app: webadmin
title: Meios de pagamento (Asaas)
category: Financeiro
summary: Configure o gateway, crie produtos com categoria contabil e gere cobrancas para os membros.
routes: /dashboard/finance/payments
permissions: payments.read
feature: finance
tags: asaas, gateway, cobranca, pix, boleto, cartao, produto
order: 27
---

# Meios de pagamento (Asaas)

A tela **Meios de pagamento** permite cobrar os membros pela igreja (Fluxo B). Cada produto define a categoria contabil do lancamento gerado quando a cobranca e paga.

## Aba Gateway

1. Informe a chave de API e a URL base do Asaas.
2. Salve. As credenciais sao cifradas e nunca sao devolvidas.
3. O webhook do provedor precisa estar configurado com o token `asaas-access-token`.

## Aba Produtos

1. Crie o produto (nome, valor, tipo de cobranca, categoria contabil e conta).
2. A categoria e obrigatoria: e ela que classifica a receita no financeiro.
3. Ative/desative conforme o uso.

## Aba Cobrancas

1. Selecione o produto e o pagador (membro/benfeitor).
2. Escolha PIX, boleto ou cartao.
3. Gere a cobranca e compartilhe o link/QR.

Quando o gateway confirma o pagamento, o webhook cria o lancamento em `financial_transactions` com a categoria/conta do produto, de forma idempotente.

## Estorno

Use **Estornar** na cobranca. O lancamento financeiro correspondente deve ser tratado conforme a politica da igreja.

## Erros comuns

- **Credenciais invalidas**: revise a chave e o ambiente (sandbox/producao).
- **Webhook nao chega**: confira o token e a URL publica.
- **Produto sem categoria**: o lancamento nao sera gerado corretamente.
