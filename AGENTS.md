# Chosen ERP — Guia de Desenvolvimento

SaaS eclesiástico multi-tenant. Backend em **Go** (monolito modular), banco
**PostgreSQL 17** com Row-Level Security, frontend **Next.js** e infra em **Docker Compose**.

Este arquivo é a fonte das instruções de build/execução para agentes e devs.

---

## Estrutura

```
apps/webadmin/        Painel administrativo (Next.js 15, App Router, Tailwind v4) + Dockerfile
cmd/api/              Entrypoint do serviço Go (monolito modular)
internal/             Domínios: auth, store (pool+migrações+RLS), httpapi, members
db/migrations/        Migrações SQL versionadas (embutidas no binário via Go embed)
db/init/setup.sql     Criação do papel de aplicação (ref. infra/postgres/initdb)
infra/                docker-compose.yml, Dockerfile da API, nginx, Prometheus/Grafana
Docs/                 PRD, blueprint e checkpoints
```

## Como rodar

### Início rápido (arquivo único)

```powershell
.\start.ps1            # tudo: .env, build do Go e Docker (postgres + api + webadmin + nginx)
.\start.ps1 -Full      # + redis, rabbitmq, prometheus e grafana
```

### 1. Infra + API + Webadmin em Docker

```bash
# sobe os containers (postgres + api + webadmin + nginx). Stack completa opcional:
#   docker compose --profile full up -d
docker compose -f infra/docker-compose.yml up -d

# a API aplica as migrações automaticamente ao iniciar (papel migrador)
# o webadmin (Next.js standalone) é buildado na primeira subida
```

Copie `.env.example` para `.env` e ajuste credenciais se necessário.

### 2. Webadmin (dev local, opcional)

Em produção o webadmin roda no Compose (`chosen-webadmin`, porta 33000). Para
desenvolver o frontend com hot-reload no host:

```bash
cd apps/webadmin
npm install
npm run dev        # http://localhost:33000
```

### Webadmin — boas práticas de frontend

- **Design system** em `apps/webadmin/components/ui/*` (Button, Card, Badge, Table,
  Modal/Drawer, Tabs, Toast, Skeleton, EmptyState, Pagination, Avatar, StatCard,
  PageHeader). Reutilize em vez de criar classes avulsas.
- **Sessão/RBAC:** usar `useAuth()` (`components/providers/auth-provider.tsx`).
  O cliente `lib/api.ts` faz login, guarda tokens e **auto-refresca em 401**.
  Nunca passar token como argumento — as funções já usam a sessão.
- Menu lateral é **filtrado por permissão** (`hasPerm`); nav definida no `layout.tsx`.
- Tema dark (toggle em `components/theme-toggle.tsx`). Helpers em `lib/format.ts`
  (`currency`, `datePt`, `relativePt`) e `lib/constants.ts` (rótulos de status).
- **Dependências:** `tailwind-merge`+`clsx` para classes e `recharts` para gráficos.

### 3. Serviço Go localmente (para depurar)

```bash
# build do binário
go build ./...
go run ./cmd/api
```

### 4. Testes

Os testes de RLS precisam de um PostgreSQL real (Docker de pé). Eles **não tocam**
no banco de desenvolvimento: criam e recriam um banco descartável
`chosenerp_test`.

```powershell
# usa os mesmos DSNs do .env (só para derivar o banco de teste)
$env:CHOSEN_TEST_MIGRATE_URL="postgres://postgres:sinc@127.0.0.1:35432/chosenerp?sslmode=disable"
$env:CHOSEN_TEST_APP_URL="postgres://chosenerp_app:chosenapp@127.0.0.1:35432/chosenerp?sslmode=disable"

go test ./... -count=1
```

Variáveis:

| Var | Fallback | Papel |
|---|---|---|
| `CHOSEN_TEST_MIGRATE_URL` | `MIGRATE_DATABASE_URL` | dono/superuser (cria o banco e aplica migrações) |
| `CHOSEN_TEST_APP_URL` | `DATABASE_URL` | `chosenerp_app`, sujeito a RLS |
| `CHOSEN_TESTS_REQUIRED=1` | – | **falha** em vez de pular, quando os DSNs não existem (use no CI) |

Se os DSNs não estiverem definidos, a suíte é pulada com aviso.

Antes de commitar: `go build ./... && go vet ./... && go test ./... -count=1`.

---

## Credenciais de desenvolvimento (seed)

- Admin (Sede): `admin@demo.local`  (papel `super_admin`)
- Filial Norte: `pastor.norte@demo.local`

As senhas **não são versionadas**: ficam no `.env` (gitignorado), em
`DEMO_ADMIN_PASSWORD` e `DEMO_NORTE_PASSWORD`. Leia com `grep '^DEMO_' .env`.

> ⚠️ `pastor.norte@demo.local` (e a filial Norte) **não é criado por
> nenhuma migração** — o seed (000009) cria apenas o tenant `demo`, a filial
> "Sede Matriz" e o `admin@demo.local`. Ver `Docs/02_Backlog.md` (#40).
> A conta existe no banco de desenvolvimento atual porque foi criada à mão.

---

## Segurança multi-tenant (como funciona)

- Duas roles de banco:
  - **migrador** (`postgres`): dono do schema, executa as migrações `db/migrations`.
  - **app** (`chosenerp_app`): conexão da API. **Não é dono nem superuser** ⇒ o
    PostgreSQL aplica Row-Level Security em todas as queries.
- Cada requisição roda dentro de uma transação com
  `set_config('app.tenant_id'/'app.branch_id'/'app.role'/'app.branch_scope', ...)`.
  As políticas RLS (`000007_rls_policies.up.sql`, revisadas pela migração
  `000016_tenant_scope_rls.up.sql`) filtram por **tenant e** `branch_id`; escopo
  "Sede" (branch NULL) enxerga **todo o seu tenant**, e escopo `system` (workers)
  enxerga tudo.
- **Hierarquia de filiais (sub-congregações, `000036`):** o gateway grava em
  `app.branch_scope` a filial do contexto + descendentes. A **leitura** inclui os
  descendentes (`rls_read_scope`); a **escrita** continua no branch exato. A
  migração `000037` corrigiu as políticas `*_sel`, que eram `FOR ALL` e davam
  escrita pela leitura — hoje são `FOR SELECT`.
- **Escrita pela Sede (`000045`):** `rls_write` passou a incluir `is_headquarters()`,
  então a Sede (branch NULL + `super_admin`/`admin_sede`) mantém registros de
  qualquer filial do próprio tenant. Antes, UPDATE/DELETE da Sede afetavam 0
  linhas e viravam 404 (ex.: editar evento, estornar lançamento). Quem tem filial
  continua gravando só no branch exato.
- **Troca de contexto filial/Sede:** o header `X-Branch-Id` permite que
  `super_admin`/`admin_sede` escolham a filial de trabalho por requisição. O
  middleware (`internal/httpapi/middleware.go`) sobrescreve `claims.BranchID`:
  ausente = escopo do token, `all` = Sede (todas as filiais), uuid = opera como
  aquela filial (leitura e gravação). Ignorado para os demais papéis. No webadmin
  o seletor fica na topbar; a escolha vai em `localStorage` e acompanha todo
  request via `lib/api.ts`.
- **Identidade global + multi-igreja (`000053`):** `users` guarda só a
  identidade (e-mail UNIQUE global, senha, MFA). O vínculo pessoa ↔ igreja vive
  em `memberships` (`user_id`, `tenant_id`, `role_id`, `branch_id`,
  `is_active`; UNIQUE `user_id,tenant_id`). O login (`auth_lookup_user`) resolve
  a identidade e `auth_memberships(user_id)` lista as igrejas: com 1 vínculo
  entra direto; com >1 devolve `requires_tenant_selection` +
  `selection_token` (`typ=select`, 5 min) + lista, e o front chama
  `POST /auth/select-tenant`. Já autenticado, `POST /auth/switch-tenant` troca a
  igreja. O JWT continua carregando o tenant ATIVO (`tid/bid/role`); o RLS de
  `users` (`id = current_user_id()` ou membership no tenant) e de `memberships`
  usa o GUC `app.user_id`, setado por `store.WithTenant`. O middleware aceita
  `X-Tenant-Id`/`X-Tenant-Slug` (validando vínculo ativo) além do `X-Branch-Id`.
- **Subdomínio / white-label (`000054`):** `tenants` ganhou
  `logo_url/brand_color/favicon_url/custom_domain`; `public_tenant(slug)`
  (SECURITY DEFINER) alimenta a tela de login do subdomínio. O nginx
  (`infra/nginx/conf.d/default.conf`) tem `server_name` wildcard
  (`*.chosenerp.mgmconsultoria.com`) e repassa `X-Tenant-Slug`; a Cloudflare
  precisa do DNS wildcard + TLS `*.dominio`. No webadmin, o slug é detectado via
  `tenantSlugFromHost()` (`lib/api.ts`) e há seletor de igreja na topbar quando a
  identidade tem mais de um vínculo.
- O isolamento é verificado por **testes automatizados** em
  `internal/store/rls_test.go` e `internal/store/memberships_test.go` (varredura
  de todas as tabelas, além de casos dedicados de identidade/membership).
- `financial_transactions` e `audit_log` são **append-only** (triggers impedem
  UPDATE/DELETE) com **hash-chain** de integridade.
- Login usa função `SECURITY DEFINER` (`auth_lookup_user`) para localizar o usuário
  **fora** do escopo RLS (o tenant ainda não é conhecido).

### Teste manual de isolamento

```
docker exec chosen-postgres psql -U postgres -d chosenerp \
  -c "SELECT set_config('app.branch_id','<branchA>',true); SELECT * FROM members;"
```

---

## Endpoints da API (base `http://localhost:38080`)

| Método | Rota | Auth | Descrição |
|---|---|---|---|
| GET  | `/healthz` | – | Health check |
| GET  | `/metrics` | – | Métricas Prometheus |
| POST | `/api/v1/auth/login` | – | Login (access + refresh); `tenant_slug` opcional; pode exigir seleção de igreja |
| POST | `/api/v1/auth/select-tenant` | selection token | Conclui o login escolhendo a igreja |
| POST | `/api/v1/auth/switch-tenant` | Bearer | Troca a igreja ativa (novos tokens) |
| GET  | `/api/v1/auth/refresh` | refresh | Renova tokens |
| GET  | `/api/v1/me` | Bearer | Perfil + contexto + memberships |
| GET  | `/api/v1/me/tenants` | Bearer | Igrejas da identidade (seletor) |
| GET  | `/api/v1/public/tenant/{slug}` | – | Branding público da igreja (login do subdomínio) |
| PATCH | `/api/v1/me` | Bearer | Edita o próprio perfil (nome/e-mail) |
| POST | `/api/v1/me/password` | Bearer | Troca a própria senha (senha atual + nova) |
| GET  | `/api/v1/members` | Bearer | Lista membros (escopo RLS) |
| POST | `/api/v1/members` | Bearer | Cria membro (escopo RLS) |
| GET  | `/api/v1/members/{id}` | Bearer | Detalhe de membro |
| GET  | `/api/v1/members/{id}/tree` | Bearer | Árvore genealógica + discipulado |
| PATCH | `/api/v1/members/{id}` | Bearer | Edita perfil do membro |
| POST | `/api/v1/members/{id}/relationships` | Bearer | Cria vínculo (cônjuge/filho/discípulo...) |
| POST | `/api/v1/members/{id}/photo` | Bearer | Envia foto (multipart, `UPLOAD_DIR`, disco local) |
| DELETE | `/api/v1/members/{id}/photo` | Bearer | Remove a foto do membro |
| POST | `/api/v1/members/{id}/card` | Bearer | Emite carteirinha QR — **idempotente** (devolve `card_ref` + `token`) |
| GET  | `/api/v1/members/{id}/card` | Bearer | Lê a carteirinha já emitida (`card_ref` + `token`) |
| GET  | `/api/v1/members/{id}/families` | Bearer | Famílias das quais o membro participa |
| GET  | `/api/v1/cargos` | Bearer | Catálogo de cargos do tenant |
| POST | `/api/v1/cargos` | Bearer | Cria cargo (`name`, `kind`, `requires_term`, `sort_order`) |
| PATCH | `/api/v1/cargos/{id}` | Bearer | Edita/ativa/desativa cargo |
| DELETE | `/api/v1/cargos/{id}` | Bearer | Exclui cargo (**409** se houver mandatos — desative) |
| GET  | `/api/v1/members/{id}/cargos` | Bearer | Mandatos do membro (histórico completo) |
| POST | `/api/v1/members/{id}/cargos` | Bearer | Atribui cargo (`started_at`, `ends_at`, `status`) |
| PATCH | `/api/v1/members/{id}/cargos/{linkId}` | Bearer | Edita mandato; `ends_at: ""` **limpa** o vencimento |
| DELETE | `/api/v1/members/{id}/cargos/{linkId}` | Bearer | Remove o mandato do histórico |
| GET  | `/api/v1/members/{id}/history` | Bearer | Histórico eclesiástico (append-only) |
| POST | `/api/v1/members/{id}/history` | Bearer | Registra evento no histórico |
| GET  | `/api/v1/members/{id}/frequency` | Bearer | Histórico de frequência |
| POST | `/api/v1/members/{id}/frequency` | Bearer | Atualiza a frequência (mantém histórico) |
| GET  | `/api/v1/event-kinds` | Bearer | Tipos de evento |
| POST | `/api/v1/event-kinds` | Bearer | Cria tipo de evento |
| PATCH | `/api/v1/event-kinds/{id}` | Bearer | Edita tipo de evento |
| DELETE | `/api/v1/event-kinds/{id}` | Bearer | Exclui tipo de evento (409 se em uso) |
| GET  | `/api/v1/events?from=&to=&kind=` | Bearer | Lista eventos |
| POST | `/api/v1/events` | Bearer | Cria evento (data/hora, tipo, total) |
| GET  | `/api/v1/events/{id}` | Bearer | Detalhe do evento |
| PATCH | `/api/v1/events/{id}` | Bearer | Edita evento |
| DELETE | `/api/v1/events/{id}` | Bearer | Exclui evento |
| GET  | `/api/v1/events/{id}/attendance` | Bearer | Chamada nominal do evento |
| POST | `/api/v1/events/{id}/attendance` | Bearer | Salva chamada nominal + total |
| GET  | `/api/v1/events/{id}/invitees` | Bearer | Convocados (pessoas/ministerios) |
| POST | `/api/v1/events/{id}/invitees` | Bearer | Define os convocados do evento |
| GET  | `/api/v1/consent-terms` | Bearer | Termos de consentimento (LGPD) |
| POST | `/api/v1/consent-terms` | Bearer (Sede) | Cria termo de consentimento |
| GET  | `/api/v1/members/{id}/consents` | Bearer | Consentimentos do membro |
| POST | `/api/v1/members/{id}/consents` | Bearer | Registra/revoga consentimento |
| GET  | `/api/v1/members/{id}/export` | Bearer | Exporta os dados do titular (JSON) |
| POST | `/api/v1/members/{id}/anonymize` | Bearer (Sede) | Anonimiza os dados pessoais |
| GET/POST | `/api/v1/minutes` | Bearer | Lista/cria atas (livro digital) |
| GET/PATCH/DELETE | `/api/v1/minutes/{id}` | Bearer | Lê/edita/exclui ata (bloqueia se assinada) |
| POST | `/api/v1/minutes/{id}/sign` | Bearer | Assinatura interna (hash + credenciais) |
| GET | `/api/v1/minutes/{id}/signatures` | Bearer | Assinaturas da ata (hash-chain) |
| GET/POST | `/api/v1/votes` | Bearer | Lista/cria votações (quórum + voto secreto) |
| GET/PATCH/DELETE | `/api/v1/votes/{id}` | Bearer | Lê/edita/exclui votação |
| POST | `/api/v1/votes/{id}/open` | Bearer | Abre a votação |
| POST | `/api/v1/votes/{id}/ballot` | Bearer | Vota em segredo (sem vínculo eleitor↔voto) |
| POST | `/api/v1/votes/{id}/close` | Bearer | Encerra, apura e anexa à ata vinculada |
| GET | `/api/v1/votes/{id}/result` | Bearer | Apuração (contagens + quórum) |
| GET/POST | `/api/v1/legal-documents` | Bearer | Convênios/documentos legais com vencimento |
| PATCH/DELETE | `/api/v1/legal-documents/{id}` | Bearer | Edita/exclui documento legal |
| GET | `/api/v1/governance/mandates` | Bearer | Painel de mandatos (vigentes/vencendo) |
| GET  | `/api/v1/families` | Bearer | Lista famílias |
| POST | `/api/v1/families` | Bearer | Cria família (código `#NNN` gerado na transação) |
| GET  | `/api/v1/families/{id}` | Bearer | Detalhe da família (chefe + endereço jsonb) |
| PATCH | `/api/v1/families/{id}` | Bearer | Renomeia / troca chefe (`head_id`) / grava `address` |
| DELETE | `/api/v1/families/{id}` | Bearer | Exclui família (vínculos de parentesco sobrevivem) |
| GET/POST | `/api/v1/families/{id}/members` | Bearer | Lista/vincula membros da família |
| DELETE | `/api/v1/families/{id}/members/{memberId}` | Bearer | Desvincula (limpa `family_id` e o chefe, se for o caso) |
| GET  | `/api/v1/visitors` | Bearer | Lista visitantes |
| POST | `/api/v1/visitors` | Bearer | Registra visitante |
| PATCH | `/api/v1/visitors/{id}/stage` | Bearer | Avança trilha de acolhimento |
| GET/POST | `/api/v1/benefactors` | Bearer | Lista/cadastra benfeitores |
| GET/POST | `/api/v1/suppliers` | Bearer | Lista/cadastra fornecedores (CPF/CNPJ opcionais) |
| PATCH/DELETE | `/api/v1/suppliers/{id}` | Bearer | Edita/exclui fornecedor |
| GET  | `/api/v1/finance/categories` | Bearer | Plano de contas |
| POST | `/api/v1/finance/categories` | Bearer | Cria categoria |
| PATCH | `/api/v1/finance/categories/{id}` | Bearer | Edita/ativa/desativa categoria |
| DELETE | `/api/v1/finance/categories/{id}` | Bearer | Exclui categoria (409 se em uso — desative) |
| GET  | `/api/v1/finance/accounts` | Bearer | Contas bancárias |
| POST | `/api/v1/finance/accounts` | Bearer | Cria conta bancária |
| PATCH | `/api/v1/finance/accounts/{id}` | Bearer | Edita/ativa/desativa conta |
| DELETE | `/api/v1/finance/accounts/{id}` | Bearer | Exclui conta (409 se em uso — desative) |
| GET  | `/api/v1/finance/transactions?type=` | Bearer | Lista lançamentos (com account_id) |
| POST | `/api/v1/finance/transactions` | Bearer | Lança dízimo/oferta/despesa (+ recibo auto) |
| POST | `/api/v1/finance/transactions/import` | Bearer | Importa lançamentos (CSV ou XLSX, com mapeamento de colunas) |
| POST | `/api/v1/finance/transactions/import/preview` | Bearer | Pré-visualiza as linhas da planilha para mapear colunas |
| POST | `/api/v1/finance/transactions/{id}/void` | Bearer | Estorna o lançamento (append-only; sai dos relatórios) |
| GET  | `/api/v1/finance/transactions/{id}/events` | Bearer | Rateio do lançamento por evento |
| PATCH | `/api/v1/ministries/{id}` | Bearer | Edita ministério (responsável/situação) |
| DELETE | `/api/v1/ministries/{id}` | Bearer | Exclui ministério |
| PATCH | `/api/v1/groups/{id}` | Bearer | Edita grupo/célula |
| DELETE | `/api/v1/groups/{id}` | Bearer | Exclui grupo/célula |
| GET/POST | `/api/v1/rosters` | Bearer | Lista/cria escalas (evento ou tipo; `create_event` gera o evento na grade) |
| GET | `/api/v1/rosters/suggestions?ministry_id=&starts_at=` | Bearer | Sugere voluntários (marca conflito) |
| GET/PATCH/DELETE | `/api/v1/rosters/{id}` | Bearer | Lê/edita/exclui escala (`DELETE ?delete_event=true` remove o evento gerado) |
| POST | `/api/v1/rosters/{id}/assignments` | Bearer | Define os escalados (preserva confirmações) |
| PATCH | `/api/v1/rosters/{id}/assignments/{assignmentId}` | Bearer | Confirma/recusa presença |
| GET | `/api/v1/rosters/{id}/conflicts` | Bearer | Conflitos de agenda da escala |
| GET  | `/api/v1/finance/transactions/{id}/attachments` | Bearer | Lista anexos de um lançamento |
| POST | `/api/v1/finance/transactions/{id}/attachments` | Bearer | Anexa documento (multipart, até 50MB) |
| GET  | `/api/v1/attachments/{filename}` | – | Download de anexo (nome opaco) |
| GET  | `/api/v1/finance/balance` | Bearer | Balancete/DRE (entradas, saídas, por categoria) |
| GET  | `/api/v1/finance/audits` | Bearer | Lista auditorias financeiras |
| POST | `/api/v1/finance/audits` | Bearer | Cria auditoria (período) |
| GET  | `/api/v1/finance/audits/{id}` | Bearer | Auditoria + lançamentos do período |
| POST | `/api/v1/finance/audits/{id}/mark` | Bearer | Marca/desmarca auditado (lista vazia = todos) |
| POST | `/api/v1/finance/audits/{id}/close` | Bearer | Fecha e assina (documento imutável) |
| GET  | `/api/v1/finance/audits/{id}/export?format=` | Bearer | PDF/CSV/XLSX do documento auditado |
| DELETE | `/api/v1/finance/audits/{id}` | Bearer | Exclui a auditoria e seus itens (mesmo fechada) |
| GET  | `/api/v1/documents/by-token/{token}` | Bearer | Lê documento por token do QR |
| GET  | `/api/v1/receipts/{id}` | Bearer | Recibo em **HTML** pronto p/ impressão |
| POST | `/api/v1/receipts/{id}/send` | Bearer | Enfileira+envia recibo (`channel`: email\|whatsapp) |
| GET  | `/api/v1/receipts/{id}/deliveries` | Bearer | Histórico de envios |
| GET  | `/api/v1/reports/balance?from=&to=` | Bearer | Balancete mensal (série por mês) |
| GET  | `/api/v1/reports/dre?from=&to=` | Bearer | DRE por categoria + comparativo de período |
| GET  | `/api/v1/reports/birthdays?month=` | Bearer | Aniversariantes (nascimento + casamento) |
| GET  | `/api/v1/reports/demographics` | Bearer | Painel demográfico (idade/status/UF/cidade) |
| GET  | `/api/v1/reports/balance/export?format=csv\|xlsx\|pdf` | Bearer | Exporta o balancete |
| GET  | `/api/v1/reports/dre/export?format=csv\|xlsx\|pdf` | Bearer | Exporta o DRE |
| GET  | `/api/v1/reports/birthdays/export?month=&format=` | Bearer | Exporta aniversariantes |
| GET  | `/api/v1/reports/demographics/export?format=` | Bearer | Exporta demográficos |
| GET  | `/api/v1/reports/monthly-statement?month=` | Bearer | Demonstrativo Mensal (regime de caixa) |
| GET  | `/api/v1/reports/monthly-statement/export?month=&format=csv\|xlsx\|pdf` | Bearer | Exporta o Demonstrativo Mensal |
| GET  | `/api/v1/reports/consolidated?from=&to=` | Bearer | Painel consolidado por filial (respeita a hierarquia) |
| GET  | `/api/v1/reports/assembly?from=&to=` | Bearer | Demonstrativo para assembleia (JSON) |
| GET  | `/api/v1/reports/assembly/export?from=&to=&format=` | Bearer | Exporta o Demonstrativo para assembleia |
| GET  | `/api/v1/users` | Bearer (Sede) | Lista usuários |
| POST | `/api/v1/users` | Bearer (Sede) | Cria usuário |
| PATCH | `/api/v1/users/{id}` | Bearer (Sede) | Edita usuário (perfil/filial/ativo) |
| POST | `/api/v1/users/{id}/password` | Bearer (Sede) | Redefine a senha |
| GET  | `/api/v1/roles` | Bearer (Sede) | Perfis + permissões |
| GET  | `/api/v1/permissions` | Bearer (Sede) | Catálogo de permissões |
| GET  | `/api/v1/auth/mfa` | Bearer | Estado do MFA |
| POST | `/api/v1/auth/mfa/setup` | Bearer | Gera segredo TOTP |
| POST | `/api/v1/auth/mfa/enable` | Bearer | Ativa MFA (código) |
| POST | `/api/v1/auth/mfa/disable` | Bearer | Desativa MFA |
| GET  | `/api/v1/branches` | Bearer | Lista filiais/congregações do tenant |
| POST | `/api/v1/branches` | Bearer (Sede) | Cria filial/congregação (`cnpj` opcional) |
| PATCH | `/api/v1/branches/{id}` | Bearer (Sede) | Edita filial (nome/tipo/CNPJ/endereço/ativa) |
| DELETE | `/api/v1/branches/{id}` | Bearer (Sede) | Exclui filial (409 se houver membros) |
| GET  | `/api/v1/tenant` | Bearer | Dados da igreja (tenant) |
| PATCH | `/api/v1/tenant` | Bearer (Sede) | Edita dados da igreja |

---

## Banco / migrações

- Migrações versionadas em `db/migrations/*.up.sql` (e `.down.sql`).
- O binário as embute e aplica no boot, rastreando versões em `schema_migrations`.
- Convenção: `0000xx_nome_curto.up.sql`.

## Build da imagem da API

O build usa `FROM scratch` (sem pull de imagens) — compile o binário Linux antes:

```bash
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -o bin/api-linux-amd64 ./cmd/api
docker build -f infra/api/Dockerfile -t chosenerp/api:local .
```

O container roda como UID `65532` (não-root) e o Dockerfile copia
`infra/api/uploads/` para `/uploads` com `--chown=65532:65532`. Isso é o que faz o
volume nomeado de uploads **nascer com o dono certo** e a API conseguir gravar fotos
de membro e anexos do financeiro — não remova nem o diretório nem o `--chown`.

---

## Observabilidade (opcional)

Com o profile completo (`--profile full`), Prometheus (porta 39090) e Grafana (porta 33001)
coletam métricas do serviço `api`. O Prometheus aponta para `api:8080/metrics`.

## Fases (roadmap)

Fase 0 (fundação técnica) implementada. Próximo: Fase 1 (MVP) — secretaria/comunicação
e financeiro básico. Ver `Docs/Chosen_ERP_Documentacao_Completa.md` e
`Docs/01_Blueprint_Arquitetura.md`.
