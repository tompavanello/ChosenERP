# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> Convencao do repositorio: **comentarios, docs, mensagens de commit e UI em portugues (pt-BR)**.

## Visao geral

Chosen ERP - SaaS eclesiastico **multi-tenant**: monolito modular em **Go**, **PostgreSQL 17** com Row-Level Security, frontend **Next.js 15** (App Router + Tailwind v4) e infra em **Docker Compose**.

| Documento | Conteudo |
|---|---|
| `AGENTS.md` | Guia de desenvolvimento completo - inclui a **tabela de todos os endpoints** da API |
| `Docs/00_Checkpoint.md` | Estado do projeto, decisoes de arquitetura, incidentes (5.1) |
| `Docs/02_Backlog.md` | Backlog priorizado (P0-P4) confrontado com o PRD |
| `Docs/Chosen_ERP_Documentacao_Completa.md` | PRD / especificacao do produto |

## Comandos

### Subir o ambiente

```powershell
.\start.ps1            # .env + build do Go + Docker (postgres+api+webadmin+nginx)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus, grafana
```

Portas: API **38080** - PostgreSQL **35432** - Webadmin **33000**. O webadmin roda em Docker (`chosen-webadmin`, Next.js standalone, `restart: unless-stopped`) - nao e iniciado no host. Login dev: `admin@demo.local` e `pastor.norte@demo.local` - as senhas ficam no `.env` (`DEMO_ADMIN_PASSWORD` / `DEMO_NORTE_PASSWORD`), nao versionadas. Health: `curl http://localhost:38080/healthz`.

### Backend (Go)

```bash
go build ./...
go vet ./...
go run ./cmd/api
```

### Testes

A unica suite e a de RLS (`internal/store/rls_test.go`), que exige um **PostgreSQL real de pe**. Ela cria e recria um banco descartavel `chosenerp_test` - **nao toca o banco de desenvolvimento**.

```powershell
$env:CHOSEN_TEST_MIGRATE_URL="postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"
$env:CHOSEN_TEST_APP_URL="postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"

go test ./... -count=1

# um unico teste
go test ./internal/store -run TestRLS_NoCrossTenantReadForAnyTable -count=1
```

As variaveis caem para `MIGRATE_DATABASE_URL` / `DATABASE_URL`; sem nenhuma delas a suite e **pulada com aviso** (`CHOSEN_TESTS_REQUIRED=1` faz **falhar** em vez de pular - use no CI). O harness reaplica `db/init/setup.sql` no banco de teste porque `GRANT` tem escopo de database.

Antes de commitar: `go build ./... && go vet ./... && go test ./... -count=1`.

### Webadmin

```bash
cd apps/webadmin
npm run dev                                # next dev -p 33000 (porta fixa no package.json)
npx tsc --noEmit                           # unica verificacao automatica disponivel
NEXT_DIST_DIR=.next-verify npm run build   # evita corromper o .next do dev em execucao
```

 Qualquer `build` com `NEXT_DIST_DIR` **reescreve dois arquivos versionados** e eles precisam
voltar depois - o Next grava o distDir alternativo dentro deles:

```bash
git checkout -- apps/webadmin/tsconfig.json apps/webadmin/next-env.d.ts
```

Sem isso o `next-env.d.ts` fica com um `/// <reference path="./.next-verify/types/routes.d.ts" />`
pendurado (aponta para pasta que pode nao existir) e o `include` do `tsconfig.json` vai acumulando
`.next-*/types/**` a cada build.

Nao ha test runner nem ESLint instalado: `npm run lint` cai no prompt interativo de setup do Next e **falha em modo nao-interativo**. Use `tsc --noEmit` (tsconfig `strict`). Alias de import: `@/*` -> raiz de `apps/webadmin`.

### Imagem da API (`FROM scratch`)

A imagem nao faz pull de nada - o binario Linux e compilado **no host** antes:

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o bin/api-linux-amd64 ./cmd/api
docker build -f infra/api/Dockerfile -t chosenerp/api:local .
```

(`start.ps1` ja executa exatamente isso.)

O container roda como **UID `65532`** e o unico diretorio gravavel e `/uploads`
(volume nomeado `uploads` do Compose, montado sobre `UPLOAD_DIR`). Como a imagem e
`FROM scratch`, `/uploads` **nao existe** nela por padrao - por isso o Dockerfile tem
`COPY --chown=65532:65532 infra/api/uploads/ /uploads/`: ao montar um volume nomeado
vazio, o Docker inicializa o volume com o dono do diretorio correspondente na imagem.
Sem esse `COPY` o volume nasce `root:root`, e upload de foto/anexo falha com
`permission denied` **sem erro no log** do container.

## Arquitetura

### Backend: monolito modular com `tx` injetado

`cmd/api/main.go` e o composition root: `store.New` -> `st.Migrate` -> `auth.Service` -> `delivery.Dispatcher` -> `httpapi.NewRouter`, e entao **tres workers em goroutines** (todos ticker-based, controlados por env):

- `delivery.Worker` - outbox de recibos/carteirinhas (`document_deliveries`), com retry;
- `delivery.AnnouncementWorker` - disparo de comunicados;
- `finance.RecurringWorker` - doacoes recorrentes (gera lancamento + recibo).

Os pacotes de dominio (`internal/members`, `internal/finance`, `internal/visitors`, ...) **nao tem o pool de conexoes**: cada metodo de repositorio recebe `(ctx, tx pgx.Tx, ...)` e o handler e quem abre a transacao. O padrao de todo handler autenticado e:

```go
claims, ok := claimsFrom(r.Context())   // internal/httpapi/middleware.go
b := boundsFromClaims(claims)           // internal/httpapi/handlers.go
err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
    out, err = a.Members.List(r.Context(), tx, q)
    return err
})
```

Convencoes de HTTP:

- `writeJSON(w, status, v)` e `writeErr(w, status, msg)` -> corpo de erro `{"error":"..."}` (`internal/httpapi/response.go`).
- `readJSON(r, v)` usa `DisallowUnknownFields` e limite de 1 MB - **campo extra no JSON vira 400**.
- Envelopes de sucesso sao `map[string]any` ad-hoc (`{"members": [...]}`, `{"transactions": [...]}`), sem tipo compartilhado. Renomear uma chave quebra o webadmin **em runtime, sem erro de compilacao**.

### RLS e a fronteira de seguranca real

- Duas roles de banco: **migrador** (`postgres`, dono do schema, roda as migracoes) e **app** (`chosenerp_app`, nao-superuser  o PostgreSQL aplica RLS em toda query).
- `Store.WithTenant` seta `app.tenant_id` / `app.branch_id` / `app.role` como GUCs **locais da transacao**; `Store.WithSystem` seta `role='system'` e escopo vazio (**acesso total** - usado pelos workers e pelo cartao publico em `memberapp_handlers.go`).
- `BranchID` vazio = escopo **Sede** (enxerga todo o seu tenant); `role='system'` enxerga tudo.
- Helpers de politica (`000016_tenant_scope_rls.up.sql`): `is_system()`, `is_headquarters()`, `rls_read()`, `rls_write()`, `rls_hq()`. `rls_write` e **branch exato** - Sede **nao** escreve em outra filial.
- `financial_transactions` e `audit_log` sao **append-only com hash-chain** (triggers bloqueiam UPDATE/DELETE).
- Login usa `auth_lookup_user` (`SECURITY DEFINER`) porque o tenant ainda nao e conhecido nesse ponto.
-  **Nao existe verificacao de permissao no servidor.** Nao ha `requirePerm` nem 403: o RBAC por `permissions` e aplicado **apenas no frontend** (`hasPerm`), e no backend a autorizacao e RLS + `app.role` dentro do SQL. Um endpoint de escrita novo fica acessivel a **qualquer usuario autenticado** do tenant, salvo se o RLS bloquear.

### Migracoes

`db/migrations/NNNNNN_nome_curto.{up,down}.sql`, embutidas no binario via `//go:embed` (`db/embed.go`) e aplicadas **no boot da API** (`internal/store/migrate.go`), numa conexao separada com o DSN do migrador em *simple protocol* (permite multiplos comandos por arquivo).

Detalhes que importam:

- A chave de versao e o **nome completo do arquivo sem `.up.sql`**, e a ordem e `sort.Strings` (lexicografica). O prefixo `000017` esta **duplicado** (`announcement_deliveries` e `member_enhancements`): ambos aplicam. A ultima e a `000020_cargos` - **a proxima migracao deve ser `000021_`.**
- Arquivos `.down.sql` sao embutidos mas **nunca executados** - nenhum codigo os le (rollback e manual).
- Nao ha advisory lock: nao suba duas instancias da API aplicando migracoes ao mesmo tempo.
- `db/init/setup.sql` cria a role da aplicacao e e exposto como `db.Setup`; precisa rodar **em cada database** (GRANT e database-scoped).

### Webadmin (Next.js)

- **A API e same-origin por caminho relativo.** O browser chama `/api/v1/...` sem host nenhum e resolve contra a origem que ele abriu; quem traduz isso para a API e o `rewrites()` do `next.config.ts` (`/api/:path*` -> `API_ORIGIN`, default `http://localhost:38080`). O rewrite e configuracao de **servidor**: o destino fica no `routes-manifest.json` do build, mas **nenhum endereco e gravado no bundle do cliente** - o mesmo build serve `localhost:33000` e o dominio publico. Para apontar para outra API, mude `API_ORIGIN` e reinicie o `next start` (nao precisa rebuildar). Antes havia um `env: { API_URL }` gravado em build time, o que amarrava o bundle a um endereco - foi o que fez o `localhost:33000` mandar o browser para o dominio publico (ver incidente em `Docs/00_Checkpoint.md`). Hoje nao ha CORS em jogo: as chamadas sao same-origin.
- `lib/api.ts`: tokens vivem em variaveis de modulo + `localStorage` (`chosen_token`, `chosen_refresh`, `chosen_user`). **As funcoes nao recebem token** - o helper `api()` injeta o Bearer e, em **401, faz refresh e repete a requisicao uma vez**; se o refresh falha, `clearTokens()` + `onSessionLost` (o AuthProvider registra `logout`). Use `apiRaw()` para FormData/HTML/blob (nao forca `Content-Type: application/json`). O arquivo tambem exporta os helpers de URL que os componentes devem usar em vez de montar caminho a mao: `cardPrintURL`, `cardPhotoURL` e `assetURL` (traduz um caminho devolvido pelo backend - ex.: `photo_url` - em URL para o browser; hoje e quase passagem, ja que as origens coincidem, mas e o unico ponto que muda se elas voltarem a divergir).
- `useAuth()` -> `{ user, ready, login, logout, hasPerm }`. `hasPerm` e **match exato de string** num array plano (sem hierarquia de papeis, sem wildcard). O catalogo de permissoes vem do banco (seed em `000009_seed_demo.up.sql`), nao de um enum no TypeScript.
- Menu lateral: array `NAV_SECTIONS` **inline em `app/dashboard/layout.tsx`** (agrupado por Pessoas/Financeiro/Organizacao), filtrado por `perms.some(hasPerm)` - semantica **OR**, array vazio = sempre visivel. `NAV_BY_PERMISSION` e `OFFICES` foram **removidos** de `lib/constants.ts`: a unica fonte de verdade do menu e o `layout.tsx`, e a de cargos e a API (`/api/v1/cargos`).
- Protecao de rota e **so client-side** (em `dashboard/layout.tsx`, via `router.replace("/")`). **Nao existe `middleware.ts`.**
- Tailwind v4 **CSS-first**: nao ha `tailwind.config.*`. Os tokens sao CSS custom properties em `app/globals.css` (`--ink`, `--paper`, `--card`, `--brand`, ...) e existem classes utilitarias proprias (`.btn-primary`, `.input`, `.card`) - `bg-brand` **nao existe**. Os componentes de `components/ui/*` sao artesanais e usam `cn()` (`lib/utils.ts`, clsx + tailwind-merge). Reutilize-os em vez de escrever classes avulsas.
- `lib/swr-hooks.ts` tem um `fetcher` proprio que le o token do `localStorage` **direto, sem passar pelo `api()`** - logo, **nao** faz refresh em 401. Chamadas via SWR e chamadas via `api()` se comportam de forma diferente em caso de token expirado.

### Autenticacao

JWT access (15 min) + refresh (168 h), claims `{user_id, tenant_id, branch_id, role, type}` - **tenant, filial e papel viajam dentro do token** e nao sao reconsultados a cada request. O middleware so valida o Bearer e injeta as claims no contexto (chave `claims`).

`GET /api/v1/me` devolve **a mesma forma** do payload do login (inclui `permissions`) porque o `AuthProvider` reidrata a sessao por ele apos um reload - mantenha os dois em sincronia.

## Armadilhas conhecidas

- **`pastor.norte@demo.local` / filial Norte nao sao criados por migracao.** O seed (`000009`) cria so o tenant `demo`, a filial "Sede Matriz" e `admin@demo.local`. A conta **existe no banco de desenvolvimento atual** porque foi criada a mao - ou seja, um banco recriado do zero nao a tera. Ver backlog #40.
- Testes de RLS so rodam com o Docker de pe e os DSNs exportados; caso contrario a suite e pulada **em silencio** (parece verde).
- `Docs/` contem artefatos nao versionados do cliente (`exemplos/`, `requisitos_basicos.txt`) e um `.zip` de ~1,1 MB - evite adiciona-los ao indice.
