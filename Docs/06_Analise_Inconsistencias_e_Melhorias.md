# Chosen ERP - Analise de Inconsistencias e Melhorias

> Documento de diagnostico. **Nenhum codigo foi alterado** para produzir esta analise.
> Data: 2026-09-30. Base: `master` (commit `6ce4f1e`).

---

## 1. Escopo e metodo

Foram revisadas, em modo somente-leitura, as seguintes areas:

| Area | Escopo |
|---|---|
| Backend HTTP | `internal/httpapi/*` (rotas, middleware, handlers, rate limit, export) |
| Dominios Go | `internal/*` (auth, members, finance, org, store, delivery, audit, ...) |
| Banco | `db/migrations/*` (000001-000066), `db/init`, `internal/store/migrate.go` |
| Frontend | `apps/webadmin` (Next.js 15, App Router, Tailwind v4) |
| Site | `apps/marketing` + `POST /api/v1/public/leads` |
| Infra | `infra/*`, `start.ps1`, `.env.example`, `.gitignore`, `internal/config` |
| Docs | `README.md`, `AGENTS.md`, `CLAUDE.md`, `Docs/*` |

Metodo: leitura direta do codigo + verificacao dos achados de maior severidade
contra a fonte (arquivo e linha). Os itens marcados com **[verificado]** foram
confirmados por inspecao direta nesta analise; os demais vem de varredura
sistematica e devem ser confirmados antes da correcao.

### Resumo executivo

| Severidade | Qtd | Tema dominante |
|---|---|---|
| Critico | 4 | Exposicao de anexos financeiros; dinheiro em `float64`; cadeia de auditoria quebrada; endpoint `/people` inexistente no front |
| Alto | 9 | Autorizacao ausente/inconsistente; wipe do catalogo de permissoes; workers sem lock; segredos e defaults divergentes |
| Medio | 64 | Numeracao de migracoes, autorizacao/paginacao no backend, drift de env, site institucional, permisssoes no front, docs |
| Baixo | 15 | Higiene, dead code, acessibilidade, polimento |
| **Total** | **92** | |

A distribuicao de Medio e: banco/migracoes 9 (`DB-3..DB-11`), backend 15
(`BE-1..BE-15`), configuracao/infra 12 (`CFG-3..CFG-14`), site 10
(`MKT-1..MKT-10`), webadmin 12 (`FE-2..FE-13`), documentacao 6 (`DOC-2..DOC-7`).

Os tres riscos que eu corrigiria primeiro: **SEC-1** (anexo financeiro publico),
**FIN-1** (dinheiro em ponto flutuante) e **AUTH-1** (mutacoes financeiras sem
verificacao de permissao). Os tres sao exploraveis ou afetam correcao contabil
sem exigir pre-requisitos.

---

## 2. Critico

### SEC-1 - `GET /api/v1/attachments/{filename}` e publico e serve comprovante financeiro **[verificado]**

**Evidencia** (`internal/httpapi/router.go:220`):

```go
mux.Handle("GET /api/v1/attachments/{filename}", http.HandlerFunc(app.handleDownloadAttachment))
```

Esta e a unica rota de `/api/v1/attachments` registrada **fora** do wrapper
`authed(...)` - compare com as linhas 191-192 do mesmo arquivo, que envolvem as
rotas irmao em `authed`.

O codigo se contradiz sobre isso. A linha 219 justifica a excecao:

```go
// Download de anexos (arquivo por nome - opaco, nao requer auth)
```

mas as linhas 415-418 afirmam o oposto - que as rotas `publico` sao **"as unicas
sem auth"** - e citam nominalmente `/api/v1/attachments/{arquivo}` como um
caminho que "serve tambem comprovante financeiro". Ou seja: a rota que o proprio
codigo identifica como sensivel e justamente a que ficou sem auth.

Agrava o caso que a rota tambem **nao passa pelo wrapper `publico`**, portanto
nao tem limite por IP nem `X-Robots-Tag: noindex` - ao contrario das rotas
publicas de carteirinha (`router.go:419-421`).

**Por que importa.** O diretorio de uploads e compartilhado entre fotos de membro
e anexos do financeiro. A premissa de "nome opaco" nao se sustenta: o nome do
arquivo e `hex(12 bytes)_<nome original saneado>` (`finance_handlers.go:940-943`),
ou seja, **incorpora o nome original enviado pelo usuario**. Sem autenticacao e
sem transacao com contexto de tenant, nao ha RLS na leitura: quem observar ou
adivinhar um nome le documento financeiro de outra igreja.

**Melhoria.** Autenticar a rota e resolver o anexo dentro de `WithTenant`,
validando `tenant_id`/`branch_id` antes de servir. Alternativa: separar fotos de
membro em outro prefixo/diretorio e manter apenas esse caminho publico e opaco.

---

### SEC-2 - Upload de anexo sem limite de corpo e sem validacao de tipo **[verificado parcialmente]**

`finance_handlers.go:922` usa `r.ParseMultipartForm(50 << 20)`. Esse argumento e
o limite *em memoria*; o excedente vai para arquivo temporario **sem teto
total**, permitindo streaming arbitrario para o disco. O tipo e lido do
`Content-Type` do cliente (`:959`), sem deteccao por magic bytes.

Contraste: o upload de foto de membro (`member_photo_handlers.go:45`) usa
`http.MaxBytesReader` e valida magic bytes (`:66-71`). O anexo financeiro deveria
seguir o mesmo padrao. Envolver com `http.MaxBytesReader(w, r.Body, max)` antes do
`ParseMultipartForm` e validar com `http.DetectContentType`.

---

### FIN-1 - Dinheiro trafega como `float64` de ponta a ponta **[verificado]**

O banco esta correto: todas as colunas monetarias sao `numeric(14,2)`
(`000006_finance.up.sql:23,44`, `000018_finance_restructure.up.sql:51`, entre
outras). O problema esta no Go, que le e escreve tudo como `float64`.

O caso mais claro e o rateio por evento, em `internal/finance/finance.go:380-384`:

```go
perMissing := 0.0
if missing > 0 {
	perMissing = (total - fixed) / float64(missing)
```

`(total - fixed) / float64(missing)` nao e representavel em binario; a soma dos
rateios pode nao fechar com o total. Somado aos casts `::float8` no SQL
(`finance.go:341,413,523,586`, `audit.go:103,104,154,195`, `org.go:433,437`),
isso produz centavos de diferenca, relatorios nao reprodutiveis e divergencia
entre recalculado e armazenado em um sistema contabil.

**Melhoria.** Representar dinheiro como `int64` (centavos) ou
`shopspring/decimal` no Go, ler `numeric` como `decimal.Decimal` e remover os
casts `::float8` do SQL. No minimo, fazer a aritmetica de rateio em SQL com
`numeric` e arredondar explicitamente.

---

### FE-1 - Todo o conjunto `/api/v1/people` do frontend aponta para rota inexistente **[verificado]**

**Evidencia**

```
apps/webadmin/lib/api.ts:509  /api/v1/people
apps/webadmin/lib/api.ts:513  /api/v1/people/${type}
apps/webadmin/lib/api.ts:516  /api/v1/people/${type}/${id}
```

`rg "api/v1/people" internal/httpapi` nao retorna nada: nao existe essa rota no
backend. `people_handlers.go` contem apenas handlers de familias, visitantes e
benfeitores.

**Por que importa.** `listPeople`, `createPerson`, `updatePerson`, os tipos
`Person`/`PeopleQuery`/`PeopleResponse`, os hooks SWR correspondentes e o cluster
`components/people/` (`entity-data-table.tsx`, `unified-drawer.tsx`,
`person-card.tsx`) sao codigo morto que retornaria 404 se ligado. E uma
armadilha ativa para quem for usar `usePeople`.

**Melhoria.** Remover a superficie morta ou, se o endpoint unificado e planejado,
cria-lo primeiro no `router.go`.

---

## 3. Alto

### AUTH-1 - Mutacoes financeiras sem verificacao de permissao **[verificado]**

Existe um resolvedor de permissao real, `hasFinancePerm`
(`reconciliation_handlers.go:18`), mas ele so e usado em **tres** arquivos:
`reconciliation_handlers.go`, `audit_handlers.go` e
`bank_reconciliation_handlers.go`. Em `finance_handlers.go` - o arquivo principal
de lancamentos, com 1064 linhas - **nao ha uma unica chamada a `hasFinancePerm`**
(confirmado: zero ocorrencias).

| Rota | Handler (linha) | Verificacao |
|---|---|---|
| `POST /finance/audits` | `handleCreateAudit` (48) | `finance.audit` (64) |
| `POST /finance/reconciliations` | `handleCreateReconciliation` (80) | `finance.reconcile` (96) |
| `POST /finance/bank-imports` | `handleImportBankStatement` (68) | `finance.reconcile` (93) |
| `POST /finance/bank-imports/{id}/entries/{entryId}/generate` | `handleGenerateBankEntry` (147) | `finance.write` (163) |
| `DELETE /finance/reconciliations/{id}` | `handleDeleteReconciliation` (173) | `isAdmin` (179) |
| `DELETE /finance/audits/{id}` | `handleDeleteAudit` (283) | `isAdmin` (289) |
| `POST /finance/transactions` | `handleCreateTxn` (241) | **nenhuma** |
| `POST /finance/transactions/batch` | `handleCreateTxnBatch` (286) | **nenhuma** |
| `POST /finance/transactions/import` | `handleImportTransactions` | **nenhuma** |
| `POST /finance/transactions/{id}/void` | `handleVoidTxn` (417) | **nenhuma** |
| `DELETE /finance/transactions/{id}` | `handleDeleteTxn` (452) | **nenhuma** |
| `POST /finance/transfers` | `handleCreateTransfer` (674) | **nenhuma** |

Confirmei lendo `handleCreateTxn` (`finance_handlers.go:241-281`): a unica guarda
e `claimsFrom(r.Context())`, que apenas checa autenticacao. **Qualquer usuario
autenticado** (inclusive um papel de leitura) pode criar, importar em lote,
estornar e **excluir definitivamente** lancamentos financeiros e repasses,
enquanto as operacoes de auditoria e conciliacao estao protegidas. Alem disso, a
exclusao de auditoria e de conciliacao usa `isAdmin(claims.Role)` em vez da
permissao `finance.audit` - dois criterios de autorizacao diferentes para a mesma
area.

**Melhoria.** Aplicar `hasFinancePerm(..., "finance.write")` em todas as
mutacoes de lancamento/repasses e `finance.audit` na exclusao de auditoria,
espelhando `bank_reconciliation_handlers.go:163`.

---

### AUTH-2 - O middleware autentica, mas nunca autoriza

`middleware.go:27-49` valida o Bearer token e injeta claims; nao ha checagem de
papel/permissao. Toda rota e envolvida apenas em `authed(...)`, e a autorizacao
fica espalhada e ad hoc nos handlers:

- `adminClaims` (`users_handlers.go:20`) - usuarios, papeis, filiais, tenant.
- `superAdminClaims` (`admin_handlers.go:16`) - `admin/reset-data`, `admin/tenants`.
- Checagem inline `isAdmin(claims.Role)` - `handlers.go:303`,
  `audit_handlers.go:289`, `bank_reconciliation_handlers.go:195`,
  `reconciliation_handlers.go:179`, `lgpd_handlers.go:40,165`.
- **Sem checagem alguma** - a maioria dos CRUD (members, families, visitors,
  events, groups, ministries, rosters, kids, governance, announcements...).

O comentario em `router.go:316` documenta isso como intencional ("autorizacao por
papel no handler"), mas o resultado ja e inconsistente (ver AUTH-1) e nao ha um
ponto unico onde auditar "quem pode o que".

**Melhoria.** Introduzir um middleware `requirePerm(perm string)` e declarar as
permisssoes no `router.go`, deixando a autorizacao visivel na tabela de rotas.

---

### DB-1 - `audit_log_hash` mantem o bug de ordenacao nao temporal e sem escopo de tenant **[verificado]**

**Evidencia** (`db/migrations/000003_users_rbac_audit.up.sql:84`):

```sql
NEW.prev_hash := (SELECT hash FROM audit_log WHERE id < NEW.id ORDER BY id DESC LIMIT 1);
```

`id` e `uuid` aleatorio (`:45`), entao `id < NEW.id ORDER BY id DESC` ordena por
**valor de UUID, nao por tempo de insercao**, e **nao filtra por tenant**. E
exatamente o defeito que `fin_tx_hash` corrigiu em `000018` e `000027` (o
comentario de `000027:8-9` diz textualmente que "UUID ordering produzia cadeia
nao-temporal e podia atravessar tenants"), mas a correcao **nunca foi aplicada ao
`audit_log`**.

**Por que importa.** A garantia de integridade da trilha de auditoria e mais fraca
que a do financeiro, a verificacao por tenant e impossivel de isolar e o
encadeamento entre tenants vaza ordenacao.

**Melhoria.** Nova migracao redefinindo `audit_log_hash()` no padrao de
`member_history_hash()` (`000022:97-100`):
`WHERE tenant_id = NEW.tenant_id AND (created_at, id) < (NEW.created_at, NEW.id) ORDER BY created_at DESC, id DESC LIMIT 1`,
com re-encadeamento das linhas existentes.

---

### DB-2 - Down de `000009` apaga o catalogo global de permissoes **[verificado]**

**Evidencia** (`db/migrations/000009_seed_demo.down.sql`):

```sql
DELETE FROM role_permissions WHERE role_id IN (
    SELECT id FROM roles WHERE tenant_id = '11111111-1111-1111-1111-111111111111'
);
...
DELETE FROM permissions;
```

As demais instrucoes sao escopadas ao tenant `demo`, mas `DELETE FROM permissions;`
remove o catalogo **global**. Como `role_permissions.permission_id` tem
`ON DELETE CASCADE` (`000003:23`), o rollback apaga os vinculos
papel->permissao de **todas as igrejas**, removendo o RBAC. `permissions` e um
catalogo populado por varias migracoes (000003, 000050, 000062...), nao um dado
do seed demo.

**Melhoria.** Remover a instrucao (o catalogo nao pertence ao seed) ou escopar aos
grants do tenant demo.

---

### DEL-1 - Workers de entrega sem lock, sem timeout por item e sem idempotencia

`internal/delivery/*` (`worker.go`, `announcement_worker.go`,
`notification_worker.go`, `schedule_worker.go`):

1. **Sem claim/lock.** `ListPending` (`documents/deliveries.go:76`) e
   `PendingDeliveries` (`announcements/deliveries.go:165`) fazem
   `SELECT ... WHERE status IN ('pending','failed') AND attempts < $1` sem
   `FOR UPDATE SKIP LOCKED` e sem transicao para "em envio". Com duas instancias
   da API (ou overlap de ticks), a **mesma mensagem e enviada duas vezes** -
   WhatsApp/e-mail duplicado para membros.
2. **Sem timeout por item.** `Dispatcher.Send` (`senders.go:160`) usa o contexto
   longo do processo (`main.go:40`). O `sendSMTP` (`senders.go:356-425`) nao tem
   deadline de dial/escrita - um SMTP pendurado travа o loop inteiro.
3. **Sem backoff.** `attempts < maxAttempts` re-tenta a cada tick (30s por
   padrao, `config.go:91`) sem `next_attempt_at`: um destinatario que sempre falha
   e atingido 5x em ~2,5 min.
4. **Vazamento de goroutine.** `senders.go:406-423` dispara goroutine para o corpo
   e, no `ctx.Done()`, retorna sem fechar a conexao nem drenar `done`.

**Melhoria.** Claim atomico (`UPDATE ... SET status='sending' ... RETURNING` ou
`FOR UPDATE SKIP LOCKED`), `context.WithTimeout` por item, backoff exponencial via
`next_attempt_at` e fechar o cliente SMTP no cancelamento.

---

### DEL-2 - Workers de notificacao/agenda rodam todos os tenants em uma unica transacao

`notification_worker.go:97` e `schedule_worker.go:107` abrem **uma** transacao
`WithSystem` e iteram todos os tenants dentro dela. Um erro em qualquer tenant
(por exemplo violacao de constraint) aborta a transacao, e o `continue`
(`notification_worker.go:83`) nao recupera porque o `tx` ja esta abortado. Efeito:
o problema de dados de uma igreja bloqueia silenciosamente a automacao de todas as
outras naquele tick.

**Melhoria.** Uma transacao por tenant (abrir `WithSystem`/`WithTenant` dentro do
loop) ou savepoints por tenant.

---

### CFG-1 - `EVOLUTION_API_KEY` tem tres valores default diferentes **[verificado]**

| Local | Valor |
|---|---|
| `internal/config/config.go:88` | `""` |
| `.env.example:85` | `chosen-evolution-local-key` |
| `infra/docker-compose.yml:36,132` | `chosen-evolution-key` |

O Compose usa o mesmo valor para `AUTHENTICATION_API_KEY` (Evolution) e
`EVOLUTION_API_KEY` (API), entao na doc ficam coerentes entre si - mas um dev que
copie `.env.example` para `.env` define `chosen-evolution-local-key`, que difere do
default do Compose. O resultado e falha silenciosa de autenticacao no WhatsApp.

**Melhoria.** Unificar o valor (e o texto do default) nos tres locais e, de
preferencia, falhar no boot quando a chave estiver vazia.

---

### CFG-2 - Segredos e metadados sensiveis fora do `.gitignore` **[verificado]**

```
?? Docs/05_Plano_Fluxo_de_Caixa_Contas_Pagar_Receber.md
?? "Docs/WhatsApp Unknown 2026-09-22 at 10.09.19.zip"
?? Docs/exemplos/
?? infra/cloudflare/config-erpchosen.yml.bak-20260925-144807
```

O `.gitignore` cobre `.env`, `.env.bak-*` (`:27`), `bin/` e
`infra/cloudflare/*.json` (`:31`) - tudo verificado e correto. Mas **nao** cobre:

- `infra/cloudflare/config-erpchosen.yml.bak-20260925-144807` (backup do config do
  tunnel, contem o UUID do tunnel) - o padrao `.env.bak-*` nao pega `*.yml.bak-*`.
- `Docs/*.zip` - `CLAUDE.md:165` alerta para nao commitar, mas nada impede.
- **`Docs/exemplos/`** - contem `ROL DE MEMBROS - POR FAMILIA.xlsx` e
  `rol_membros_agrupado_por_familia.xlsx`, com **dados pessoais reais de membros**
  (CPF/RG/endereco). Risco LGPD se alguem rodar `git add .`.

**Melhoria.** Adicionar `*.bak-*`, `Docs/*.zip` e `Docs/exemplos/` ao
`.gitignore` enquanto esses artefatos nao forem removidos do disco.

---

### DOC-1 - `GET /api/v1/auth/refresh` documentado, mas a rota e `POST` **[verificado]**

`AGENTS.md:273` documenta `GET /api/v1/auth/refresh`. O `router.go` registra o
metodo `POST`. Um cliente que siga a doc recebe 405.

**Melhoria.** Corrigir a doc para `POST`.

---

## 4. Medio

### Migracoes e banco

| ID | Achado | Evidencia | Melhoria |
|---|---|---|---|
| DB-3 | Dois prefixos duplicados: `000017` (`announcement_deliveries`, `member_enhancements`) e `000062` (`fin_account_groups`, `financial_reconciliation`) | `db/migrations/` | Renumerar para prefixos unicos e adicionar checagem de CI |
| DB-4 | `000051` inexistente (salta 000050 -> 000052) | `db/migrations/` | Documentar o buraco ou renumerar |
| DB-5 | Down de `000037` tem lista de policies hardcoded e incompleta vs. o up (que e dinamico) | `000037_sel_policies_select_only.down.sql:7-13` | Tornar o down dinamico, espelhando o up |
| DB-6 | `permissions` e `role_permissions` sem RLS (catalogos globais) | `000003_users_rbac_audit.up.sql:14,21` | Aceitavel por design, mas documentar; considerar grant restrito |
| DB-7 | Passwords demo (bcrypt) embutidos na cadeia de migracao | `000009:57`, `000021:39` | Mover seeds demo para um alvo/target separado, fora das migracoes |
| DB-8 | `memberships.role_id` sem `ON DELETE` explicito e sem indice | `000053_identity_memberships.up.sql:22` | Adicionar `ON DELETE` explicito + indice |
| DB-9 | `member_history.hash NOT NULL DEFAULT ''` enfraquece a garantia | `000022:62` | Remover o `DEFAULT ''` |
| DB-10 | Policy `guests_sel` numa tabela chamada `visitors` | `000007:79`, `000016:122` | Renomear para `visitors_sel` |
| DB-11 | `migrate.go` sem advisory lock: dois migradores concorrentes podem correr no mesmo version | `internal/store/migrate.go:59-83` | Envolver em `pg_advisory_lock` |

**Nota importante sobre DB-3 (nuance que corrige a leitura mais obvia).** O runner
calcula `version = nome[:len(nome)-len(".up.sql")]` (`migrate.go:60`), ou seja o
**stem completo do arquivo**, e ordena por `sort.Strings` dos nomes
(`migrate.go:57`). Logo os dois arquivos `000017_*` produzem versoes distintas
(`000017_announcement_deliveries` vs `000017_member_enhancements`) e **ambos sao
aplicados** - nao ha colisao de chave nem perda de migracao. O prefixo numerico e
apenas decorativo para a ordenacao: `000017_announcement_deliveries` roda antes de
`000017_member_enhancements` por ordem alfabetica, nao por intencao. E um risco
para humanos (um futuro `000017_zzz.up.sql` cairia depois dos dois), nao um bug em
tempo de execucao.

### Backend

| ID | Achado | Evidencia | Melhoria |
|---|---|---|---|
| BE-1 | `handleListAttendance` nunca e roteado; `GET /groups/{id}/attendance` aponta para o handler de **eventos** | `groups_handlers.go:363` vs `router.go:296` | Definir a fonte correta: rotear para o handler de grupos ou remover o codigo morto |
| BE-2 | `GET /documents/by-token/{token}` exige auth, mas o token e feito para QR publico | `router.go:223` vs `finance/receipt.go:28,76` | Mover para `publico(...)` como o fluxo de carteirinha, ou confirmar que e so para staff |
| BE-3 | Envelope de erro consistente, mas **`err.Error()` cru vaza em 5xx** e o mesmo tipo de erro vira 400/404/409/500 conforme o handler | `handlers.go:153,168,225,262,288,324`; `finance_handlers.go:796` vs `:875`; `governance_handlers.go:320,345` | Centralizar `mapErr(err)` e nunca devolver `err.Error()` em 5xx |
| BE-4 | Ordem invertida: `400` e retornado antes do `403` da checagem de permissao | `audit_handlers.go:75-86,140-155,182-197` | Checar `!allowed` imediatamente apos a transacao |
| BE-5 | Path params `{id}` nao validados como UUID -> erro `22P02` vira 500 em vez de 400 | `handlers.go:275`, `finance_handlers.go:86`, `kids_handlers.go:70` | Validar com `uuidRe` (`middleware.go:104`) no inicio do handler |
| BE-6 | Paginacao inconsistente: a maioria das listas nao tem paginacao; `limit` sem teto em deliveries | `announcement_handlers.go:183-187` | Teto maximo no handler (ex. 500) |
| BE-7 | `handleListBranches` faz N+1 chamadas HTTP externas em um GET | `finance_handlers.go:647-668` | Mover o enriquecimento para um worker |
| BE-8 | `handleSendTestMessage` sem permissao: qualquer autenticado envia WhatsApp para numero arbitrario | `announcement_handlers.go:221-257` | Exigir `adminClaims`/permissao + rate limit |
| BE-9 | CORS `Access-Control-Allow-Origin: *` global, inclusive em rotas autenticadas | `router.go:426-437` | Restringir a allowlist de origens + `Vary: Origin` |
| BE-10 | `clientIP` (`lgpd_handlers.go:185`) e `ipDoCliente` (`ratelimit.go:69`) discordam no tratamento de proxy; a trilha de consentimento grava um IP menos confiavel | citado | Reutilizar `ipDoCliente` em `clientIP` |
| BE-11 | Pool `pgxpool` sem `MaxConns`/`MinConns`/`MaxConnLifetime` | `store/db.go:31` | Configurar explicitamente, idealmente por env |
| BE-12 | `WithTenant` faz 6 round-trips por transacao (5 `set_config` + CTE) mesmo sem branch | `store/db.go:53-78` | Combinar em um unico `set_config(...)` e pular a CTE quando `BranchID == ""` |
| BE-13 | `Pool()` expoe a pool crua (bypass de RLS) e `reset_operational_data()` roda sem contexto de tenant | `store/db.go:41`, `admin_handlers.go:43` | Remover `Pool()` do API publica; expor metodos nomeados |
| BE-14 | Convencoes de erro divergentes entre pacotes (sentinela, tipo custom, `errors.New` inline) e helpers duplicados (`str`, `nullStr` com **duas assinaturas**, `deref`, `truncate`) | `events/events.go:481`, `org/org.go:400`, `ministries:192`, `groups:217`, ... | Padronizar em erros sentinela + pacote `internal/pgutil` |
| BE-15 | `audit_log.payload` e sempre `{}`: a trilha registra *que* aconteceu, nao *o que* mudou | `finance_handlers.go:271`, `handlers.go:256,314` | Serializar o diff/entrada relevante |

### Configuracao e infra

| ID | Achado | Evidencia | Melhoria |
|---|---|---|---|
| CFG-3 | `RECURRING_POLL_SECONDS` e lido pelo codigo e usado no Compose, mas **falta no `.env.example`** | `config.go:92`, `docker-compose.yml:136` | Adicionar ao `.env.example` |
| CFG-4 | `WHATSAPP_PROVIDER`: default do codigo e `meta`, `.env.example` diz `evolution` | `config.go:86` vs `.env.example:75` | Alinhar |
| CFG-5 | Servico `api` sem `healthcheck`; `webadmin`/`marketing` dependem de `service_started`, nao `service_healthy` | `docker-compose.yml:100-144,160-162,184-186` | Adicionar healthcheck e usar `service_healthy` |
| CFG-6 | Upstream `ipilp-moderno:3000` no config do tunnel nao existe no Compose | `infra/cloudflare/config-erpchosen.yml:15-16` | Remover a regra ou adicionar o servico |
| CFG-7 | nginx nao adiciona headers de seguranca (o marketing seta na app, o **webadmin nao**) | `nginx.conf`, `default.conf` | Adicionar `X-Frame-Options`, `nosniff`, `Referrer-Policy`, HSTS |
| CFG-8 | `/metrics` e publico e proxied, sem allowlist | `router.go:94`, `default.conf:38-42,103-107` | Restringir a rede interna ou exigir auth |
| CFG-9 | `/metrics` nao e formato Prometheus valido (sem `# HELP`/`# TYPE`, sem `promhttp`): so 3 gauges custom | `httpapi/health.go:18-28` | Usar `promhttp.Handler()` |
| CFG-10 | `infra/postgres/initdb/01-setup.sql` e `db/init/setup.sql` sao quase-duplicatas **divergentes** (so o primeiro cria o banco `evolution`) | ambos | Deduplicar ou documentar a divergencia |
| CFG-11 | Config morta: `POSTGRES_USER`/`POSTGRES_DB` do `.env.example` nao sao interpolados (o Compose hardcoda); Redis/RabbitMQ sao usados pelo Compose mas **nenhum codigo Go os le** | `.env.example:7,9,17,20-23` vs `docker-compose.yml:12,14` | Remover ou comentar o que e morto; confirmar se Redis/RabbitMQ tem uso previsto |
| CFG-12 | `JWT_SECRET` com default `dev-insecure-secret-change-me` e sem fail-fast; bcrypt no custo default; refresh token de 7 dias sem rotacao/revogacao (`jti`) | `config.go:70-72`, `auth/password.go:7`, `auth/jwt.go:19-34` | Falhar no boot com segredo default; adicionar `jti` + revogacao |
| CFG-13 | `next dev -p 33000` e igual em webadmin e marketing: **nao da para rodar os dois em dev ao mesmo tempo** | `apps/*/package.json` | Dar porta distinta ao marketing (ex. 33010), como ja acontece no Compose |
| CFG-14 | `README.md:25-30` omite as portas de marketing (33010) e Evolution (38081); `README.md:46` ainda cita o dominio antigo como principal | `README.md` | Atualizar |

### Site institucional

| ID | Achado | Evidencia | Melhoria |
|---|---|---|---|
| MKT-1 | Sem checkbox de consentimento LGPD e sem coluna `consent`/`consent_at`; o site **anuncia** "LGPD: consentimento..." | `lead-form.tsx:82-203`, `000060_marketing_leads.up.sql:8-18`, `lib/site.ts:215` | Adicionar consentimento obrigatorio + persistir |
| MKT-2 | `DisallowUnknownFields` no backend (`helpers.go:22`) cria acoplamento fragil: **qualquer campo novo no form vira 400** ate o struct Go ser atualizado | `leads_handlers.go:10-18` | Remover a restricao nesse endpoint publico ou versionar o contrato |
| MKT-3 | `whatsappLabel` e string hardcoded separada de `site.whatsapp`: mudar o numero por build arg atualiza o link e **deixa o numero exibido errado** | `lib/site.ts:15-16`, `site-footer.tsx:70`, `final-cta.tsx:33` | Derivar o label de `site.whatsapp` |
| MKT-4 | Dados de contato placeholder em producao (`+55 (48) 99999-0000`, `Florianopolis · SC`) | `lib/site.ts:16,18` | Substituir por valores reais ou por env |
| MKT-5 | Depoimentos simulados apresentados como pessoas reais, sem aviso | `lib/site.ts:236-259`, `sections/testimonials.tsx` | Substituir por reais autorizados ou remover |
| MKT-6 | Plano `Pro` no marketing **nao existe** no enum do webadmin (so `starter`/`enterprise`) | `lib/site.ts:265-301` vs `settings/page.tsx:360-362` | Alinhar os nomes de plano |
| MKT-7 | `ProductMockup` hardcoda `app.erpchosen.com.br` em vez de `site.appUrl` | `product-mockup.tsx:38` | Derivar do site config |
| MKT-8 | `sitemap.ts` usa `lastModified: new Date()` a cada request | `app/sitemap.ts:4-13` | Data fixa de build ou omitir |
| MKT-9 | Sem guarda in-flight no submit (so `disabled`); erro/sucesso sem `role="alert"`/`aria-live` | `lead-form.tsx:23-64,177-181` | Guarda de reenvio + ARIA |
| MKT-10 | JSON-LD `Offer` com `price: "0"` | `app/layout.tsx:69-74` | Usar `priceSpecification` ou omitir |

### Frontend webadmin

| ID | Achado | Evidencia | Melhoria |
|---|---|---|---|
| FE-2 | Menu renderiza itens que o usuario nao pode usar (perms divergentes do gate real da pagina); `announcements` usa `perms: []` (sempre visivel); faltam chaves `users.read`/`settings.read`/`governance.read` em `PERMISSION_LABELS` | `dashboard/layout.tsx:28-79`, `lib/constants.ts:157-170` | Alinhar `NavItem.perms` ao gate real e unificar o vocabulario de permissao |
| FE-3 | 12+ paginas **sem nenhuma** checagem de permissao (visitors, benefactors, suppliers, transfers, kids, e todas de reports) | citado | Adicionar `hasPerm` nos botoes de acao |
| FE-4 | Fetcher do SWR le o token direto do `localStorage`, **nao envia `X-Branch-Id`** e **nao dispara refresh** no 401 | `lib/swr-hooks.ts:44-49` | Reutilizar o `api()`/`doFetch` com refresh |
| FE-5 | Refresh concorrente sem dedup: N chamadas 401 disparam N `POST /auth/refresh`; com rotacao, podem se invalidar e forcar logout | `lib/api.ts:568-608` | Guardar um `refreshPromise` module-level |
| FE-6 | Estados de loading/erro/vazio inconsistentes entre paginas (skeleton vs `"..."`; erro engolido com `.catch(() => null)`/`.catch(() => {})`) | `dashboard/page.tsx:48-49`, `ministries/page.tsx:64` | Padronizar skeleton + `EmptyState` + toast de erro |
| FE-7 | Formatacao ad hoc de data fora do `lib/format.ts` (`toLocaleString` direto) e sem formatador de bytes | `announcements/page.tsx:109,637,880`, `kids/page.tsx:619,717` | Usar `dateTimePt`/`datePt`; criar `fileSize()` |
| FE-8 | Listas de dominio duplicadas nas paginas em vez de importar `lib/constants.ts` | `visitors/page.tsx:23-24` vs `constants.ts:115-123` | Importar `VISITOR_SOURCES`/`JOURNEY_ORDER` |
| FE-9 | `X-Tenant-Id` nunca e enviado (backend aceita); `getTenantContext` e codigo morto apesar de gravar `chosen_tenant` | `lib/api.ts:80-88,101-104` | Usar de verdade ou remover |
| FE-10 | `assetURL` e no-op (os dois ramos do ternario retornam `path`) e e usado em 6 lugares | `lib/api.ts:1899-1900` | Implementar a traducao pretendida ou simplificar |
| FE-11 | `confirm()` nativo em 20+ acoes destrutivas, sem dialog acessivel (ja existe `Modal`) | `members/page.tsx:105`, `finance/page.tsx:319`, ... | Criar `ConfirmDialog` no design system |
| FE-12 | `Modal`/`Drawer`/`Pagination`/`Tabs` sem `type="button"` (risco de submit acidental dentro de form) e sem ARIA de tab | `components/ui/modal.tsx:41,79`, `pagination.tsx:52,72,80`, `tabs.tsx:19` | Adicionar `type="button"` e roles |
| FE-13 | `any` no kids: `(form as any).id` | `kids/page.tsx:244` | Tipar o form |

### Documentacao

| ID | Achado | Evidencia |
|---|---|---|
| DOC-2 | Tres fontes de verdade para status de fase que **discordam entre si** (Fase 0 "80%" vs "90%"; Fase 1 "55%" vs "60%") | `Docs/00_Checkpoint.md`, `02_Backlog.md`, `03_Plano_de_Execucao_Pendencias.md` |
| DOC-3 | PRD descreve arquitetura **microservices + Python/FastAPI**, mas o codigo e monolito Go | `Chosen_ERP_Documentacao_Completa.md:101,107-114` vs `cmd/api/main.go` |
| DOC-4 | `CLAUDE.md` afirma "3 workers em goroutines" (sao 5), "unica suite e a de RLS" (ha `middleware_test.go`, `store/*_test.go`, `delivery/*_test.go`, `kids_test.go`), e "proxima migracao deve ser 000021" (ja estamos em 000066) | `CLAUDE.md:39,101,140` |
| DOC-5 | `00_Checkpoint.md` afirma "sem rota DELETE alguma na API" (o router tem dezenas) e "10 migracoes" (sao 66) | `Docs/00_Checkpoint.md:634,55` |
| DOC-6 | Contradicao interna sobre `pastor.norte@demo.local` e sobre a tabela `financial_classification_types` (removida pela 000026) | `00_Checkpoint.md:246,578`; `AGENTS.md:101` |
| DOC-7 | Plano 04 ainda cita `000051_identity_memberships`; a real e `000053` | `Docs/04_Plano_Multi_Tenant_Subdominio.md:62,348` |

---

## 5. Baixo

| ID | Achado | Evidencia |
|---|---|---|
| LOW-1 | `CardBody` e wrapper inutil (`<>{children}</>`) | `components/ui/card.tsx:20-22` |
| LOW-2 | Codigo morto extenso: hooks SWR nao usados, `components/people/{entity-data-table,unified-drawer,person-card}.tsx`, `createPersonTypeColumn` | `lib/swr-hooks.ts`, `components/people/`, `components/ui/data-table.tsx:343` |
| LOW-3 | `mark(...providerMsgID)` sempre recebe `""`; `simulatedSender.provider` e `""` para e-mail enquanto `ProviderFor` responde `"smtp"` | `delivery/announcement_worker.go:86-91`, `senders.go:108,230` |
| LOW-4 | `delay: 1234` hardcoded no payload da Evolution | `delivery/senders.go:509` |
| LOW-5 | `Content-Disposition: inline` para upload arbitrario | `finance_handlers.go:1053` |
| LOW-6 | `handleConvertVisitor` faz duas transacoes separadas (falha parcial possivel) e engole not-found como 500 | `people_handlers.go:406-460` |
| LOW-7 | `handleUpdateProfile` aceita nome/e-mail vazios, enquanto `handleCreateUser` valida | `settings_handlers.go:55-91` vs `users_handlers.go:64-67` |
| LOW-8 | MFA: TOTP sem protecao a replay e segredo em plaintext; sem politica de forca de senha | `auth/totp.go:48-61`, `auth/service.go:420-435,503-518` |
| LOW-9 | `api.ts` com `eslint-disable` em 9 pontos (majoritariamente `exhaustive-deps`) | varios |
| LOW-10 | `Button` do marketing sem `type="button"` default (latente dentro de form) | `components/ui/button.tsx:60-64` |
| LOW-11 | `SectionHeading` com override `max-w-none` duplicado em 2 secoes; 3 "pills" quase identicas repetidas | `sections/governance.tsx:20-27`, `multi-branch.tsx:44-50`, `hero.tsx:19-22` |
| LOW-12 | `opengraph-image.tsx` usa o glifo `✝` em vez do logo SVG | `app/opengraph-image.tsx:37` |
| LOW-13 | `robots.ts` do marketing nao bloqueia `/api/` | `app/robots.ts` |
| LOW-14 | `nginx`: `proxy_set_header Connection "upgrade"` incondicional | `nginx.conf:40`, `default.conf:28,93` |
| LOW-15 | `000062` redefine `create_tenant`; `create_tenant` foi redefinido 3x (000059/000062/000065) - ok por `CREATE OR REPLACE`, mas aumenta a superficie de atencao | `db/migrations/` |

---

## 6. O que esta correto e deve ser preservado

Vale registrar os pontos fortes, para nao serem perdidos em uma refatoracao:

- **Modelo RLS coerente.** Toda tabela com `tenant_id` tem RLS habilitada e ao menos
  uma policy; tabelas filhas usam `EXISTS` sobre o pai. `marketing_leads` e a
  excecao deliberada e documentada (`REVOKE ALL` + `GRANT INSERT`).
- **`store.WithTenant`/`WithSystem` como ponto unico de RLS** (`store/db.go:46,108`)
  - praticamente todos os handlers passam por ele; nao encontrei handler lendo
    dado de tenant direto da pool (as excecoes sao intencionais).
- **Migracoes transacionais, versionadas e idempotentes**, com `up`/`down`
  **100% pareados** (67/67, sem orfaos).
- **Todas as colunas de dinheiro no banco sao `numeric(14,2)`** - nenhum
  `double precision`/`real`. O defeito esta so na camada Go (FIN-1).
- **`000037` corrigiu corretamente o defeito de policies `_sel` `FOR ALL`**, e todas
  as `_sel` criadas depois ja nasceram `FOR SELECT`.
- **Cadeia de hash do financeiro bem desenhada** (escopo de tenant, ordenacao
  temporal, `SECURITY DEFINER` com validacao de tenant no rechain).
- **Tratamento de senha/MFA/login cuidadoso:** MFA so apos senha correta, token de
  selecao de igreja com 5 min, membership revalidada em `Me`/`SwitchTenant`.
- **Endpoints publicos deliberadamente minimos** (carteirinha devolve projecao
  reduzida; ESCAPE de HTML em recibo/carteirinha/relatorios).
- **Rate limit + `noindex` nas rotas publicas** (`ratelimit.go:90-98`).
- **Design system real e majoritariamente reutilizado** no webadmin; formatacao e
  constantes centralizadas em `lib/format.ts` e `lib/constants.ts`.
- **`--chown=65532:65532` + volume nomeado de uploads** implementado exatamente como
  documentado (`infra/api/Dockerfile:16,18`).
- **Split marketing/webadmin via `server_name` exato vs. regex** correto no nginx,
  com `X-Tenant-Slug` repassado em `/` e `/api/`.
- **Contrato do lead form alinhado** ao struct Go nos campos, honeypot corretamente
  fora do JSON, URL relativa que funciona em dev e prod.
- **`.gitignore` cobre os tres segredos de maior risco** (`.env`, `.env.bak-*`,
  `infra/cloudflare/*.json`) - verificado.
- **Existe CI** com `CHOSEN_TESTS_REQUIRED=1` para build/vet/test do Go e
  typecheck/build dos dois apps.

---

## 7. Ordem de correcao sugerida

Agrupada por relacao risco x custo. Os cinco primeiros sao os de maior retorno.

1. **SEC-1** - autenticar e escopar `GET /api/v1/attachments/{filename}`. Uma
   mudanca pequena, exposicao de dado financeiro de outro tenant.
2. **AUTH-1 + AUTH-2** - aplicar `hasFinancePerm` nas mutacoes financeiras e
   introduzir `requirePerm` declarativo no `router.go`. Elimina a classe de falha.
3. **CFG-2** - fechar o `.gitignore` (`*.bak-*`, `Docs/*.zip`, `Docs/exemplos/`) e
   remover do disco os artefatos com dado pessoal. Custo quase zero, risco LGPD.
4. **DB-1 + DB-2** - corrigir `audit_log_hash` (temporal + tenant) e remover o
   `DELETE FROM permissions` do down de `000009`.
5. **CFG-1 + CFG-3** - unificar `EVOLUTION_API_KEY` e adicionar
   `RECURRING_POLL_SECONDS` ao `.env.example`.
6. **FIN-1** - migrar dinheiro para `int64` (centavos) ou `decimal`. E o maior
   refactor da lista; vale ser uma tarefa propria.
7. **DEL-1 + DEL-2** - claim/lock nos workers, timeout por item, transacao por
   tenant nos workers de notificacao/agenda.
8. **FE-1 + FE-4 + FE-5** - remover a superficie `/people`, unificar o fetcher SWR
   com o `api()`, deduplicar o refresh.
9. **FE-2 + FE-3** - alinhar permisssoes do menu e adicionar gates nas paginas.
10. **DOC-1..DOC-7** - consolidar a documentacao em uma fonte de verdade.
11. **DB-3..DB-11, BE-*, CFG-4..CFG-14, MKT-*, FE-6..FE-13, LOW-*** - o restante,
    por area, quando houver janela.

---

## 8. Itens para validacao manual (nao concluidos nesta analise)

Estes achados vem de varredura e merecem confirmacao antes da correcao:

- **DEL-1/DEL-2**: reproduzir envio duplicado com dois workers simultaneos.
- **BE-1**: descobrir se `groups` e `events` compartilham a mesma tabela de
  presenca - define se e bug ou so codigo morto.
- **BE-2**: confirmar a intencao do QR de documento (publico vs. staff).
- **BE-3**: mapear caso a caso os status retornados por erro equivalente.
- **CFG-11**: confirmar se Redis/RabbitMQ tem uso planejado (hoje os workers sao
  goroutines in-process, nao broker).
- **MKT-6**: confirmar o conjunto real de planos suportados pelo produto.
- **FIN-1**: medir o impacto pratico nos relatorios existentes antes de refatorar.
