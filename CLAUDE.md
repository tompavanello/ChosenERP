# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> Convenção do repositório: **comentários, docs, mensagens de commit e UI em português (pt-BR)**.

## Visão geral

Chosen ERP — SaaS eclesiástico **multi-tenant**: monolito modular em **Go**, **PostgreSQL 17** com Row-Level Security, frontend **Next.js 15** (App Router + Tailwind v4) e infra em **Docker Compose**.

| Documento | Conteúdo |
|---|---|
| `AGENTS.md` | Guia de desenvolvimento completo — inclui a **tabela de todos os endpoints** da API |
| `Docs/00_Checkpoint.md` | Estado do projeto, decisões de arquitetura, incidentes (§5.1) |
| `Docs/02_Backlog.md` | Backlog priorizado (P0–P4) confrontado com o PRD |
| `Docs/Chosen_ERP_Documentacao_Completa.md` | PRD / especificação do produto |

## Comandos

### Subir o ambiente

```powershell
.\start.ps1            # .env + build do Go + Docker (postgres+api+webadmin+nginx)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus, grafana
```

Portas: API **38080** · PostgreSQL **35432** · Webadmin **33000**. O webadmin roda em Docker (`chosen-webadmin`, Next.js standalone, `restart: unless-stopped`) — não é iniciado no host. Login dev: `admin@demo.local` e `pastor.norte@demo.local` — as senhas ficam no `.env` (`DEMO_ADMIN_PASSWORD` / `DEMO_NORTE_PASSWORD`), não versionadas. Health: `curl http://localhost:38080/healthz`.

### Backend (Go)

```bash
go build ./...
go vet ./...
go run ./cmd/api
```

### Testes

A única suíte é a de RLS (`internal/store/rls_test.go`), que exige um **PostgreSQL real de pé**. Ela cria e recria um banco descartável `chosenerp_test` — **não toca o banco de desenvolvimento**.

```powershell
$env:CHOSEN_TEST_MIGRATE_URL="postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"
$env:CHOSEN_TEST_APP_URL="postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"

go test ./... -count=1

# um único teste
go test ./internal/store -run TestRLS_NoCrossTenantReadForAnyTable -count=1
```

As variáveis caem para `MIGRATE_DATABASE_URL` / `DATABASE_URL`; sem nenhuma delas a suíte é **pulada com aviso** (`CHOSEN_TESTS_REQUIRED=1` faz **falhar** em vez de pular — use no CI). O harness reaplica `db/init/setup.sql` no banco de teste porque `GRANT` tem escopo de database.

Antes de commitar: `go build ./... && go vet ./... && go test ./... -count=1`.

### Webadmin

```bash
cd apps/webadmin
npm run dev                                # next dev -p 33000 (porta fixa no package.json)
npx tsc --noEmit                           # única verificação automática disponível
NEXT_DIST_DIR=.next-verify npm run build   # evita corromper o .next do dev em execução
```

⚠️ Qualquer `build` com `NEXT_DIST_DIR` **reescreve dois arquivos versionados** e eles precisam
voltar depois — o Next grava o distDir alternativo dentro deles:

```bash
git checkout -- apps/webadmin/tsconfig.json apps/webadmin/next-env.d.ts
```

Sem isso o `next-env.d.ts` fica com um `/// <reference path="./.next-verify/types/routes.d.ts" />`
pendurado (aponta para pasta que pode não existir) e o `include` do `tsconfig.json` vai acumulando
`.next-*/types/**` a cada build.

Não há test runner nem ESLint instalado: `npm run lint` cai no prompt interativo de setup do Next e **falha em modo não-interativo**. Use `tsc --noEmit` (tsconfig `strict`). Alias de import: `@/*` → raiz de `apps/webadmin`.

### Imagem da API (`FROM scratch`)

A imagem não faz pull de nada — o binário Linux é compilado **no host** antes:

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o bin/api-linux-amd64 ./cmd/api
docker build -f infra/api/Dockerfile -t chosenerp/api:local .
```

(`start.ps1` já executa exatamente isso.)

O container roda como **UID `65532`** e o único diretório gravável é `/uploads`
(volume nomeado `uploads` do Compose, montado sobre `UPLOAD_DIR`). Como a imagem é
`FROM scratch`, `/uploads` **não existe** nela por padrão — por isso o Dockerfile tem
`COPY --chown=65532:65532 infra/api/uploads/ /uploads/`: ao montar um volume nomeado
vazio, o Docker inicializa o volume com o dono do diretório correspondente na imagem.
Sem esse `COPY` o volume nasce `root:root`, e upload de foto/anexo falha com
`permission denied` **sem erro no log** do container.

## Arquitetura

### Backend: monolito modular com `tx` injetado

`cmd/api/main.go` é o composition root: `store.New` → `st.Migrate` → `auth.Service` → `delivery.Dispatcher` → `httpapi.NewRouter`, e então **três workers em goroutines** (todos ticker-based, controlados por env):

- `delivery.Worker` — outbox de recibos/carteirinhas (`document_deliveries`), com retry;
- `delivery.AnnouncementWorker` — disparo de comunicados;
- `finance.RecurringWorker` — doações recorrentes (gera lançamento + recibo).

Os pacotes de domínio (`internal/members`, `internal/finance`, `internal/visitors`, ...) **não têm o pool de conexões**: cada método de repositório recebe `(ctx, tx pgx.Tx, ...)` e o handler é quem abre a transação. O padrão de todo handler autenticado é:

```go
claims, ok := claimsFrom(r.Context())   // internal/httpapi/middleware.go
b := boundsFromClaims(claims)           // internal/httpapi/handlers.go
err := a.Store.WithTenant(r.Context(), b, func(tx pgx.Tx) error {
    out, err = a.Members.List(r.Context(), tx, q)
    return err
})
```

Convenções de HTTP:

- `writeJSON(w, status, v)` e `writeErr(w, status, msg)` → corpo de erro `{"error":"..."}` (`internal/httpapi/response.go`).
- `readJSON(r, v)` usa `DisallowUnknownFields` e limite de 1 MB — **campo extra no JSON vira 400**.
- Envelopes de sucesso são `map[string]any` ad-hoc (`{"members": [...]}`, `{"transactions": [...]}`), sem tipo compartilhado. Renomear uma chave quebra o webadmin **em runtime, sem erro de compilação**.

### RLS é a fronteira de segurança real

- Duas roles de banco: **migrador** (`postgres`, dono do schema, roda as migrações) e **app** (`chosenerp_app`, não-superuser ⇒ o PostgreSQL aplica RLS em toda query).
- `Store.WithTenant` seta `app.tenant_id` / `app.branch_id` / `app.role` como GUCs **locais da transação**; `Store.WithSystem` seta `role='system'` e escopo vazio (**acesso total** — usado pelos workers e pelo cartão público em `memberapp_handlers.go`).
- `BranchID` vazio = escopo **Sede** (enxerga todo o seu tenant); `role='system'` enxerga tudo.
- Helpers de política (`000016_tenant_scope_rls.up.sql`): `is_system()`, `is_headquarters()`, `rls_read()`, `rls_write()`, `rls_hq()`. `rls_write` é **branch exato** — Sede **não** escreve em outra filial.
- `financial_transactions` e `audit_log` são **append-only com hash-chain** (triggers bloqueiam UPDATE/DELETE).
- Login usa `auth_lookup_user` (`SECURITY DEFINER`) porque o tenant ainda não é conhecido nesse ponto.
- ⚠️ **Não existe verificação de permissão no servidor.** Não há `requirePerm` nem 403: o RBAC por `permissions` é aplicado **apenas no frontend** (`hasPerm`), e no backend a autorização é RLS + `app.role` dentro do SQL. Um endpoint de escrita novo fica acessível a **qualquer usuário autenticado** do tenant, salvo se o RLS bloquear.

### Migrações

`db/migrations/NNNNNN_nome_curto.{up,down}.sql`, embutidas no binário via `//go:embed` (`db/embed.go`) e aplicadas **no boot da API** (`internal/store/migrate.go`), numa conexão separada com o DSN do migrador em *simple protocol* (permite múltiplos comandos por arquivo).

Detalhes que importam:

- A chave de versão é o **nome completo do arquivo sem `.up.sql`**, e a ordem é `sort.Strings` (lexicográfica). O prefixo `000017` está **duplicado** (`announcement_deliveries` e `member_enhancements`): ambos aplicam. A última é a `000020_cargos` — **a próxima migração deve ser `000021_`.**
- Arquivos `.down.sql` são embutidos mas **nunca executados** — nenhum código os lê (rollback é manual).
- Não há advisory lock: não suba duas instâncias da API aplicando migrações ao mesmo tempo.
- `db/init/setup.sql` cria a role da aplicação e é exposto como `db.Setup`; precisa rodar **em cada database** (GRANT é database-scoped).

### Webadmin (Next.js)

- **A API é same-origin por caminho relativo.** O browser chama `/api/v1/...` sem host nenhum e resolve contra a origem que ele abriu; quem traduz isso para a API é o `rewrites()` do `next.config.ts` (`/api/:path*` → `API_ORIGIN`, default `http://localhost:38080`). O rewrite é configuração de **servidor**: o destino fica no `routes-manifest.json` do build, mas **nenhum endereço é gravado no bundle do cliente** — o mesmo build serve `localhost:33000` e o domínio público. Para apontar para outra API, mude `API_ORIGIN` e reinicie o `next start` (não precisa rebuildar). Antes havia um `env: { API_URL }` gravado em build time, o que amarrava o bundle a um endereço — foi o que fez o `localhost:33000` mandar o browser para o domínio público (ver incidente em `Docs/00_Checkpoint.md`). Hoje não há CORS em jogo: as chamadas são same-origin.
- `lib/api.ts`: tokens vivem em variáveis de módulo + `localStorage` (`chosen_token`, `chosen_refresh`, `chosen_user`). **As funções não recebem token** — o helper `api()` injeta o Bearer e, em **401, faz refresh e repete a requisição uma vez**; se o refresh falha, `clearTokens()` + `onSessionLost` (o AuthProvider registra `logout`). Use `apiRaw()` para FormData/HTML/blob (não força `Content-Type: application/json`). O arquivo também exporta os helpers de URL que os componentes devem usar em vez de montar caminho à mão: `cardPrintURL`, `cardPhotoURL` e `assetURL` (traduz um caminho devolvido pelo backend — ex.: `photo_url` — em URL para o browser; hoje é quase passagem, já que as origens coincidem, mas é o único ponto que muda se elas voltarem a divergir).
- `useAuth()` → `{ user, ready, login, logout, hasPerm }`. `hasPerm` é **match exato de string** num array plano (sem hierarquia de papéis, sem wildcard). O catálogo de permissões vem do banco (seed em `000009_seed_demo.up.sql`), não de um enum no TypeScript.
- Menu lateral: array `NAV_SECTIONS` **inline em `app/dashboard/layout.tsx`** (agrupado por Pessoas/Financeiro/Organização), filtrado por `perms.some(hasPerm)` — semântica **OR**, array vazio = sempre visível. `NAV_BY_PERMISSION` e `OFFICES` foram **removidos** de `lib/constants.ts`: a única fonte de verdade do menu é o `layout.tsx`, e a de cargos é a API (`/api/v1/cargos`).
- Proteção de rota é **só client-side** (em `dashboard/layout.tsx`, via `router.replace("/")`). **Não existe `middleware.ts`.**
- Tailwind v4 **CSS-first**: não há `tailwind.config.*`. Os tokens são CSS custom properties em `app/globals.css` (`--ink`, `--paper`, `--card`, `--brand`, ...) e existem classes utilitárias próprias (`.btn-primary`, `.input`, `.card`) — `bg-brand` **não existe**. Os componentes de `components/ui/*` são artesanais e usam `cn()` (`lib/utils.ts`, clsx + tailwind-merge). Reutilize-os em vez de escrever classes avulsas.
- `lib/swr-hooks.ts` tem um `fetcher` próprio que lê o token do `localStorage` **direto, sem passar pelo `api()`** — logo, **não** faz refresh em 401. Chamadas via SWR e chamadas via `api()` se comportam de forma diferente em caso de token expirado.

### Autenticação

JWT access (15 min) + refresh (168 h), claims `{user_id, tenant_id, branch_id, role, type}` — **tenant, filial e papel viajam dentro do token** e não são reconsultados a cada request. O middleware só valida o Bearer e injeta as claims no contexto (chave `claims`).

`GET /api/v1/me` devolve **a mesma forma** do payload do login (inclui `permissions`) porque o `AuthProvider` reidrata a sessão por ele após um reload — mantenha os dois em sincronia.

## Armadilhas conhecidas

- **`pastor.norte@demo.local` / filial Norte não são criados por migração.** O seed (`000009`) cria só o tenant `demo`, a filial "Sede Matriz" e `admin@demo.local`. A conta **existe no banco de desenvolvimento atual** porque foi criada à mão — ou seja, um banco recriado do zero não a terá. Ver backlog #40.
- Testes de RLS só rodam com o Docker de pé e os DSNs exportados; caso contrário a suíte é pulada **em silêncio** (parece verde).
- `Docs/` contém artefatos não versionados do cliente (`exemplos/`, `requisitos_basicos.txt`) e um `.zip` de ~1,1 MB — evite adicioná-los ao índice.
