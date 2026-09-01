# Chosen ERP — Guia de Desenvolvimento

SaaS eclesiástico multi-tenant. Backend em **Go** (monolito modular), banco
**PostgreSQL 17** com Row-Level Security, frontend **Next.js** e infra em **Docker Compose**.

Este arquivo é a fonte das instruções de build/execução para agentes e devs.

---

## Estrutura

```
apps/webadmin/        Painel administrativo (Next.js 15, App Router, Tailwind v4)
cmd/api/              Entrypoint do serviço Go (monolito modular)
internal/             Domínios: auth, store (pool+migrações+RLS), httpapi, members
db/migrations/        Migrações SQL versionadas (embutidas no binário via Go embed)
db/init/setup.sql     Criação do papel de aplicação (ref. infra/postgres/initdb)
infra/                docker-compose.yml, Dockerfile da API, Prometheus/Grafana
Docs/                 PRD, blueprint e checkpoints
```

## Como rodar

### Início rápido (arquivo único)

```powershell
.\start.ps1            # tudo: .env, build do Go, Docker (postgres+api) e webadmin
.\start.ps1 -Full      # + redis, rabbitmq, prometheus e grafana
.\start.ps1 -NoWeb     # só infraestrutura e API
```

### 1. Infra + API em Docker

```bash
# sobe os containers (postgres + api). Stack completa opcional:
#   docker compose --profile full up -d
docker compose -f infra/docker-compose.yml up -d

# a API aplica as migrações automaticamente ao iniciar (papel migrador)
```

Copie `.env.example` para `.env` e ajuste credenciais se necessário.

### 2. Webadmin (Next.js)

```bash
cd apps/webadmin
npm install
npm run dev        # http://localhost:3000
```

### 3. Serviço Go localmente (para depurar)

```bash
# build do binário
go build ./...
go run ./cmd/api
```

---

## Credenciais de desenvolvimento (seed)

- Admin (Sede): `admin@demo.local` / `admin123`  (papel `super_admin`)
- Pastor Filial (Norte): `pastor.norte@demo.local` / `norte123` (papel `pastor_filial`)

---

## Segurança multi-tenant (como funciona)

- Duas roles de banco:
  - **migrador** (`postgres`): dono do schema, executa as migrações `db/migrations`.
  - **app** (`chosenerp_app`): conexão da API. **Não é dono nem superuser** ⇒ o
    PostgreSQL aplica Row-Level Security em todas as queries.
- Cada requisição roda dentro de uma transação com
  `set_config('app.tenant_id'/'app.branch_id'/'app.role', ...)`. As políticas RLS
  (`000007_rls_policies.up.sql`) filtram por `branch_id`; escopo "Sede" (branch NULL)
  enxerga tudo.
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
| POST | `/api/v1/auth/login` | – | Login (access + refresh) |
| POST | `/api/v1/auth/refresh` | refresh | Renova tokens |
| GET  | `/api/v1/me` | Bearer | Perfil + contexto |
| GET  | `/api/v1/members` | Bearer | Lista membros (escopo RLS) |
| POST | `/api/v1/members` | Bearer | Cria membro (escopo RLS) |
| GET  | `/api/v1/members/{id}` | Bearer | Detalhe de membro |
| GET  | `/api/v1/members/{id}/tree` | Bearer | Árvore genealógica + discipulado |
| PATCH | `/api/v1/members/{id}` | Bearer | Edita perfil do membro |
| POST | `/api/v1/members/{id}/relationships` | Bearer | Cria vínculo (cônjuge/filho/discípulo...) |
| POST | `/api/v1/members/{id}/card` | Bearer | Emite carteirinha QR |
| GET  | `/api/v1/families` | Bearer | Lista famílias |
| POST | `/api/v1/families` | Bearer | Cria família |
| GET/POST | `/api/v1/families/{id}/members` | Bearer | Lista/vincula membros da família |
| GET  | `/api/v1/visitors` | Bearer | Lista visitantes |
| POST | `/api/v1/visitors` | Bearer | Registra visitante |
| PATCH | `/api/v1/visitors/{id}/stage` | Bearer | Avança trilha de acolhimento |
| GET/POST | `/api/v1/benefactors` | Bearer | Lista/cadastra benfeitores |
| GET  | `/api/v1/finance/categories` | Bearer | Plano de contas |
| POST | `/api/v1/finance/categories` | Bearer | Cria categoria |
| GET  | `/api/v1/finance/transactions?type=` | Bearer | Lista lançamentos |
| POST | `/api/v1/finance/transactions` | Bearer | Lança dízimo/oferta/despesa (+ recibo auto) |
| GET  | `/api/v1/finance/balance` | Bearer | Balancete/DRE (entradas, saídas, por categoria) |
| GET  | `/api/v1/documents/by-token/{token}` | Bearer | Lê documento por token do QR |
| GET  | `/api/v1/receipts/{id}` | Bearer | Recibo em **HTML** pronto p/ impressão |
| POST | `/api/v1/receipts/{id}/send` | Bearer | Enfileira+envia recibo (`channel`: email\|whatsapp) |
| GET  | `/api/v1/receipts/{id}/deliveries` | Bearer | Histórico de envios |
| GET  | `/api/v1/reports/balance?from=&to=` | Bearer | Balancete mensal (série por mês) |
| GET  | `/api/v1/reports/dre?from=&to=` | Bearer | DRE por categoria + comparativo de período |

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

---

## Observabilidade (opcional)

Com o profile completo (`--profile full`), Prometheus (porta 39090) e Grafana (porta 33001)
coletam métricas do serviço `api`. O Prometheus aponta para `api:8080/metrics`.

## Fases (roadmap)

Fase 0 (fundação técnica) implementada. Próximo: Fase 1 (MVP) — secretaria/comunicação
e financeiro básico. Ver `Docs/Chosen_ERP_Documentacao_Completa.md` e
`Docs/01_Blueprint_Arquitetura.md`.
