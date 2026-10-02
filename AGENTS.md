# Chosen ERP - Guia de Desenvolvimento

SaaS eclesiastico multi-tenant. Backend em **Go** (monolito modular), banco
**PostgreSQL 17** com Row-Level Security, frontend **Next.js** e infra em **Docker Compose**.

Este arquivo e a fonte das instrucoes de build/execucao para agentes e devs.

---

## Estrutura

```
apps/webadmin/        Painel administrativo (Next.js 15, App Router, Tailwind v4) + Dockerfile
apps/marketing/       Site institucional / landing (Next.js 15, Tailwind v4) + Dockerfile
apps/member/          App do Membro / PWA mobile-first (Next.js 15, Tailwind v4) + Dockerfile
cmd/api/              Entrypoint do servico Go (monolito modular)
internal/             Dominios: auth, store (pool+migracoes+RLS), httpapi, members
db/migrations/        Migracoes SQL versionadas (embutidas no binario via Go embed)
db/init/setup.sql     Criacao do papel de aplicacao (ref. infra/postgres/initdb)
infra/                docker-compose.yml, Dockerfile da API, nginx, Prometheus/Grafana
Docs/                 PRD, blueprint e checkpoints
```

## Como rodar

### Inicio rapido (arquivo unico)

```powershell
.\start.ps1            # tudo: .env, build do Go e Docker (postgres + api + webadmin + nginx)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus e grafana
```

### 1. Infra + API + Webadmin em Docker

```bash
# sobe os containers (postgres + api + webadmin + nginx). Stack completa opcional:
#   docker compose --profile full up -d
docker compose -f infra/docker-compose.yml up -d

# a API aplica as migracoes automaticamente ao iniciar (papel migrador)
# o webadmin (Next.js standalone) e buildado na primeira subida
```

Copie `.env.example` para `.env` e ajuste credenciais se necessario.

### 2. Webadmin (dev local, opcional)

Em producao o webadmin roda no Compose (`chosen-webadmin`, porta 33000). Para
desenvolver o frontend com hot-reload no host:

```bash
cd apps/webadmin
npm install
npm run dev        # http://localhost:33000
```

### Webadmin - boas praticas de frontend

- **Design system** em `apps/webadmin/components/ui/*` (Button, Card, Badge, Table,
  Modal/Drawer, Tabs, Toast, Skeleton, EmptyState, Pagination, Avatar, StatCard,
  PageHeader). Reutilize em vez de criar classes avulsas.
- **Sessao/RBAC:** usar `useAuth()` (`components/providers/auth-provider.tsx`).
  O cliente `lib/api.ts` faz login, guarda tokens e **auto-refresca em 401**.
  Nunca passar token como argumento - as funcoes ja usam a sessao.
- Menu lateral e **filtrado por permissao** (`hasPerm`); nav definida no `layout.tsx`.
- Tema dark (toggle em `components/theme-toggle.tsx`). Helpers em `lib/format.ts`
  (`currency`, `datePt`, `relativePt`) e `lib/constants.ts` (rotulos de status).
- **Dependencias:** `tailwind-merge`+`clsx` para classes e `recharts` para graficos.

### Site institucional (marketing)

O site publico fica em `apps/marketing` (Next.js 15) e e servido no
**dominio central** (`erpchosen.com.br` / `www`). O painel administrativo fica
em **`app.erpchosen.com.br`** (sem igreja fixa: o usuario escolhe a igreja) e o
**app do membro** nos subdominios de igreja (`{slug}.erpchosen.com.br`). O split
e feito por `server_name` no `infra/nginx/conf.d/default.conf`: blocos EXATOS
(`app`/`admin` -> webadmin; apex/www -> marketing) tem prioridade sobre o regex
wildcard, que cai no **member**. `app`/`www`/`api`/`admin` sao `RESERVED_SLUGS`
no frontend, entao nao colidem com igrejas.

```bash
cd apps/marketing
npm install
npm run dev        # http://localhost:33000 (proxied API em API_ORIGIN)

# no Compose: sobe junto com o stack na porta ${MARKETING_PORT:-33010}
docker compose -f infra/docker-compose.yml up -d marketing
```

O formulario de contato chama `POST /api/v1/public/leads`, que grava em
`marketing_leads` (migracao `000060`; sem tenant/RLS e com **apenas INSERT**
para o papel da app). CTAs usam `NEXT_PUBLIC_APP_URL` e `NEXT_PUBLIC_WHATSAPP`
(build args do Compose: `MARKETING_APP_URL` / `MARKETING_WHATSAPP`).

### App do Membro (PWA)

`apps/member` e o PWA mobile-first do membro, servido na raiz do subdominio da
igreja (`{slug}.erpchosen.com.br`). Reusa a MESMA API do webadmin - backend
unificado. O membro so enxerga o proprio cadastro: o vinculo identidade <->
pessoa vive em `memberships.member_id` (migracao `000073`) e as rotas
`/api/v1/me/*` derivam o membro do token (nunca de um id do cliente).

```bash
cd apps/member
npm install
npm run dev        # http://localhost:33020 (proxied API em API_ORIGIN)

# no Compose: sobe junto com o stack na porta ${MEMBER_PORT:-33020}
docker compose -f infra/docker-compose.yml up -d member
```

- **Vinculo de acesso:** o membro precisa de uma identidade (`users`) com
  membership no tenant (papel `membro`) e do vinculo `memberships.member_id`.
  Backfill por e-mail: `go run ./cmd/link-members` (idempotente; `-dry-run` para
  simular). Membros sem e-mail no cadastro nao sao vinculados automaticamente.
- **Pedidos de oracao** (migracao `000074`): `prayer_requests` com visibilidade
  (`pastor`/`pastor_conselho`/`grupo`/`igreja`) + `prayer_reactions` (reacao
  anonima "estou orando"). Endpoints `/api/v1/me/prayer-requests*`; moderacao em
  `/api/v1/prayer-requests` (permissoes `prayer.read`/`prayer.moderate`).
- **PWA white-label:** o manifest e gerado por igreja em
  `app/manifest.webmanifest/route.ts` (nome/cor/logo do branding do tenant). O
  app e instalavel (service worker `public/sw.js`, tela `/offline`, icones PNG)
  e oferece instalacao (banner/`beforeinstallprompt` + dica iOS).
- **Automacoes do membro:** Aniversariantes (`/me/birthdays`), Ministerios
  (`/me/ministries`) e Contribuir com Pix + historico (`/me/contributions`). O
  Pix da igreja vive em `tenants.pix_key`/`pix_name` (`000080`) e sai no
  `public_tenant`; edite em Configuracoes -> Igreja.
- **Grupos / GD (`000081`)**: `group_members` vincula membro <-> grupo com papel
  (member/host/secretary/leader); `GET /me/groups` alimenta a tela "Meu GD".
  Participantes geridos em Ministerios -> aba Grupos -> **Membros**.
- **Materiais de estudo (`000082`)**: `study_materials` (arquivo em disco com nome
  opaco ou link), geral (sem grupo) ou por grupo; upload/lista/exclusao em
  Materiais (permissoes `ministries.*`), membro ve os gerais + dos seus grupos em
  `/me/materials` e baixa por `/me/materials/{id}/file` (autenticado).
- **Build args do Compose:** `API_ORIGIN`, `NEXT_PUBLIC_BASE_DOMAIN`.
- **Acesso do membro:** a Sede cria o acesso em Membros -> aba **Acesso**, que
  grava a identidade + senha provisoria e linka `memberships.member_id` via
  `user_attach_member` (`000076`). O login aceita **e-mail ou telefone**
  (`users.phone`, normalizado e unico). Senha provisoria (`must_change_password`)
  forca a troca no 1o acesso (tela `/trocar-senha`).

### Console da plataforma (admin do SaaS)

`users.is_platform_admin` (`000075`) distingue o operador do SaaS do
`super_admin` de uma igreja. O admin de plataforma **nao tem vinculo de igreja**
(`000077` remove memberships de quem tem a flag): o login emite tokens **sem
tenant** e papel `platform_admin` (o RLS nao devolve dados operacionais). Com a
flag, o webadmin mostra apenas o menu **Plataforma** (Visao geral, Igrejas,
Planos): edita dados/plano/limites/branding, suspende/reativa a igreja, gerencia
o catalogo `plans` e ve estatisticas gerais (`GET /admin/stats`). No Gerenciar de
cada igreja ha a aba **Usuarios** (suporte): listar/criar acessos, definir
perfis (administradores), ativar/desativar e redefinir senha — via
`/admin/tenants/{id}/users*` (roda em `WithSystem`, sem impersonation, com
registro em `audit_log`). Antes, qualquer `super_admin` de igreja listava e
criava tenants (corrigido: o gate agora e `is_platform_admin()`). Enforcement de
limites ainda nao existe (so gerencia/exibe).

### Modulos por plano (entitlements)

`plans.features` (jsonb) define os modulos de cada plano e `tenants.features`
(jsonb) e o override por igreja; a feature efetiva e o plano sobreposto pelo
override, com **chave ausente = habilitada** (`000078`). Modulos core (pessoas,
admin, app do membro) nunca sao bloqueados. O catalogo fica em
`internal/org/entitlements.go` (exposto em `GET /admin/features`). O enforcement
e no backend: um middleware mapeia rota->feature e devolve **403
`plan_feature_disabled`**; o menu do webadmin so esconde (nao e a seguranca).
Quotas (`max_members/max_branches/max_users/max_storage_mb`) bloqueiam criacao
(POST membros/filiais/usuarios/acesso e uploads) com 403 `limite do plano
atingido`. A resolucao roda no escopo da igreja, por isso a policy de `plans`
libera a leitura do proprio plano (`000079`).

### 3. Servico Go localmente (para depurar)

```bash
# build do binario
go build ./...
go run ./cmd/api
```

### 4. Testes

Os testes de RLS precisam de um PostgreSQL real (Docker de pe). Eles **nao tocam**
no banco de desenvolvimento: criam e recriam um banco descartavel
`chosenerp_test`.

```powershell
# usa os mesmos DSNs do .env (so para derivar o banco de teste)
$env:CHOSEN_TEST_MIGRATE_URL="postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"
$env:CHOSEN_TEST_APP_URL="postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"

go test ./... -count=1
```

Variaveis:

| Var | Fallback | Papel |
|---|---|---|
| `CHOSEN_TEST_MIGRATE_URL` | `MIGRATE_DATABASE_URL` | dono/superuser (cria o banco e aplica migracoes) |
| `CHOSEN_TEST_APP_URL` | `DATABASE_URL` | `chosenerp_app`, sujeito a RLS |
| `CHOSEN_TESTS_REQUIRED=1` | - | **falha** em vez de pular, quando os DSNs nao existem (use no CI) |

Se os DSNs nao estiverem definidos, a suite e pulada com aviso.

Antes de commitar: `go build ./... && go vet ./... && go test ./... -count=1`.

---

## Credenciais de desenvolvimento (seed)

- Admin (Sede): `admin@demo.local`  (papel `super_admin`)
- Filial Norte: `pastor.norte@demo.local`

As senhas **nao sao versionadas**: ficam no `.env` (gitignorado), em
`DEMO_ADMIN_PASSWORD` e `DEMO_NORTE_PASSWORD`. Leia com `grep '^DEMO_' .env`.

>  `pastor.norte@demo.local` (e a filial Norte) **nao e criado por
> nenhuma migracao** - o seed (000009) cria apenas o tenant `demo`, a filial
> "Sede Matriz" e o `admin@demo.local`. Ver `Docs/02_Backlog.md` (#40).
> A conta existe no banco de desenvolvimento atual porque foi criada a mao.

---

## Seguranca multi-tenant (como funciona)

- Duas roles de banco:
  - **migrador** (`postgres`): dono do schema, executa as migracoes `db/migrations`.
  - **app** (`chosenerp_app`): conexao da API. **Nao e dono nem superuser**  o
    PostgreSQL aplica Row-Level Security em todas as queries.
- Cada requisicao roda dentro de uma transacao com
  `set_config('app.tenant_id'/'app.branch_id'/'app.role'/'app.branch_scope', ...)`.
  As politicas RLS (`000007_rls_policies.up.sql`, revisadas pela migracao
  `000016_tenant_scope_rls.up.sql`) filtram por **tenant e** `branch_id`; escopo
  "Sede" (branch NULL) enxerga **todo o seu tenant**, e escopo `system` (workers)
  enxerga tudo.
- **Hierarquia de filiais (sub-congregacoes, `000036`):** o gateway grava em
  `app.branch_scope` a filial do contexto + descendentes. A **leitura** inclui os
  descendentes (`rls_read_scope`); a **escrita** continua no branch exato. A
  migracao `000037` corrigiu as politicas `*_sel`, que eram `FOR ALL` e davam
  escrita pela leitura - hoje sao `FOR SELECT`.
- **Tipo de unidade (`000055`):** o `branches.kind` segue a estrutura de governo
  **Matriz** (Sede Administrativa, raiz - sem superior), **Filial**
  (regional/igreja local) e **PAE** (Ponto de Atendimento de Evangelizacao,
  obrigatoriamente vinculado a uma Matriz ou Filial). As regras sao garantidas
  por CHECK + trigger `branches_validate_hierarchy` no banco e validadas em
  `internal/org` (`validateBranchKind`). O padrao de uma nova unidade e
  `filial`.
- **Canais por filial (`000056`):** cada filial pode ter o proprio WhatsApp
  (instancia Evolution nomeada com o **id da filial**, conectada por QR) e o
  proprio SMTP. A senha do SMTP nunca e devolvida. No envio, o `Dispatcher`
  resolve a config da filial (`Message.TenantID/BranchID`) e cai no provedor
  global quando a filial nao tem canal proprio. API em
  `internal/httpapi/branch_channels_handlers.go`; UI em Configuracoes -> Filiais.
- **Escrita pela Sede (`000045`):** `rls_write` passou a incluir `is_headquarters()`,
  entao a Sede (branch NULL + `super_admin`/`admin_sede`) mantem registros de
  qualquer filial do proprio tenant. Antes, UPDATE/DELETE da Sede afetavam 0
  linhas e viravam 404 (ex.: editar evento, estornar lancamento). Quem tem filial
  continua gravando so no branch exato.
- **Troca de contexto filial/Sede:** o header `X-Branch-Id` permite que
  `super_admin`/`admin_sede` escolham a filial de trabalho por requisicao. O
  middleware (`internal/httpapi/middleware.go`) sobrescreve `claims.BranchID`:
  ausente = escopo do token, `all` = Sede (todas as filiais), uuid = opera como
  aquela filial (leitura e gravacao). Ignorado para os demais papeis. No webadmin
  o seletor fica na topbar; a escolha vai em `localStorage` e acompanha todo
  request via `lib/api.ts`.
- **Identidade global + multi-igreja (`000053`):** `users` guarda so a
  identidade (e-mail UNIQUE global, senha, MFA). O vinculo pessoa  igreja vive
  em `memberships` (`user_id`, `tenant_id`, `role_id`, `branch_id`,
  `is_active`; UNIQUE `user_id,tenant_id`). O login (`auth_lookup_user`) resolve
  a identidade e `auth_memberships(user_id)` lista as igrejas: com 1 vinculo
  entra direto; com >1 devolve `requires_tenant_selection` +
  `selection_token` (`typ=select`, 5 min) + lista, e o front chama
  `POST /auth/select-tenant`. Ja autenticado, `POST /auth/switch-tenant` troca a
  igreja. O JWT continua carregando o tenant ATIVO (`tid/bid/role`); o RLS de
  `users` (`id = current_user_id()` ou membership no tenant) e de `memberships`
  usa o GUC `app.user_id`, setado por `store.WithTenant`. O middleware aceita
  `X-Tenant-Id`/`X-Tenant-Slug` (validando vinculo ativo) alem do `X-Branch-Id`.
- **Subdominio / white-label (`000054`):** `tenants` ganhou
  `logo_url/brand_color/favicon_url/custom_domain`; `public_tenant(slug)`
  (SECURITY DEFINER) alimenta a tela de login do subdominio. O nginx
  (`infra/nginx/conf.d/default.conf`) tem `server_name` wildcard
  (`*.erpchosen.com.br`, com o dominio antigo como alias) e repassa
  `X-Tenant-Slug`; a Cloudflare precisa do DNS wildcard + TLS `*.dominio`. O
  dominio base do webadmin vem de `PUBLIC_BASE_DOMAIN` (build arg
  `NEXT_PUBLIC_BASE_DOMAIN`). No webadmin, o slug e detectado via
  `tenantSlugFromHost()` (`lib/api.ts`) e ha seletor de igreja na topbar quando a
  identidade tem mais de um vinculo.

### Cloudflare Tunnel (dominio `erpchosen.com.br`)

O `cloudflared` roda apontando para o nginx do stack (`http://localhost:80` se
roda no host, ou `http://nginx:80` se roda como servico no Compose). Na
Cloudflare (dashboard do tunnel) cadastre os public hostnames:

- `erpchosen.com.br` -> `http://localhost:80` (ou `http://nginx:80`)
- `www.erpchosen.com.br` -> idem
- `*.erpchosen.com.br` -> idem (wildcard, cobre as igrejas)

O subdominio de cada igreja e `{tenant.slug}.erpchosen.com.br` (ex.: o tenant
`demo` responde em `demo.erpchosen.com.br`). Para o wildcard funcionar:

1. Zona `erpchosen.com.br` com os nameservers da Cloudflare.
2. Um CNAME `*` -> `<tunnel-id>.cfargotunnel.com` (Proxy laranja ligado). O
   proprio dashboard cria os CNAMEs dos hostnames exatos; o wildcard precisa ser
   adicionado a mao (ou via `cloudflared` com a regra wildcard no `config.yml`).
3. TLS: o certificado Universal da Cloudflare cobre `erpchosen.com.br` e
   `*.erpchosen.com.br` (um nivel). O trafego Cloudflare<->origem vai em HTTP
   pela tunnel; o nginx pode continuar so na 80.
4. No webadmin, o login do subdominio envia `tenant_slug=<slug>`; se a
   identidade nao tiver vinculo ativo com aquela igreja, o login responde 403
   (`tenant_forbidden`).

> **Onboarding de igreja (`000059`):** `create_tenant(name, slug, plan)` (SECURITY
> DEFINER) cria a igreja com a base minima (tenant + Matriz + papeis padrao +
> permissoes) e `list_tenants()` lista tudo. Na API, `POST /api/v1/admin/tenants`
> (super_admin) chama isso e anexa o primeiro `super_admin` via
> `user_attach_to_tenant`. Como o wildcard de DNS/nginx ja cobre qualquer
> subdominio, **o `{slug}.erpchosen.com.br` passa a funcionar na hora** (tela em
> Configuracoes -> aba "Igrejas"). O slug e validado (minusculas/numeros/hifen,
> 2-39 chars, sem reservados) e unico.

> `erpchosen.com.br` esta em uma **conta Cloudflare separada** da do dominio
> `mgmconsultoria.com`, entao tem um **tunnel exclusivo** (`cloudflared-erpchosen`,
> profile `erpchosen` do Compose). O config/credenciais ficam em
> `infra/cloudflare/` (o JSON de credenciais e gitignored). Detalhes em
> `infra/cloudflare/README.md`. Suba com
> `docker compose --profile erpchosen up -d cloudflared-erpchosen`.
- O isolamento e verificado por **testes automatizados** em
  `internal/store/rls_test.go` e `internal/store/memberships_test.go` (varredura
  de todas as tabelas, alem de casos dedicados de identidade/membership).
- `financial_transactions` e `audit_log` sao **append-only** (triggers impedem
  UPDATE/DELETE) com **hash-chain** de integridade.
- Login usa funcao `SECURITY DEFINER` (`auth_lookup_user`) para localizar o usuario
  **fora** do escopo RLS (o tenant ainda nao e conhecido).

### Teste manual de isolamento

```
docker exec chosen-postgres psql -U postgres -d chosenerp \
  -c "SELECT set_config('app.branch_id','<branchA>',true); SELECT * FROM members;"
```

---

## Endpoints da API (base `http://localhost:38080`)

| Metodo | Rota | Auth | Descricao |
|---|---|---|---|
| GET  | `/healthz` | - | Health check |
| GET  | `/metrics` | - | Metricas Prometheus |
| POST | `/api/v1/auth/login` | - | Login por e-mail OU telefone (access + refresh); `tenant_slug` opcional; pode exigir selecao de igreja |
| POST | `/api/v1/auth/select-tenant` | selection token | Conclui o login escolhendo a igreja |
| POST | `/api/v1/auth/switch-tenant` | Bearer | Troca a igreja ativa (novos tokens) |
| GET  | `/api/v1/auth/refresh` | refresh | Renova tokens |
| GET  | `/api/v1/me` | Bearer | Perfil + contexto + memberships |
| GET  | `/api/v1/me/tenants` | Bearer | Igrejas da identidade (seletor) |
| GET  | `/api/v1/public/tenant/{slug}` | - | Branding publico da igreja (login do subdominio) |
| POST | `/api/v1/public/leads` | - | Lead do site institucional (grava `marketing_leads`) |
| PATCH | `/api/v1/me` | Bearer | Edita o proprio perfil (nome/e-mail) |
| POST | `/api/v1/me/password` | Bearer | Troca a propria senha (senha atual + nova) |
| GET  | `/api/v1/me/member` | Bearer | Cadastro de membro vinculado a identidade (app do membro) |
| PATCH | `/api/v1/me/member` | Bearer | Edita contato/endereco do proprio membro |
| GET  | `/api/v1/me/family` | Bearer | Familias do membro |
| GET  | `/api/v1/me/events` | Bearer | Agenda no escopo do membro |
| GET  | `/api/v1/me/announcements` | Bearer | Avisos ativos da igreja/filial do membro |
| GET  | `/api/v1/me/birthdays?month=` | Bearer | Aniversariantes de nascimento e casamento do mes |
| GET  | `/api/v1/me/ministries` | Bearer | Ministerios dos quais o membro participa |
| GET  | `/api/v1/me/contributions?year=` | Bearer | Contribuicoes (entradas) do proprio membro no ano |
| GET  | `/api/v1/me/groups` | Bearer | Grupos/celulas do membro (participantes e papeis) |
| GET  | `/api/v1/me/materials` | Bearer | Materiais de estudo publicados (gerais + dos meus grupos) |
| GET/POST | `/api/v1/me/prayer-requests` | Bearer | Meus pedidos de oracao / criar pedido |
| GET  | `/api/v1/me/prayer-wall` | Bearer | Mural publico (visibilidade `igreja`) |
| POST | `/api/v1/me/prayer-requests/{id}/react` | Bearer | "Estou orando" (reacao anonima) |
| GET  | `/api/v1/prayer-requests` | Bearer (`prayer.read`) | Moderacao: lista pedidos por visibilidade |
| PATCH | `/api/v1/prayer-requests/{id}` | Bearer (`prayer.moderate`) | Modera o pedido (status/observacao) |
| GET  | `/api/v1/members` | Bearer | Lista membros (escopo RLS) |
| POST | `/api/v1/members` | Bearer | Cria membro (escopo RLS) |
| GET  | `/api/v1/members/{id}` | Bearer | Detalhe de membro |
| GET  | `/api/v1/members/{id}/tree` | Bearer | Arvore genealogica + discipulado |
| PATCH | `/api/v1/members/{id}` | Bearer | Edita perfil do membro |
| DELETE | `/api/v1/members/{id}` | Bearer (Sede) | Exclui membro definitivamente (so `super_admin`/`admin_sede`; cascata nos vinculos) |
| POST | `/api/v1/members/{id}/relationships` | Bearer | Cria vinculo (conjuge/filho/discipulo...) |
| POST | `/api/v1/members/{id}/photo` | Bearer | Envia foto (multipart, `UPLOAD_DIR`, disco local) |
| DELETE | `/api/v1/members/{id}/photo` | Bearer | Remove a foto do membro |
| GET  | `/api/v1/members/{id}/access` | Bearer (Sede) | Situacao do acesso do membro ao app |
| POST | `/api/v1/members/{id}/access` | Bearer (Sede) | Cria acesso (e-mail/telefone + senha provisoria) |
| PATCH | `/api/v1/members/{id}/access` | Bearer (Sede) | Altera identificador / ativa-desativa acesso |
| POST | `/api/v1/members/{id}/access/password` | Bearer (Sede) | Redefine a senha (provisoria) |
| POST | `/api/v1/members/{id}/card` | Bearer | Emite carteirinha QR - **idempotente** (devolve `card_ref` + `token`) |
| GET  | `/api/v1/members/{id}/card` | Bearer | Le a carteirinha ja emitida (`card_ref` + `token`) |
| GET  | `/api/v1/members/{id}/families` | Bearer | Familias das quais o membro participa |
| GET  | `/api/v1/cargos` | Bearer | Catalogo de cargos do tenant |
| POST | `/api/v1/cargos` | Bearer | Cria cargo (`name`, `kind`, `requires_term`, `sort_order`) |
| PATCH | `/api/v1/cargos/{id}` | Bearer | Edita/ativa/desativa cargo |
| DELETE | `/api/v1/cargos/{id}` | Bearer | Exclui cargo (**409** se houver mandatos - desative) |
| GET  | `/api/v1/members/{id}/cargos` | Bearer | Mandatos do membro (historico completo) |
| POST | `/api/v1/members/{id}/cargos` | Bearer | Atribui cargo (`started_at`, `ends_at`, `status`) |
| PATCH | `/api/v1/members/{id}/cargos/{linkId}` | Bearer | Edita mandato; `ends_at: ""` **limpa** o vencimento |
| DELETE | `/api/v1/members/{id}/cargos/{linkId}` | Bearer | Remove o mandato do historico |
| GET  | `/api/v1/member-event-kinds` | Bearer | Catalogo configuravel de eventos da vida eclesiastica (000065) |
| POST | `/api/v1/member-event-kinds` | Bearer | Cria tipo de evento (e define o que ele movimenta) |
| PATCH | `/api/v1/member-event-kinds/{id}` | Bearer | Edita/ativa/desativa tipo de evento |
| DELETE | `/api/v1/member-event-kinds/{id}` | Bearer | Exclui tipo de evento (**409** se houver historico - desative) |
| GET  | `/api/v1/members/{id}/history` | Bearer | Vida eclesiastica do membro (append-only + hash-chain) |
| POST | `/api/v1/members/{id}/history` | Bearer | Lanca evento (`event_kind_id`/`kind`); aplica os efeitos do tipo no membro |
| GET  | `/api/v1/members/{id}/frequency` | Bearer | Historico de frequencia |
| POST | `/api/v1/members/{id}/frequency` | Bearer | Atualiza a frequencia (mantem historico) |
| GET  | `/api/v1/event-kinds` | Bearer | Tipos de evento (catalogo SELADO, somente leitura - 000069) |
| GET  | `/api/v1/events?from=&to=&kind=` | Bearer | Lista eventos (agenda unificada; cada um tem `origin`) |
| POST | `/api/v1/events` | Bearer | Cria evento AVULSO (data/hora, tipo, total) |
| GET  | `/api/v1/events/{id}` | Bearer | Detalhe do evento |
| PATCH | `/api/v1/events/{id}` | Bearer | Edita evento |
| DELETE | `/api/v1/events/{id}` | Bearer | Exclui evento |
| GET  | `/api/v1/events/{id}/attendance` | Bearer | Chamada nominal do evento |
| POST | `/api/v1/events/{id}/attendance` | Bearer | Salva chamada nominal + total |
| GET  | `/api/v1/events/{id}/invitees` | Bearer | Convocados (pessoas/ministerios) |
| POST | `/api/v1/events/{id}/invitees` | Bearer | Define os convocados do evento |
| GET  | `/api/v1/programacoes` | Bearer | Grade de horarios recorrentes (000069) |
| POST | `/api/v1/programacoes` | Bearer | Cria horario (kind fixo: culto/oracao/celula/ebd/ensaio/reuniao/outro) |
| POST | `/api/v1/programacoes/generate` | Bearer | Publica as ocorrencias na agenda (periodo; idempotente) |
| PATCH | `/api/v1/programacoes/{id}` | Bearer | Edita o horario |
| DELETE | `/api/v1/programacoes/{id}` | Bearer | Exclui o horario (eventos publicados permanecem) |
| GET  | `/api/v1/consent-terms` | Bearer | Termos de consentimento (LGPD) |
| POST | `/api/v1/consent-terms` | Bearer (Sede) | Cria termo de consentimento |
| GET  | `/api/v1/members/{id}/consents` | Bearer | Consentimentos do membro |
| POST | `/api/v1/members/{id}/consents` | Bearer | Registra/revoga consentimento |
| GET  | `/api/v1/members/{id}/export` | Bearer | Exporta os dados do titular (JSON) |
| POST | `/api/v1/members/{id}/anonymize` | Bearer (Sede) | Anonimiza os dados pessoais |
| GET/POST | `/api/v1/minutes` | Bearer | Lista/cria atas (livro digital) |
| GET/PATCH/DELETE | `/api/v1/minutes/{id}` | Bearer | Le/edita/exclui ata (bloqueia se assinada) |
| POST | `/api/v1/minutes/{id}/sign` | Bearer | Assinatura interna (hash + credenciais) |
| GET | `/api/v1/minutes/{id}/signatures` | Bearer | Assinaturas da ata (hash-chain) |
| GET/POST | `/api/v1/votes` | Bearer | Lista/cria votacoes (quorum + voto secreto) |
| GET/PATCH/DELETE | `/api/v1/votes/{id}` | Bearer | Le/edita/exclui votacao |
| POST | `/api/v1/votes/{id}/open` | Bearer | Abre a votacao |
| POST | `/api/v1/votes/{id}/ballot` | Bearer | Vota em segredo (sem vinculo eleitorvoto) |
| POST | `/api/v1/votes/{id}/close` | Bearer | Encerra, apura e anexa a ata vinculada |
| GET | `/api/v1/votes/{id}/result` | Bearer | Apuracao (contagens + quorum) |
| GET/POST | `/api/v1/legal-documents` | Bearer | Convenios/documentos legais com vencimento |
| PATCH/DELETE | `/api/v1/legal-documents/{id}` | Bearer | Edita/exclui documento legal |
| GET | `/api/v1/governance/mandates` | Bearer | Painel de mandatos (vigentes/vencendo) |
| GET  | `/api/v1/families` | Bearer | Lista familias |
| POST | `/api/v1/families` | Bearer | Cria familia (codigo `#NNN` gerado na transacao) |
| GET  | `/api/v1/families/{id}` | Bearer | Detalhe da familia (chefe + endereco jsonb) |
| PATCH | `/api/v1/families/{id}` | Bearer | Renomeia / troca chefe (`head_id`) / grava `address` |
| DELETE | `/api/v1/families/{id}` | Bearer | Exclui familia (vinculos de parentesco sobrevivem) |
| GET/POST | `/api/v1/families/{id}/members` | Bearer | Lista/vincula membros da familia |
| DELETE | `/api/v1/families/{id}/members/{memberId}` | Bearer | Desvincula (limpa `family_id` e o chefe, se for o caso) |
| GET  | `/api/v1/visitors` | Bearer | Lista visitantes |
| POST | `/api/v1/visitors` | Bearer | Registra visitante |
| PATCH | `/api/v1/visitors/{id}/stage` | Bearer | Avanca trilha de acolhimento |
| GET/POST | `/api/v1/benefactors` | Bearer | Lista/cadastra benfeitores |
| GET/POST | `/api/v1/suppliers` | Bearer | Lista/cadastra fornecedores (CPF/CNPJ opcionais) |
| PATCH/DELETE | `/api/v1/suppliers/{id}` | Bearer | Edita/exclui fornecedor |
| GET  | `/api/v1/finance/categories` | Bearer | Plano de contas |
| POST | `/api/v1/finance/categories` | Bearer | Cria categoria |
| PATCH | `/api/v1/finance/categories/{id}` | Bearer | Edita/ativa/desativa categoria |
| DELETE | `/api/v1/finance/categories/{id}` | Bearer | Exclui categoria (409 se em uso - desative) |
| GET/POST | `/api/v1/finance/category-groups` | Bearer | Grupos de contas (agrupam o plano de contas) |
| PATCH/DELETE | `/api/v1/finance/category-groups/{id}` | Bearer | Edita/exclui grupo (contas ficam sem grupo) |
| GET  | `/api/v1/finance/accounts` | Bearer | Contas bancarias |
| POST | `/api/v1/finance/accounts` | Bearer | Cria conta bancaria |
| PATCH | `/api/v1/finance/accounts/{id}` | Bearer | Edita/ativa/desativa conta |
| DELETE | `/api/v1/finance/accounts/{id}` | Bearer | Exclui conta (409 se em uso - desative) |
| GET  | `/api/v1/finance/transactions?type=` | Bearer | Lista lancamentos (com account_id) |
| POST | `/api/v1/finance/transactions` | Bearer | Lanca dizimo/oferta/despesa (+ recibo auto) |
| POST | `/api/v1/finance/transactions/batch` | Bearer | Lanca varios de uma vez (grid em lote; erros por linha) |
| POST | `/api/v1/finance/transactions/import` | Bearer | Importa lancamentos (CSV ou XLSX, com mapeamento de colunas) |
| POST | `/api/v1/finance/transactions/import/preview` | Bearer | Pre-visualiza as linhas da planilha para mapear colunas |
| POST | `/api/v1/finance/transactions/reorder` | Bearer | Renumera a sequencia (entry_seq) dos lancamentos da mesma conta/data |
| DELETE | `/api/v1/finance/transactions/{id}` | Bearer | Exclui o lancamento definitivamente (recalcula hash-chain) |
| POST | `/api/v1/finance/transactions/{id}/void` | Bearer | Estorna o lancamento (legado; sai dos relatorios) |
| GET  | `/api/v1/finance/transactions/{id}/events` | Bearer | Rateio do lancamento por evento |
| PATCH | `/api/v1/ministries/{id}` | Bearer | Edita ministerio (responsavel/situacao) |
| DELETE | `/api/v1/ministries/{id}` | Bearer | Exclui ministerio |
| PATCH | `/api/v1/groups/{id}` | Bearer | Edita grupo/celula |
| DELETE | `/api/v1/groups/{id}` | Bearer | Exclui grupo/celula |
| GET/POST | `/api/v1/groups/{id}/members` | Bearer | Lista/vincula participantes do grupo (papel: member/host/secretary/leader) |
| DELETE | `/api/v1/groups/{id}/members/{memberId}` | Bearer | Desvincula participante do grupo |
| GET/POST | `/api/v1/materials` | Bearer (`ministries.read`/`write`) | Lista/cria material de estudo (multipart arquivo ou JSON link) |
| DELETE | `/api/v1/materials/{id}` | Bearer (`ministries.write`) | Exclui o material e o arquivo |
| GET | `/api/v1/materials/{id}/file` | Bearer (`ministries.read`) | Download autenticado do arquivo (staff) |
| GET | `/api/v1/me/materials/{id}/file` | Bearer | Download autenticado pelo membro (core, RLS) |
| GET/POST | `/api/v1/rosters` | Bearer | Lista/cria escalas (evento ou tipo; `create_event` gera o evento na grade) |
| GET | `/api/v1/rosters/suggestions?ministry_id=&starts_at=` | Bearer | Sugere voluntarios (marca conflito) |
| GET/PATCH/DELETE | `/api/v1/rosters/{id}` | Bearer | Le/edita/exclui escala (`DELETE ?delete_event=true` remove o evento gerado) |
| POST | `/api/v1/rosters/{id}/assignments` | Bearer | Define os escalados (preserva confirmacoes) |
| PATCH | `/api/v1/rosters/{id}/assignments/{assignmentId}` | Bearer | Confirma/recusa presenca |
| GET | `/api/v1/rosters/{id}/conflicts` | Bearer | Conflitos de agenda da escala |
| GET  | `/api/v1/finance/transactions/{id}/attachments` | Bearer | Lista anexos de um lancamento |
| POST | `/api/v1/finance/transactions/{id}/attachments` | Bearer | Anexa documento (multipart, ate 50MB) |
| GET  | `/api/v1/attachments/{filename}` | - | Download de anexo (nome opaco) |
| GET  | `/api/v1/finance/balance` | Bearer | Balancete/DRE (entradas, saidas, por categoria) |
| GET  | `/api/v1/finance/audits` | Bearer | Lista auditorias financeiras |
| POST | `/api/v1/finance/audits` | Bearer | Cria auditoria (periodo) |
| GET  | `/api/v1/finance/audits/{id}` | Bearer | Auditoria + lancamentos do periodo |
| POST | `/api/v1/finance/audits/{id}/mark` | Bearer | Marca/desmarca auditado (lista vazia = todos) |
| POST | `/api/v1/finance/audits/{id}/close` | Bearer | Fecha e assina (documento imutavel) |
| GET  | `/api/v1/finance/audits/{id}/export?format=` | Bearer | PDF/CSV/XLSX do documento auditado |
| DELETE | `/api/v1/finance/audits/{id}` | Bearer | Exclui a auditoria e seus itens (mesmo fechada) |
| GET  | `/api/v1/documents/by-token/{token}` | Bearer | Le documento por token do QR |
| GET  | `/api/v1/receipts/{id}` | Bearer | Recibo em **HTML** pronto p/ impressao |
| POST | `/api/v1/receipts/{id}/send` | Bearer | Enfileira+envia recibo (`channel`: email\|whatsapp) |
| GET  | `/api/v1/receipts/{id}/deliveries` | Bearer | Historico de envios |
| GET  | `/api/v1/reports/balance?from=&to=` | Bearer | Balancete mensal (serie por mes) |
| GET  | `/api/v1/reports/dre?from=&to=` | Bearer | DRE por categoria + comparativo de periodo |
| GET  | `/api/v1/reports/birthdays?month=` | Bearer | Aniversariantes (nascimento + casamento) |
| GET  | `/api/v1/reports/demographics` | Bearer | Painel demografico (idade/status/UF/cidade) |
| GET  | `/api/v1/reports/balance/export?format=csv\|xlsx\|pdf` | Bearer | Exporta o balancete |
| GET  | `/api/v1/reports/dre/export?format=csv\|xlsx\|pdf` | Bearer | Exporta o DRE |
| GET  | `/api/v1/reports/birthdays/export?month=&format=` | Bearer | Exporta aniversariantes |
| GET  | `/api/v1/reports/demographics/export?format=` | Bearer | Exporta demograficos |
| GET  | `/api/v1/reports/monthly-statement?month=` | Bearer | Demonstrativo Mensal (regime de caixa) |
| GET  | `/api/v1/reports/monthly-statement/export?month=&format=csv\|xlsx\|pdf` | Bearer | Exporta o Demonstrativo Mensal |
| GET  | `/api/v1/reports/consolidated?from=&to=` | Bearer | Painel consolidado por filial (respeita a hierarquia) |
| GET  | `/api/v1/reports/assembly?from=&to=` | Bearer | Demonstrativo para assembleia (JSON) |
| GET  | `/api/v1/reports/assembly/export?from=&to=&format=` | Bearer | Exporta o Demonstrativo para assembleia |
| GET  | `/api/v1/users` | Bearer (Sede) | Lista usuarios |
| POST | `/api/v1/users` | Bearer (Sede) | Cria usuario |
| PATCH | `/api/v1/users/{id}` | Bearer (Sede) | Edita usuario (perfil/filial/ativo) |
| POST | `/api/v1/users/{id}/password` | Bearer (Sede) | Redefine a senha |
| GET  | `/api/v1/roles` | Bearer (Sede) | Perfis + permissoes |
| GET  | `/api/v1/permissions` | Bearer (Sede) | Catalogo de permissoes |
| POST | `/api/v1/admin/reset-data` | Bearer (super_admin) | Limpa dados operacionais (mantem base + super_admin); corpo `{"confirm":"RESET"}` |
| GET  | `/api/v1/admin/tenants` | Bearer (super_admin) | Lista as igrejas (onboarding) |
| POST | `/api/v1/admin/tenants` | Bearer (super_admin) | Cria igreja (tenant + Matriz + papeis + super_admin); o subdominio ja vale |
| GET  | `/api/v1/admin/tenants/{id}` | Bearer (platform admin) | Detalhe da igreja para o console |
| PATCH | `/api/v1/admin/tenants/{id}` | Bearer (platform admin) | Edita dados/plano/branding/limites/situacao (suspender/reativar) |
| GET  | `/api/v1/admin/tenants/{id}/usage` | Bearer (platform admin) | Uso da igreja (membros/filiais/usuarios/storage) |
| GET  | `/api/v1/admin/stats` | Bearer (platform admin) | Estatisticas gerais (igrejas por plano, totais, recentes) |
| GET  | `/api/v1/admin/features` | Bearer (platform admin) | Catalogo de modulos gateaveis por plano |
| GET/POST | `/api/v1/admin/tenants/{id}/users` | Bearer (platform admin) | Suporte: lista/cria acessos da igreja |
| PATCH | `/api/v1/admin/tenants/{id}/users/{userId}` | Bearer (platform admin) | Suporte: edita perfil/filial/ativo (define administradores) |
| POST | `/api/v1/admin/tenants/{id}/users/{userId}/password` | Bearer (platform admin) | Suporte: redefine senha do acesso |
| GET  | `/api/v1/admin/tenants/{id}/roles` | Bearer (platform admin) | Perfis da igreja (seletor do console) |
| GET  | `/api/v1/admin/tenants/{id}/branches` | Bearer (platform admin) | Filiais da igreja (seletor do console) |
| GET  | `/api/v1/admin/plans` | Bearer (platform admin) | Catalogo de planos |
| POST | `/api/v1/admin/plans` | Bearer (platform admin) | Cria plano |
| PATCH | `/api/v1/admin/plans/{key}` | Bearer (platform admin) | Edita plano |
| GET  | `/api/v1/auth/mfa` | Bearer | Estado do MFA |
| POST | `/api/v1/auth/mfa/setup` | Bearer | Gera segredo TOTP |
| POST | `/api/v1/auth/mfa/enable` | Bearer | Ativa MFA (codigo) |
| POST | `/api/v1/auth/mfa/disable` | Bearer | Desativa MFA |
| GET  | `/api/v1/branches` | Bearer | Lista filiais/congregacoes do tenant |
| POST | `/api/v1/branches` | Bearer (Sede) | Cria filial/congregacao (`cnpj` opcional) |
| PATCH | `/api/v1/branches/{id}` | Bearer (Sede) | Edita filial (nome/tipo/CNPJ/endereco/ativa) |
| DELETE | `/api/v1/branches/{id}` | Bearer (Sede) | Exclui filial (409 se houver membros) |
| GET  | `/api/v1/tenant` | Bearer | Dados da igreja (tenant) |
| PATCH | `/api/v1/tenant` | Bearer (Sede) | Edita dados da igreja (inclui branding e Pix) |

---

## Banco / migracoes

- Migracoes versionadas em `db/migrations/*.up.sql` (e `.down.sql`).
- O binario as embute e aplica no boot, rastreando versoes em `schema_migrations`.
- Convencao: `0000xx_nome_curto.up.sql`.
- **Vida eclesiastica (`000065`)**: `member_event_kinds` e o catalogo configuravel
  de eventos (substitui a lista fixa e os 15 tipos do legado `cadorg`). Cada tipo
  declara o que movimenta o membro (`sets_status`, `sets_exit_reason`, `clears_exit`,
  `sets_baptism`, `sets_date_field`); o `POST /members/{id}/history` aplica os
  efeitos na mesma transacao. Seed por tenant; `create_tenant` tambem semeia.
- **Situacao do membro (`000066`)**: `members.membership_status` tem so **3**
  valores - `active` (Ativo Professo), `member` (Ativo Nao Professo) e `inactive`
  (Inativo). A antiga distincao (baixado/transferido/falecido/outros) foi
  colapsada em `inactive` e vive no **motivo** (`exit_reason`, obrigatorio quando
  inativo). Os tipos de saida do catalogo (`000065`) gravam `inactive` + o motivo.
- **Importacao do legado**: `go run ./cmd/import-legacy-history` (uma vez, fora das
  migracoes). Le `cadcre`/`cadorg` do banco MCM/IPI e grava em `member_history`,
  casando o membro por `external_id = 'sincad:<cpf>'` e caindo para CPF. Idempotente.

## Build da imagem da API

O build usa `FROM scratch` (sem pull de imagens) - compile o binario Linux antes:

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o bin/api-linux-amd64 ./cmd/api
docker build -f infra/api/Dockerfile -t chosenerp/api:local .
```

O container roda como UID `65532` (nao-root) e o Dockerfile copia
`infra/api/uploads/` para `/uploads` com `--chown=65532:65532`. Isso e o que faz o
volume nomeado de uploads **nascer com o dono certo** e a API conseguir gravar fotos
de membro e anexos do financeiro - nao remova nem o diretorio nem o `--chown`.

---

## Observabilidade (opcional)

Com o profile completo (`--profile full`), Prometheus (porta 39090) e Grafana (porta 33001)
coletam metricas do servico `api`. O Prometheus aponta para `api:8080/metrics`.

## WhatsApp local (Evolution API)

O Compose sobe a **Evolution API** (`chosen-evolution`,
`evoapicloud/evolution-api:latest`) junto do stack, na mesma rede e no mesmo
Postgres (banco proprio `evolution`, criado por
`infra/postgres/initdb/01-setup.sql`). Ela e o backend dos **canais por filial**:
cada filial cria uma instancia nomeada com o **id da filial** e conecta por QR
Code.

- Manager/UI: `http://localhost:38081/manager` (chave `EVOLUTION_API_KEY`).
- Dentro da rede o `api` fala `http://evolution:8080` (definido no Compose). No
  `.env`, `EVOLUTION_API_URL` aponta para `http://localhost:38081` (uso no host).
- `EVOLUTION_API_KEY` precisa ser **igual** ao `AUTHENTICATION_API_KEY` da
  Evolution (o Compose usa o mesmo valor para os dois).
- Instancias ficam no volume `evolution_instances`. Para zerar tudo:
  `docker compose ... down -v` (apaga tambem os dados do Postgres).
- Fluxo na plataforma: Configuracoes -> Filiais -> editar filial -> Canais ->
  **Conectar WhatsApp** (gera o QR), ler no WhatsApp -> Aparelhos conectados.

> Se o QR nao aparecer, veja `docker logs chosen-evolution`. A Evolution precisa
> alcancar `web.whatsapp.com`; versoes antigas da imagem deixam de conectar
> quando o WhatsApp muda (por isso usamos `evoapicloud/evolution-api:latest`).

## Reset operacional (reiniciar testes)

Para zerar os dados de teste mantendo a base (tenant, filiais, papeis,
permissoes) e apenas o usuario **super_admin**, use a funcao
`reset_operational_data()` (migracao `000058`). Ela trunca tudo que e operacional
(membros, financeiro, eventos, kids, governanca, comunicados, auditoria...) e
remove vinculos/identidades que nao sejam de super_admin.

- Via API (super_admin): `POST /api/v1/admin/reset-data` com corpo
  `{"confirm":"RESET"}`.
- Via SQL:
  `docker exec chosen-postgres psql -U postgres -d chosenerp -c "SELECT reset_operational_data();"`.

> E destrutivo e nao tem volta. Instancias da Evolution (`evolution_instances`)
> e a config de canais das filiais NAO sao apagadas (ficam em `branches`).

## Fases (roadmap)

Fase 0 (fundacao tecnica) implementada. Proximo: Fase 1 (MVP) - secretaria/comunicacao
e financeiro basico. Ver `Docs/Chosen_ERP_Documentacao_Completa.md` e
`Docs/01_Blueprint_Arquitetura.md`.
