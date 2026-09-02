# Chosen ERP — Checkpoint (histórico consolidado)

**Última atualização:** Setembro/2026
**Responsável:** tompavanello (proprietário) + opencode

> Este é o **único** checkpoint do projeto. Ele consolida e substitui os arquivos
> `00_Checkpoint_Detalhado.md`, `00_Checkpoint_Fase0_Concluida.md`,
> `00_Checkpoint_Fase1.md`, `00_Checkpoint_Fase1_Pessoas.md` e
> `00_Checkpoint_Fase1_Recibos_Relatorios.md`.

---

## Resumo do status

| Fase | Status | Escopo principal |
|---|---|---|
| **Fase 0 — Fundação técnica** | ✅ Concluída | Postgres+RLS em Docker, migrações, auth/RBAC, gateway, API containerizada |
| **Fase 1 — MVP** | 🟡 Em andamento (≈85%) | Secretaria/pessoas, financeiro, recibos, relatórios |
| Fase 2 — Multi-filial | ⏳ Não iniciada | Repasses, ministérios, comunicação |
| Fase 3 — Governança | ⏳ Não iniciada | Atas, votações, patrimônio |
| Fase 4 — IA/IoT | ⏳ Não iniciada | Predição de evasão, Open Finance |

---

## 1. Decisões de arquitetura (confirmadas)

| Tema | Decisão |
|---|---|
| Stack | **Go** (monolito modular) + **Python/FastAPI** (futuro: relatórios/IA) + **PostgreSQL 17** |
| Frontend | **Next.js 15** (App Router) + Tailwind v4 |
| Multi-tenant | Banco compartilhado + **RLS** por `branch_id`/`tenant_id` |
| Papéis de banco | **migrador** (`postgres`, dono do schema) e **aplicação** (`chosenerp_app`, não-superuser ⇒ RLS ativo) |
| Integridade | `financial_transactions` e `audit_log` **append-only** com **hash-chain** |
| Login | `auth_lookup_user` — função **SECURITY DEFINER** (fora do escopo RLS) |
| Mensageria/cache | RabbitMQ + Redis (profile `full`, opcionais) |

## 2. Portas (todas > 30000)

| Serviço | Porta host |
|---|---|
| PostgreSQL | **35432** |
| API (Go) | **38080** |
| Webadmin (Next) | **33000** |
| Redis / RabbitMQ / RabbitMQ-mgmt | 36379 / 35672 / 31572 (profile `full`) |
| Prometheus / Grafana | 39090 / 33001 (profile `full`) |

## 3. Fase 0 — Fundação técnica ✅

- `postgres:17-alpine` em Docker (volume `pgdata`), healthcheck, init script de roles.
- **10 migrações** versionadas, embutidas no binário via `go:embed`, aplicadas no boot
  (`schema_migrations`).
- Schema: tenants, branches, users, roles/permissions, members, families,
  member_relationships, visitors, benefactors, ministries, ministry_members,
  small_groups, group_attendance, finance, transfers, documents, consent, audit_log.
- **RLS** com políticas por tabela + helpers `current_branch/current_tenant/is_headquarters`.
- **Append-only + hash-chain** (triggers) em `financial_transactions` e `audit_log`.
- Auth (Go): JWT access/refresh, RBAC, MFA previsto, resolução de tenant/branch.
- Gateway (Go): router, middleware Bearer, transação com `set_config` RLS, auditoria, CORS.
- API containerizada em imagem `FROM scratch` (sem pull de imagens externas).
- Observabilidade: `/metrics` + Prometheus/Grafana (profile `full`).

**Critério de aceite cumprido:** Filial A nunca lê dados da Filial B via API
(isolamento de leitura e escrita validado; Sede enxerga tudo).

## 4. Fase 1 — MVP 🟡

### Secretaria / Pessoas ✅
- Membros: CRUD, **edição de perfil** (`PATCH /members/{id}`).
- **Árvore genealógica + discipulado** (`GET /members/{id}/tree`).
- **Vínculos** cônjuge/pai/filho/discípulo (`POST /members/{id}/relationships`).
- **Famílias**: criar/listar (com contagem)/vincular/listar membros.
- **Visitantes**: CRUD + trilha de acolhimento (`journey_stage`).
- **Benfeitores**: CRUD.
- Carteirinha QR (`POST /members/{id}/card`).

### Financeiro ✅
- Plano de contas (`GET/POST /finance/categories`) + seed de categorias.
- Lançamentos de dízimo/oferta/despesa (`POST /finance/transactions`), append-only.
- Balancete (`GET /finance/balance`).

### Recibos ✅
- **Recibo digital automático** por lançamento (ref + token p/ QR).
- **Layout HTML** pronto p/ impressão (`GET /receipts/{id}`).
- **Envio (outbox)**: `POST /receipts/{id}/send` (e-mail/WhatsApp) +
  `GET /receipts/{id}/deliveries`. Tabela `document_deliveries` (migração 000012).

### Relatórios ✅
- `GET /reports/balance?from&to` → balancete **por mês**.
- `GET /reports/dre?from&to` → **DRE por categoria** + **comparativo** (delta %).

### Webadmin — UI rica (Etapa A + B) ✅
- **Design system** próprio em `components/ui/*` (Button, Card, Badge, Table, Modal/Drawer,
  Tabs, Toast, Skeleton, EmptyState, Pagination, Avatar, StatCard, PageHeader).
- **Tema dark** (toggle) + layout responsivo (sidebar colapsável, topbar, breadcrumbs).
- **Sessão/RBAC**: `AuthProvider` + `useAuth` — carrega `/me`, **auto-refresh do JWT em 401**,
  logout automático; **menu filtrado por permissão**.
- Membros: busca/filtro/paginação + tabela rica; nova página **Perfil 360º** (`/members/[id]`)
  com abas (Dados, Contato, Vínculos, Espiritual, Documentos) e edição completa.
- Famílias: página nova (criar, vincular, listar).
- Financeiro: KPIs, **gráfico Recharts**, filtros, **detalhe do lançamento** (drawer),
  tela de **Plano de Contas**.
- Relatórios: **gráficos Recharts** (área/barras), seletor de período, **exportação CSV**.
- Visitantes/Benfeitores/Visão Geral: colunas completas, busca, KPIs e gráfico de saldo.

## 5. Validação acumulada

- [x] Isolamento RLS: Sede vê só seu branch; Norte só o dela; Sede vê tudo.
- [x] Escrita cross-branch bloqueada (`WITH CHECK`).
- [x] API conecta como `chosenerp_app` (não-superuser) ⇒ RLS aplicado de verdade.
- [x] Login, RBAC e refresh token funcionais.
- [x] Série mensal e DRE corretos; RLS também nos relatórios (Norte vazio).
- [x] Recibo HTML renderizado; envio registrado (`status=sent`) e histórico listado.
- [x] `go build ./...` e `next build` OK; rotas HTTP 200.

### Correções relevantes
- Postgres nativo (`D:\Postgres`) ocupa IPv4:5432 ⇒ Docker movido para **35432**.
- Docker Hub com TLS instável ⇒ serviços opcionais sob `--profile full`;
  imagem da API usa `FROM scratch` (sem pull).
- Conflito de rotas no ServeMux: recibos movidos para `/api/v1/receipts/…`.

## 6. Notas de ambiente

- `python` não está no PATH (usar `py`).
- Build da imagem exige binário Linux pré-compilado (ver `start.ps1`).
- Credenciais dev: `admin@demo.local`/`admin123` (Sede) e
  `pastor.norte@demo.local`/`norte123` (Norte).

## 7. Próximos passos (Fase 1 → Fase 2)

- [ ] Envio real: SMTP + WhatsApp Business API (substituir `Dispatch` simulado).
- [ ] Exportação de relatórios (CSV/PDF) e DRE anual.
- [ ] App do membro v1 (carteirinha + avisos).
- [ ] Fase 2: repasses entre filiais, ministérios/escalas, check-in infantil,
      integração WhatsApp, doações recorrentes.

---

*Documento vivo — atualizar a cada fechamento de fase.*
