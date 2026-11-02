---
id: configuracoes
app: webadmin
title: Configuracoes da igreja
category: Organizacao
summary: Ajuste dados da igreja, white-label, Pix, filiais, canais de envio e catalogos.
routes: /dashboard/settings
permissions: settings.read
tags: configuracao, igreja, filial, logo, cor, pix, whatsapp, smtp, cargos, eventos
order: 38
tour: administracao-geral
---

# Configuracoes da igreja

A tela **Configuracoes** tem as abas: Dados da igreja, Filiais, Eventos eclesiasticos, Cargos e funcoes e Assinatura.

## Dados da igreja

- Nome, razao social, CNPJ, fuso e idioma.
- Subdominio (slug): o endereco da igreja no app do membro.
- White-label: logo, cor da marca, favicon e dominio proprio.
- Pix: chave e nome do beneficiario exibidos no botao **Contribuir**.

Alterar o slug muda o endereco da igreja; o antigo deixa de funcionar.

## Filiais

Cadastre Matriz, Filiais e PAEs. Cada unidade tem tipo, CNPJ, endereco, situacao e canais proprios.

- **WhatsApp**: conecte a instancia da filial lendo o QR Code.
- **SMTP**: configure o e-mail proprio da filial (a senha nunca e devolvida).
- Sem canal proprio, o envio usa o provedor global.

## Eventos eclesiasticos

Catalogo de tipos de evento da vida do membro (batismo, profissao de fe, transferencia etc.). Define o que cada tipo movimenta no cadastro.

## Cargos e funcoes

Catalogo de cargos, com tipo, exige termo e ordem. Cargos em uso nao podem ser excluidos; desative.

## Assinatura

Mostra o plano e as faturas da igreja. Para mudar plano/forma de pagamento, fale com o suporte da plataforma.

## Boas praticas

- Preencha o Pix antes de divulgar o app.
- Teste o WhatsApp apos conectar.
- Mantenha os catalogos enxutos.
