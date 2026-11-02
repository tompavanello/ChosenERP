# Manual do Usuario — Chosen ERP

Este diretorio e o indice do **Manual do Usuario** exibido dentro do produto.

A fonte oficial do conteudo fica em:

```
internal/help/content/
  webadmin/   artigos do painel administrativo e do console da plataforma
  member/     artigos do app do membro e do visitante
```

Cada artigo e um Markdown com frontmatter (id, titulo, categoria, rotas,
permissoes, modulo do plano, tags e tour). O binario Go embute esses arquivos
(`internal/help`) e os serve pela API:

- `GET /api/v1/help/articles`
- `GET /api/v1/help/articles/{slug}`
- `GET /api/v1/help/context?app=&path=`
- `POST /api/v1/help/chat`
- `POST /api/v1/help/feedback`

O mesmo conteudo alimenta:

- o botao **Ajuda** no topo do webadmin e do app do membro;
- a Central de Ajuda (`/dashboard/help` no webadmin e `/ajuda` no membro);
- a busca com normalizacao de acentos;
- o contexto enviado para a IA;
- o **Guia de uso** de primeiro acesso (modal + checklist + tours).

## O que o manual cobre

- Primeiros passos, navegacao, permissoes e planos.
- Pessoas: membros, ficha, visitantes, solicitacoes, area do membro, benfeitores
  e fornecedores.
- Financeiro: lancamentos, agrupamentos, contas/categorias, conciliacao,
  conciliacao bancaria, transferencias entre contas, extrato, meios de pagamento,
  auditoria e repasses.
- Organizacao: ministerios/grupos, materiais, escalas, kids, eventos,
  programacao, governanca, usuarios, configuracoes e comunicados.
- Conteudo: devocionais, videos, culto ao vivo e Biblia.
- Relatorios.
- Console da plataforma: visao geral, igrejas, planos, cobranca, backups e
  suporte.
- App do membro: inicio, agenda, avisos, oracao, perfil, contribuir, devocional,
  Biblia, videos, ao vivo, escalas, GD/ministerios, solicitacoes,
  aniversariantes, frequencia e area do lider.
- Visitante e perguntas frequentes.

## Como editar

1. Altere o Markdown em `internal/help/content/<app>/`.
2. Mantenha o frontmatter valido.
3. Rode `go test ./internal/help -count=1` e `go build ./internal/...`.
4. Ajuste a rota/permissao/modulo quando a tela mudar.

## Ajuda com IA e chamados

- A IA usa **somente** os artigos visiveis para o perfil/plano + a rota atual.
- Quando nao ha base suficiente, a resposta orienta abrir um chamado.
- O chamado entra no inbox da plataforma (`/dashboard/platform/support`) e pode
  notificar `SUPPORT_EMAIL`/`SUPPORT_WHATSAPP`.
- A IA da ajuda usa `HELP_AI_*` e, se vazio, cai para `AI_*`.

## Imagens

Os artigos usam a convencao `![Tela: ...](TODO)` para placeholders. As imagens
podem ser adicionadas depois em `apps/webadmin/public/help/` e referenciadas no
Markdown.
