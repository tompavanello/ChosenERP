# Chosen ERP - Blueprint de Arquitetura e Execucao

**Versao:** 1.0
**Data:** Setembro/2026
**Base:** `Chosen_ERP_Documentacao_Completa.md` (PRD)
**Status:** Aprovado - base para a Fase 0/1

---

## 1. Decisoes-chave (tradeoffs resolvidos)

| Tema | Decisao | Justificativa |
|---|---|---|
| Microservicos vs. monolito | **Monolito modular em Go** (dominios bem separados) + **1 servico Python** para tarefas assincronas/IA | Parte do padrao da doc, mas na Fase 0/1 microservicos de verdade custam caro e atrasam o MVP. Os limites de modulo ficam prontos para extrair depois. |
| Banco | PostgreSQL **17** + RLS + JSONB | Doc pede 16+; 17 e superior e ja instalado. |
| Auth | **Go**: JWT (access+refresh) + RBAC + MFA (TOTP) | Servico critico em Go, como na doc. |
| Mensageria | **RabbitMQ** (Redis so p/ cache/sessao) | Fiel a doc para jobs longos (relatorios, repasses, notificacao). |
| Frontend | **Next.js 15 (App Router) + Tailwind + shadcn/ui** | Painel admin web. Design system proprio. |
| Multi-tenant | Banco compartilhado + **RLS por `branch_id`** | Como especificado na secao 5 da doc. |
| Porta de entrada | **API Gateway** (Go) -> roteia para servicos; **gRPC** interno | Contrato unico + OpenAPI publico futuro. |

## 2. Stack concreta (compativel com o ambiente local)

| Camada | Tecnologia | Versao local |
|---|---|---|
| Banco | PostgreSQL | 17.8 (servico parado - iniciar) |
| Backend critico | Go | 1.26.5 |
| Backend de dados | Python (FastAPI + Celery) | 3.14 (`py`) |
| Frontend | Next.js (React) + Tailwind + shadcn/ui | Node 24.15.0 / npm 11.12.1 |
| Cache | Redis | via docker-compose |
| Mensageria | RabbitMQ | via docker-compose |
| Versionamento | Git | 2.54.0 |

## 3. Layout do repositorio (monorepo)

```
ChosenERP/
 Docs/                       (PRD + blueprint)
 apps/
   webadmin/                Next.js - painel admin (Sede/Filial/Funcoes)
   superapp/                (fase futura - app membro, Expo RN)
 services/
   gateway/                 Go - API Gateway + WebSocket
   auth/                    Go - Auth, RBAC, MFA, tenant resolution
   ledger/                  Go - financeiro, repasses, folha, auditoria
   governance/              Go - atas, votos, mandatos
   members/                 Go - cadastros, membros, celulas, discipulado
   workers/                 Python/FastAPI+Celery - relatorios, PDF, IA, integracao
 packages/                   libs compartilhadas (Go modules / npm)
 db/
   migrations/              SQL + golang-migrate
 infra/                      docker-compose, k8s manifests, terraform
 .github/workflows/          CI/CD
 opencode.json
```

## 4. Modelo de dados (convencoes + tabelas-nucleo da Fase 0/1)

Convencoes globais:
- Todo schema tem `tenant_id` e/ou `branch_id` (se aplicavel) -> polpa da RLS.
- `financial_transactions` e `audit_log` -> **append-only**: sem UPDATE/DELETE (triggers `BEFORE UPDATE/DELETE RAISE EXCEPTION`), coluna `prev_hash` encadeando (hash-chain).
- `pastoral_records` -> colunas de conteudo em `bytea` cifradas (chave do pastor fora do cluster); indice por `member_id` + `author`, nao por conteudo.
- Formularios dinamicos por denominacao -> `extra_json JSONB` + `field_definitions` (motor de campos dinamicos do MVP).

Tabelas da Fase 0/1 em ordem de criacao:
```
tenants -> branches -> users -> roles/perms -> members -> families -> member_relationships
-> visitors -> benefactors -> ministries -> ministry_members -> small_groups -> group_attendance
-> financial_categories -> financial_transactions(hash-chain) -> transfers -> documents
-> audit_log(hash-chain)
```
Governanca/patrimonio/payroll/pastoral -> Fase 3.

### Padrao RLS (uma politica por tabela)
```sql
ALTER TABLE members ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_iso_members ON members
  USING (branch_id = current_setting('app.branch_id')::uuid);
```
- Sessao de API do gateway seta `SELECT set_config('app.branch_id', $1, true)` e `app.role`.
- Sede (`role='headquarters'` / `branch_id` nulo) ignora o filtro -> visao consolidada.
- **Teste automatizado de RLS ja no CI da Fase 0** (usuario Filial A nunca le Filial B).

## 5. Seguranca
- **RBAC** granular com permissoes por modulo (tabela `permissions` + `role_permissions`), no servico Go.
- **E2EE** para prontuario pastoral; **MFA (TOTP)** para admins/tesoureiro.
- **LGPD**: `consent_terms` + assinatura digital, job de portabilidade (export do titular), job de anonimizacao/exclusao.
- **Trilha de auditoria**: `audit_log` hash-encadeado gravado via middleware do gateway (actor, action, antes/depois, hash).

## 6. Frontend - Next.js admin
- **Design system:** Tailwind + shadcn/ui (acessivel, dark mode), tema "chosen" com tokens de cor.
- **Layout:** sidebar por modulo, breadcrumbs, dashboard com widgets.
- **Fluxos prioritarios (MVP):** Cadastro de membro (perfil 360o), Familias/arvore, Visitante+trilha, Lancamento de dizimo/oferta/despesa, Plano de contas, Recibo digital, Carteirinha QR, Relatorios (balancete/DRE).
- **Estado/Data:** TanStack Query + server actions; zod p/ validacao.

## 7. Roadmap executavel (tarefas -> entregaveis)

**Fase 0 - Fundacao (semanas 1-6)**
1. Infra: docker-compose (postgres, redis, rabbitmq), ambientes, CI.
2. Migracoes + RLS + **teste de isolamento** (parte do aceite da doc).
3. `auth` (Go): login JWT, refresh, RBAC, MFA, resolucao tenant/branch.
4. `gateway` (Go): roteamento, middleware (set_config RLS, trafego p/ auditoria).
5. Skeleton: `members` (CRUD minimo) + `webadmin` (Next.js) autenticado.
6. Observabilidade: OpenTelemetry + Prometheus/Grafana.

**Fase 1 - MVP (semanas 7-16):** membros, familias, visitantes, benfeitores, financeiro manual (dizimos/ofertas/despesas/plano de contas), recibos digitais, carteirinha QR, relatorios basicos + app-membro v1. -> *Um unico servico Go modular + 1 worker Python auxiliar.*

**Fases 2-4:** multiplicar conforme doc (multi-filial/repasses, ministerios, governanca, IA, IoT, ecossistema).

## 8. Criterios de aceite (resumo da doc)
- **Fase 0:** testes provam que Filial A nunca le dados da Filial B via API, mesmo manipulando requisicoes.
- **Fase 1:** zero dependencia de planilha externa na operacao diaria da igreja piloto.
- **Fase 2:** repasse auditado bate 100% com conferencia manual em 3 ciclos.
- **Fase 3:** ata gerada aceita como documento oficial (cartorio/registro).
- **Fase 4:** modelo preditivo de evasao supera baseline ingenuo em precisao.
