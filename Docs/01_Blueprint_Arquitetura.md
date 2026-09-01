# Chosen ERP — Blueprint de Arquitetura e Execução

**Versão:** 1.0
**Data:** Setembro/2026
**Base:** `Chosen_ERP_Documentacao_Completa.md` (PRD)
**Status:** Aprovado — base para a Fase 0/1

---

## 1. Decisões-chave (tradeoffs resolvidos)

| Tema | Decisão | Justificativa |
|---|---|---|
| Microserviços vs. monolito | **Monolito modular em Go** (domínios bem separados) + **1 serviço Python** para tarefas assíncronas/IA | Parte do padrão da doc, mas na Fase 0/1 microserviços de verdade custam caro e atrasam o MVP. Os limites de módulo ficam prontos para extrair depois. |
| Banco | PostgreSQL **17** + RLS + JSONB | Doc pede 16+; 17 é superior e já instalado. |
| Auth | **Go**: JWT (access+refresh) + RBAC + MFA (TOTP) | Serviço crítico em Go, como na doc. |
| Mensageria | **RabbitMQ** (Redis só p/ cache/sessão) | Fiel à doc para jobs longos (relatórios, repasses, notificação). |
| Frontend | **Next.js 15 (App Router) + Tailwind + shadcn/ui** | Painel admin web. Design system próprio. |
| Multi-tenant | Banco compartilhado + **RLS por `branch_id`** | Como especificado na seção 5 da doc. |
| Porta de entrada | **API Gateway** (Go) → roteia para serviços; **gRPC** interno | Contrato único + OpenAPI público futuro. |

## 2. Stack concreta (compatível com o ambiente local)

| Camada | Tecnologia | Versão local |
|---|---|---|
| Banco | PostgreSQL | 17.8 (serviço parado — iniciar) |
| Backend crítico | Go | 1.26.5 |
| Backend de dados | Python (FastAPI + Celery) | 3.14 (`py`) |
| Frontend | Next.js (React) + Tailwind + shadcn/ui | Node 24.15.0 / npm 11.12.1 |
| Cache | Redis | via docker-compose |
| Mensageria | RabbitMQ | via docker-compose |
| Versionamento | Git | 2.54.0 |

## 3. Layout do repositório (monorepo)

```
ChosenERP/
├─ Docs/                       (PRD + blueprint)
├─ apps/
│  ├─ webadmin/                Next.js — painel admin (Sede/Filial/Funções)
│  └─ superapp/                (fase futura — app membro, Expo RN)
├─ services/
│  ├─ gateway/                 Go — API Gateway + WebSocket
│  ├─ auth/                    Go — Auth, RBAC, MFA, tenant resolution
│  ├─ ledger/                  Go — financeiro, repasses, folha, auditoria
│  ├─ governance/              Go — atas, votos, mandatos
│  ├─ members/                 Go — cadastros, membros, células, discipulado
│  └─ workers/                 Python/FastAPI+Celery — relatórios, PDF, IA, integração
├─ packages/                   libs compartilhadas (Go modules / npm)
├─ db/
│  └─ migrations/              SQL + golang-migrate
├─ infra/                      docker-compose, k8s manifests, terraform
├─ .github/workflows/          CI/CD
└─ opencode.json
```

## 4. Modelo de dados (convenções + tabelas-núcleo da Fase 0/1)

Convenções globais:
- Todo schema tem `tenant_id` e/ou `branch_id` (se aplicável) → polpa da RLS.
- `financial_transactions` e `audit_log` → **append-only**: sem UPDATE/DELETE (triggers `BEFORE UPDATE/DELETE RAISE EXCEPTION`), coluna `prev_hash` encadeando (hash-chain).
- `pastoral_records` → colunas de conteúdo em `bytea` cifradas (chave do pastor fora do cluster); índice por `member_id` + `author`, não por conteúdo.
- Formulários dinâmicos por denominação → `extra_json JSONB` + `field_definitions` (motor de campos dinâmicos do MVP).

Tabelas da Fase 0/1 em ordem de criação:
```
tenants → branches → users → roles/perms → members → families → member_relationships
→ visitors → benefactors → ministries → ministry_members → small_groups → group_attendance
→ financial_categories → financial_transactions(hash-chain) → transfers → documents
→ audit_log(hash-chain)
```
Governança/patrimônio/payroll/pastoral → Fase 3.

### Padrão RLS (uma política por tabela)
```sql
ALTER TABLE members ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso_members ON members
  USING (branch_id = current_setting('app.branch_id')::uuid);
```
- Sessão de API do gateway seta `SELECT set_config('app.branch_id', $1, true)` e `app.role`.
- Sede (`role='headquarters'` / `branch_id` nulo) ignora o filtro → visão consolidada.
- **Teste automatizado de RLS já no CI da Fase 0** (usuário Filial A nunca lê Filial B).

## 5. Segurança
- **RBAC** granular com permissões por módulo (tabela `permissions` + `role_permissions`), no serviço Go.
- **E2EE** para prontuário pastoral; **MFA (TOTP)** para admins/tesoureiro.
- **LGPD**: `consent_terms` + assinatura digital, job de portabilidade (export do titular), job de anonimização/exclusão.
- **Trilha de auditoria**: `audit_log` hash-encadeado gravado via middleware do gateway (actor, action, antes/depois, hash).

## 6. Frontend — Next.js admin
- **Design system:** Tailwind + shadcn/ui (acessível, dark mode), tema "chosen" com tokens de cor.
- **Layout:** sidebar por módulo, breadcrumbs, dashboard com widgets.
- **Fluxos prioritários (MVP):** Cadastro de membro (perfil 360º), Famílias/árvore, Visitante+trilha, Lançamento de dízimo/oferta/despesa, Plano de contas, Recibo digital, Carteirinha QR, Relatórios (balancete/DRE).
- **Estado/Data:** TanStack Query + server actions; zod p/ validação.

## 7. Roadmap executável (tarefas → entregáveis)

**Fase 0 — Fundação (semanas 1–6)**
1. Infra: docker-compose (postgres, redis, rabbitmq), ambientes, CI.
2. Migrações + RLS + **teste de isolamento** (parte do aceite da doc).
3. `auth` (Go): login JWT, refresh, RBAC, MFA, resolução tenant/branch.
4. `gateway` (Go): roteamento, middleware (set_config RLS, tráfego p/ auditoria).
5. Skeleton: `members` (CRUD mínimo) + `webadmin` (Next.js) autenticado.
6. Observabilidade: OpenTelemetry + Prometheus/Grafana.

**Fase 1 — MVP (semanas 7–16):** membros, famílias, visitantes, benfeitores, financeiro manual (dízimos/ofertas/despesas/plano de contas), recibos digitais, carteirinha QR, relatórios básicos + app-membro v1. → *Um único serviço Go modular + 1 worker Python auxiliar.*

**Fases 2–4:** multiplicar conforme doc (multi-filial/repasses, ministérios, governança, IA, IoT, ecossistema).

## 8. Critérios de aceite (resumo da doc)
- **Fase 0:** testes provam que Filial A nunca lê dados da Filial B via API, mesmo manipulando requisições.
- **Fase 1:** zero dependência de planilha externa na operação diária da igreja piloto.
- **Fase 2:** repasse auditado bate 100% com conferência manual em 3 ciclos.
- **Fase 3:** ata gerada aceita como documento oficial (cartório/registro).
- **Fase 4:** modelo preditivo de evasão supera baseline ingênuo em precisão.
