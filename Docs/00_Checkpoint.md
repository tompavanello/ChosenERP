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
| **Fase 0 — Fundação técnica** | 🟡 ~90% | Postgres+RLS, migrações, auth/RBAC, gateway, API containerizada, **testes automatizados de RLS** + correção do vazamento multi-tenant (000016). **Falta:** S3, CI/CD, tracing, MFA |
| **Fase 1 — MVP** | 🟡 ~60% | Secretaria/pessoas, financeiro, recibos, relatórios, exportação, carteirinha pública, **núcleo do Rol de Membros** (endereço/situação/motivo/histórico). **Falta:** LGPD, certificados/cartas, contas a pagar, centro de custo, relatórios demográficos e de aniversariantes, eventos/frequência, usuários do sistema |
| **Fase 2 — Multi-filial** | 🟡 ~55% | Repasses manuais, ministérios, células c/ frequência, doações recorrentes, **sub-congregações + consolidado**, **escalas**. **Falta:** split automático, WhatsApp, OFX, check-in infantil real |
| Fase 3 — Governança | 🟡 parcial | **Módulo 6 entregue (Etapa 7):** atas, votação (quórum/secreto), assinatura interna, mandatos e convênios. Falta: patrimônio, folha, portal do contador, discipulado, E2EE |
| Fase 4 — IA/IoT | ⏳ 0% | Predição de evasão, Open Finance, facial, IoT, marketplace, missões, casamentos/funerais |

> **Revisão de status (Set/2026):** as Fases 1 e 2 estavam marcadas como "Concluídas",
> mas o confronto com `Chosen_ERP_Documentacao_Completa.md` mostrou lacunas relevantes
> (detalhadas nas seções 4.4 e 4.6). Os percentuais acima são estimativa de escopo
> entregue, não de tempo. Backlog priorizado em **`Docs/02_Backlog.md`**.

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
(isolamento de leitura e escrita validado; Sede enxerga todo o seu tenant).
Agora **automatizado** em `internal/store/rls_test.go` (31 testes) — ver seção 5.
O mesmo esforço revelou e corrigiu um vazamento **cross-tenant** (seção 5.1).

## 4. Fase 1 — MVP 🟡 (parcial — ver 4.4)

### Secretaria / Pessoas ✅
- Membros: CRUD, **edição de perfil** (`PATCH /members/{id}`, incluindo nome desde a
  migração do formulário unificado).
- **Árvore genealógica + discipulado** (`GET /members/{id}/tree`).
- **Vínculos** cônjuge/pai/filho/discípulo (`POST /members/{id}/relationships`).
- **Famílias**: criar/listar (com contagem)/vincular/listar membros, **mais** detalhe,
  edição (nome, chefe, endereço jsonb) e desvínculo; **código `#NNN`** por tenant
  (migração 000020) e `GET /members/{id}/families`.
- **Cargos e mandatos** (migração 000020): catálogo `cargos` customizável pela igreja
  (CRUD + seed dos 8 cargos do requisito) e vínculo `member_cargos` com **N cargos por
  membro**, cada um com início, vencimento e situação Ativo/Encerrado.
- **Foto do membro**: upload real em disco local (`POST/DELETE /members/{id}/photo`),
  exibida no grid, no perfil e na carteirinha pública.
- **Visitantes**: CRUD + trilha de acolhimento (`journey_stage`).
- **Benfeitores**: CRUD.
- Carteirinha QR (`POST /members/{id}/card`) — **idempotente** (índice único parcial
  em `documents(member_id) WHERE kind='membership_card'`) e agora **devolve o token**,
  o que permite abrir/imprimir a carteirinha direto do grid.

### Financeiro ✅
- Plano de contas (`GET/POST /finance/categories`) + seed de categorias.
- Lançamentos de dízimo/oferta/despesa (`POST /finance/transactions`), append-only.
- Balancete (`GET /finance/balance`).

### Recibos ✅
- **Recibo digital automático** por lançamento (ref + token p/ QR).
- **Layout HTML** pronto p/ impressão (`GET /receipts/{id}`).
- **Envio (outbox)**: `POST /receipts/{id}/send` (e-mail/WhatsApp) +
  `GET /receipts/{id}/deliveries`. Tabela `document_deliveries` (migração 000012).
- **Envio real** (`internal/delivery`): e-mail via **SMTP** (`net/smtp`) e
  **WhatsApp Business Cloud API** (texto com link de validação). Sem credencial
  configurada, cai em modo **simulado** (mantém dev). Worker de outbox com
  **retry** de pendentes/falhos (intervalo via `DELIVERY_POLL_SECONDS`, máx. 5
  tentativas), processando cada entrega **dentro do escopo RLS** da filial.

### Relatórios ✅
- `GET /reports/balance?from&to` → balancete **por mês**.
- `GET /reports/dre?from&to` → **DRE por categoria** + **comparativo** (delta %).
- **Exportação** (`GET /reports/balance/export` e `GET /reports/dre/export?format=csv|pdf&year=`)
  — CSV (BOM, `;`) e **HTML de impressão** (PDF via navegador).

### App do membro v1 ✅
- **Carteirinha pública** (`GET /api/v1/public/card/{token}`, sem auth): resolve a
  carteirinha pelo token do QR e devolve membro + avisos no escopo da filial dele.
- **Impressão** (`GET /api/v1/public/card/{token}/print`) — HTML.
- **Avisos** (`announcements`, migração 000013) + gestão (`GET/POST /api/v1/announcements`).
- Página pública `/member/[token]` no webadmin, com **QR real** (`qrcode.react`)
  apontando para a própria página, **foto**, número da carteirinha e filial.

### 4.4 Pendências da Fase 1 (escopo do PRD ainda não entregue) ❌

Secretaria digital (1.4)
- [ ] Cartas de transferência e recomendação.
- [ ] Certificados (batismo, casamento, apresentação de bebês).
- [ ] Transferência digital de cadastro de membro entre filiais.
- [ ] Hierarquia Sede > Congregações > **Sub-congregações**: a coluna `branches.parent_id`
      e o `kind='sub_congregation'` **já existem** (migração 000002); falta usar
      (políticas RLS ignoram `parent_id`, e não há CRUD/UI de hierarquia).

Compliance LGPD (1.5) — *risco legal, bloqueia comercialização*
- [ ] API/UI de termo de consentimento (tabelas `consent_terms` e `member_consents` existem, **sem endpoint nem tela**).
- [ ] Exportação/portabilidade dos dados do titular.
- [ ] Anonimização e exclusão sob solicitação.

Financeiro (2.1 / 2.2)
- [ ] Contas a pagar **vs. pagas** (hoje só há lançamento de despesa; sem quitação/vencimento).
- [ ] Anexo de comprovantes digitais (**depende de S3**, também pendente da Fase 0).
- [ ] Centro de custo por ministério/célula (não existe coluna `cost_center`).
- [ ] Orçamento inteligente com workflow de aprovação do tesoureiro.
- [ ] Campanha de crowdfunding com página pública e termômetro (o enum prevê, sem implementação).
- [ ] Recibo com validade jurídica para dedução de IR.

Relatórios (Módulo 12)
- [ ] Relatório de inadimplência / queda de contribuição.
- [ ] Dashboard demográfico: pirâmide etária, distribuição geográfica.
- [ ] Aniversariantes do mês (membros e casamentos).
- [ ] Engajamento e frequência por ministério/célula.
  > A Visão Geral hoje mostra apenas: entradas, saídas, saldo, total de membros,
  > total de visitantes e gráfico mensal de saldo.

App do membro
- [ ] O item "App do membro — versão básica" do PRD foi atendido apenas como
      **página pública** de carteirinha + avisos. Não há app nativo nem login de membro.

## 4.5 Fase 2 — Multi-filial 🟡 (parcial — ver 4.6)
- **Repasses entre filiais** (`finance/transfers` + migração 000013 com policy de INSERT):
  `GET/POST /api/v1/finance/transfers`, `GET /api/v1/branches`. RLS: origem/destino/Sede.
- **Ministérios / escalas** (`ministries`, `ministry_members`): `GET/POST /api/v1/ministries`,
  `GET/POST /api/v1/ministries/{id}/members` (voluntários/coordenadores/líderes).
- **Grupos / células + check-in infantil** (`small_groups`, `group_attendance`):
  `GET/POST /api/v1/groups`, `POST/GET /api/v1/groups/{id}/attendance`.
- **Doações recorrentes** (`recurring_donations`, migração 000014):
  `GET/POST /api/v1/finance/recurring`, `PATCH /api/v1/finance/recurring/{id}`;
  worker `finance.RecurringWorker` (intervalo via `RECURRING_POLL_SECONDS`) gera
  lançamentos automáticos **dentro do RLS** de cada filial, com **recibo automático**.
- **Envio automático do recibo recorrente**: o worker enfileira o recibo na outbox
  para o doador (e-mail preferencial; senão WhatsApp do membro/benfeitor);
  o worker de entrega (`delivery.Worker`) envia de fato (SMTP/WhatsApp ou simulado).
  `finance.Create` passou a devolver o **id do documento** para permitir o enfileiramento.
- **WhatsApp em massa (Set/2026, #31/#32)**: a outbox `announcement_deliveries`
  ganhou snapshot de título/corpo, `source` e `dedupe_key` (migração
  `000040_whatsapp_mass`); resolvedor de público segmentado
  (`internal/announcements/audience.go`) com filtros de filial, sexo, estado
  civil, faixa etária e situação; `NotificationWorker` enfileira aniversários,
  lembretes de escala e boas-vindas a visitantes, configuráveis por tenant em
  `notification_settings`. Tela **Comunicados** com pré-visualização de público
  e painel de automações.
- **Comunicado editável com segmentação e agendamento (Set/2026, #32b)**:
  `000041_announcement_scheduling` guarda `audience_filter`, `channel` e o
  agendamento (`schedule_type` = manual/once/daily/event, com `schedule_at`,
  `schedule_time`, `schedule_event_id` + `schedule_offset_minutes`). `PATCH
  /announcements/{id}` edita o comunicado; `delivery.ScheduleWorker` dispara uma
  vez, diariamente (fuso da igreja) ou N minutos antes/depois de um evento, com
  dedupe por destinatário/período. Na tela, o formulário de criação/edição traz
  a segmentação e o agendamento.
- **Excluir comunicado + histórico de execuções (Set/2026)**: `DELETE
  /announcements/{id}` (cascata em entregas/execuções) e
  `000042_announcement_runs` + `GET /notification-runs` — o `ScheduleWorker`
  grava cada disparo (tipo, período, nº de destinatários, data) e a tela
  Comunicados expõe a aba **Execuções**.
- Frontend: páginas **Repasses**, **Ministérios/Grupos** (abas, modais, check-in) e aba
  **Recorrências** no Financeiro.

### 4.6 Pendências da Fase 2 (escopo do PRD ainda não entregue) ❌

- [x] **Escalas (`rosters`)**: `rosters` + `roster_assignments` (`000038`), CRUD,
      convocação com função, **confirmação/recusa de presença**, **conflito de
      agenda** (`/rosters/{id}/conflicts`) e **sugestão** de voluntários
      (`/rosters/suggestions`, por ministério e frequência, marcando conflito).
      Tela **Escalas** no menu Organização. Faltam férias/disponibilidade
      explícita (#26).
- [ ] **Motor de repasses dinâmico (split automático)**: o `POST /finance/transfers`
      é **manual**. Não há regras configuráveis (ex.: 10% filial → sede, 5% missões)
      executadas automaticamente no lançamento.
- [x] **Visão consolidada Sede > Filiais**: `GET /api/v1/reports/consolidated` +
      página "Consolidado Sede > Filiais" (por filial, respeitando a hierarquia).
- [x] **Sub-congregações** (3º nível hierárquico): RLS com leitura hierárquica
      (`rls_read_scope`, migração `000036`) e correção das políticas `*_sel`
      (`000037`); CRUD de filiais com `parent_id` e guarda contra ciclo.
- [x] **Check-in infantil real / Ministério Kids (Set/2026, #29)**: módulo `internal/kids`
      + migrações `000043_kids` e `000044_kids_hq_write`. **Trilha** de conteúdo com
      **lições ordenadas** (objetivo, versículo, texto, materiais); **turmas** por
      faixa etária ligadas a uma trilha; **participantes** = membros crianças
      matriculados com **responsáveis** (membros) e restrições alimentares;
      **encontros** que ministram uma lição; **check-in/check-out** com
      responsável, **código de segurança** e etiqueta para impressão; e
      **relatório de evolução** por criança (frequência, lições e progresso na
      trilha). Tela **Kids** no menu Organização com as abas Turmas, Conteúdo,
      Participantes, Encontros, Check-in e Evolução. Falta o fluxo tablet/totem.
- [ ] **Mapa de calor geográfico** de células (há lat/long em `small_groups`, sem uso).
- [x] **WhatsApp além do recibo**: aniversariantes, lembretes de escala,
      boas-vindas a visitantes (**#31**), **disparo em massa segmentado (#32)**.
      Migração `000040_whatsapp_mass` (outbox com snapshot/dedupe + `notification_settings`),
      `internal/delivery/notification_worker.go` (enfileira as automações) e
      `internal/announcements/audience.go` (filtros de filial, sexo, estado civil,
      faixa etária e situação). Endpoints `POST /announcements/audience/preview`,
      `GET/PATCH /notifications/settings` e `POST /notifications/run`; tela
      **Comunicados** com segmentação e painel de automações.
- [ ] **Trilha de acolhimento automatizada**: `journey_stage` é avançado
      manualmente (`PATCH /visitors/{id}/stage`); sem gatilhos nem agendamento.
- [ ] **Conciliação bancária via OFX** (importação e vinculação de lançamentos).
- [ ] **Doações recorrentes com gateway real**: o worker gera lançamentos internos;
      não há débito automático em cartão/PIX de verdade (sem integração de pagamentos).
- [ ] **Banco de dons e talentos** (3.1).
- [ ] **Background check** de voluntários (3.4).

### Plano do cliente — Etapas 0, 1 e 2 (Set/2026) ✅

Execução do plano `Docs/03_Plano_de_Execucao_Pendencias.md` (decisões do cliente
de 23/09/2026).
- **Etapa 0:** `.github/workflows/ci.yml` (Go build/vet/testes de RLS com
  `CHOSEN_TESTS_REQUIRED=1` + typecheck/build do webadmin); migração
  `000021_seed_north` (filial Norte + `pastor.norte@demo.local` + 9 permissões do
  `pastor_filial`) — resolve o item #40, que era documentado mas inexistente.
- **Etapa 1 (migração `000022_member_lifecycle`):** `members.address` (jsonb,
  **endereço por membro**), `CHECK` fechando `membership_status`
  (`active|member|inactive|dismissed|transferred|deceased|other`),
  `members.exit_reason`/`exited_at`, e `member_history` **append-only com
  hash-chain** e RLS. A **classificação Professos/Não Professos é derivada**
  (`active`), sem coluna (não existe "professo inativo").
- **API/UI:** `GET/POST /members/{id}/history`, campos de endereço/motivo/data no
  formulário único, badge de classificação e **aba Histórico** no perfil do
  membro. O histórico é gravado automaticamente no cadastro e nas mudanças de
  situação.
- **Testes:** `TestMemberHistory_AppendOnly`,
  `TestMemberHistory_CascadeDeleteWhenMemberRemoved` e
  `TestRLS_MemberHistory_InsertCrossBranchBlocked` (a varredura genérica de RLS
  passou a cobrir `member_history` sozinha).
- **Etapa 2:** migração `000023_member_marriage` (`members.marriage_date`);
  `GET /api/v1/reports/birthdays?month=` (nascimentos + casamentos) e
  `GET /api/v1/reports/demographics` (pirâmide etária, situação, estado civil,
  sexo, UF/cidade); **widget de aniversariantes** na Visão Geral e **painel
  demográfico** na página de Relatórios.
- **Relatórios com menu e exportação:** cada relatório virou página própria no
  grupo **Relatórios** (Balancete mensal, DRE, Aniversariantes, Demográficos),
  com exportação em **CSV, Excel (XLSX) e PDF** — `internal/xlsx` gera OOXML sem
  dependência externa e `internal/httpapi/report_export.go` centraliza os três
  formatos.
- **Etapa 3 — Usuários e Acessos:** migração `000024_users_roles_mfa` (permissões
  `users.*`, perfis `lider`/`pastor`/`contador`/`visitante`), API de usuários e
  catálogos (`/users`, `/roles`, `/permissions`), tela "Usuários e Acessos" e
  **MFA/TOTP** (`internal/auth/totp.go`) com login por código.
- **Etapa 4 — Financeiro do cliente:** migração `000025_legacy_chart_of_accounts`
  (plano de contas do legado com códigos 101–111 e 1–32, re-apontando os
  lançamentos do seed); **Demonstrativo Mensal** (`GET /reports/monthly-statement`
  + export CSV/XLSX/PDF) e página **Entradas × Saídas**.
- **Ajustes do financeiro:** removida a tabela redundante
  `financial_classification_types` (`000026`); **conta contábil obrigatória** no
  lançamento (casando o tipo entrada/saída) com o rótulo renomeado de
  "categoria" para **"Conta contábil"**; **anexo no ato do lançamento**; e
  **importação em lote por CSV** (`POST /finance/transactions/import`). Corrigido
  o bug do hash-chain (`000027`) que quebrava lançamento sem forma de pagamento.
- **Etapa 5 — Eventos e Frequência:** migração `000028_church_events`
  (`event_kinds` com 10 tipos, `church_events`, `event_attendance`,
  `member_frequency_history`). API de tipos/eventos, **chamada nominal** +
  total digitado e **frequência com histórico** (`/members/{id}/frequency`).
  Tela **Eventos** (com checklist de presença) e aba **Frequência** no membro.
- **Melhorias de eventos (`000029`):** modo de presença (chamada **ou** número),
  custo estimado, eventos multi-dia e **convocados** por pessoa/ministério
  (`event_invitees`); visão de **calendário**.
- **UI de eventos/financeiro:** cor por tipo de evento (`000030`), calendário com
  clique-no-dia para criar, **arrastar para mover** (preservando a duração),
  convocados no hover, drawers mais largos e descrição do lançamento com 2000
  caracteres (ícones de envio removidos do grid).
- **Etapa 6 — LGPD:** `internal/lgpd` com **consentimento** (termos + registro),
  **portabilidade** (`/members/{id}/export` em JSON) e **anonimização**
  (`/members/{id}/anonymize`); aba **LGPD** no membro. Exclusão física não é
  oferecida (retenção fiscal do livro financeiro).
- **Etapa 7 — Governança (Módulo 6):** migração `000033_governance` e pacote
  `internal/governance`. **Atas digitais** (`minutes`) com **assinatura
  eletrônica interna** (`POST /minutes/{id}/sign`; a ata é congelada ao assinar)
  e `minute_signatures` **append-only com hash-chain**. **Votação eletrônica**
  (`votes`/`vote_options`): **quórum obrigatório** e **voto secreto** — a
  participação (`vote_registrations`) fica separada da escolha (`vote_ballots`,
  também hash-chain), então a apuração devolve só contagens. **Apuração
  automática** e **ata automática** (ao encerrar, `POST /votes/{id}/close`
  apura e anexa o resultado à ata vinculada). **Painel de mandatos**
  (`GET /governance/mandates`) e **convênios/documentação legal**
  (`legal_documents`) com alerta de vencimento. Tela **Governança** (abas Atas,
  Votações, Mandatos, Convênios). Testes de RLS novos para as 6 tabelas +
  append-only de `minute_signatures`/`vote_ballots`.
- **Ajustes de UI (Set/2026):** lançamento com **estorno** e correção
  (append-only via `000031`), indicador/atalho de **anexo** no grid (vários anexos
  abrem o detalhe), recibo corrigido (pop-up aberto antes do fetch); **grades mais
  compactas**; **aniversariantes separados** em relatórios de nascimento e
  casamento; card-resumo na Visão Geral (total do mês + quantos hoje);
  **ministérios e grupos** com edição, exclusão e responsável.
- **Rateio e importação (Set/2026):** `financial_event_allocations` (`000032`)
  associa lançamentos a um ou mais eventos (custo real por evento, com divisão
  igualitária automática); importação de lançamentos por **planilha XLSX ou CSV**
  com **mapeamento de colunas** e **linha inicial** (`import/preview` +
  `import`); leitor de XLSX próprio em `internal/xlsx`.
- **Meu perfil, igreja e filiais (Set/2026):** migração `000034_tenant_settings`
  (permissões `settings.read/write` + política de `UPDATE` em `tenants`).
  `PATCH /me` e `POST /me/password` permitem ao usuário editar o próprio
  nome/e-mail e trocar a senha; `GET/PATCH /tenant` edita os dados da igreja
  (razão social, CNPJ, plano, fuso); `POST/PATCH/DELETE /branches` faz o CRUD de
  filiais/congregações (pacote `internal/org`; `DELETE` responde **409** quando
  há membros). A migração `000035_branch_cnpj` adiciona **`branches.cnpj`**
  (único por tenant quando informado). Telas **Meu perfil** (acessível pelo
  avatar/topbar, com MFA) e **Configurações** (abas Igreja e Filiais, com CNPJ),
  no menu Organização (Sede).
- **Sub-congregações e consolidado (Set/2026):** migração `000036_branch_hierarchy`
  — o gateway grava `app.branch_scope` (filial + descendentes) e `rls_read` passa
  a ler a subárvore (escrita continua exata). A `000037_sel_policies_select_only`
  corrigiu um defeito latente: as políticas `*_sel` eram `FOR ALL`, então a
  leitura concedia escrita; agora são `FOR SELECT`. `internal/org` valida
  hierarquia (sem ciclo) e expõe `GET /api/v1/reports/consolidated`; tela
  **Consolidado Sede > Filiais** (membros, visitantes, entradas, saídas e saldo
  por filial). Testes: `TestRLS_BranchSeesSubCongregation` e
  `TestRLS_BranchWriteStaysExact`.
- **Escalas de voluntários (Set/2026):** migração `000038_rosters`
  (`rosters` + `roster_assignments`) e pacote `internal/rosters`. CRUD da escala
  (**evento existente OU tipo de evento**), **convocação** de membros com função
  e **confirmação/recusa** de presença; **detecção de conflito** de agenda
  (`GET /rosters/{id}/conflicts`) e **sugestão** de voluntários
  (`GET /rosters/suggestions`) ordenada por frequência e com marca de conflito.
  A migração `000039_roster_event` adiciona `event_kind_id` e `generated_event`:
  a opção **"Gerar evento automático"** cria o evento na **grade de eventos** e
  sincroniza os escalados (+ o ministério) como **convocados/responsáveis**
  (`event_invitees`), refletindo as mudanças dos escalados. Ao excluir a escala,
  a UI pergunta se o **evento gerado** também deve sair da grade
  (`DELETE /rosters/{id}?delete_event=true`). Tela **Escalas** no menu
  Organização.

> ⚠️ O guard do append-only libera DELETE quando `pg_trigger_depth() > 1`, isto
> é, quando a linha some por **cascade** da exclusão do membro. Sem isso, apagar
> um membro (ou reverter a conversão de um visitante) quebrava no trigger.

### Webadmin — Etapa D: membros, cargos, família e layout ✅ (Set/2026)
Rodada disparada pelos requisitos do cliente (`Docs/requisitos_basicos.txt`, CAD100/CAD107)
e pelas planilhas do Rol de Membros (`Docs/exemplos/`).
- **Grid de membros enxuto e informativo**: 12 colunas → **7** (Membro com foto +
  apelido/idade/profissão/filial, Cargos, Contato fundido, **Carteirinha com número**,
  Status, Desde, Ações em dropdown). Saíram a coluna morta "Última doação", o CPF
  isolado, as três colunas de contato separadas e o **UUID cru** da filial (agora nome).
- **Formulário único** para incluir e editar (`PersonForm` via `MemberForm`), com
  **upload de foto** e **seletor de N cargos**; os dois formulários artesanais por aba
  do detalhe foram apagados.
- **Carteirinha no grid**: número + "Emitir"/ver/imprimir, com o token buscado sob
  demanda (`GET /members/{id}/card`).
- **Aba Cargos** no detalhe: mandatos com início → vencimento, situação, alerta de
  vencimento (`CARGO_EXPIRY_WINDOW_DAYS`) e botão "Gerenciar cargos".
- **Aba Família** no detalhe substituiu a página `/dashboard/families` (removida do
  menu e do disco): criar/renomear/editar endereço/definir chefe/vincular/desvincular.
- **Sidebar** corporativa: `w-56`, itens `text-[13px]`, agrupada em
  Pessoas/Financeiro/Organização, acento azul da marca, item ativo por prefixo.
- **Correções de base**: `@custom-variant dark` + bloco `.dark` (o toggle de tema
  estava **inerte**), utilitárias movidas para `@layer components` (regra sem camada
  vencia o Tailwind — `p-*` e `pl-*` eram ignorados), `DataTable` com scroll
  horizontal e sem tela em branco abaixo de 1024px, paginação com janela + elipses.
- Documentados no backlog os gaps do CAD100 (1.2, 1.5–1.8) e o CAD107 — itens
  **#43 a #49**, com modelo de dados sugerido e 4 decisões a confirmar com o cliente.

### Erros de console corrigidos na revisão da Etapa D (Set/2026)

- **`<th>` dentro de `<th>`** (`components/ui/data-table.tsx`): o `SortableHeader`
  devolvia um `<th>` próprio e o `DataTable` já criava o `<th>` da coluna — HTML
  inválido, com o React acusando *"In HTML, `<th>` cannot be a child of `<th>`"* e
  quebra de hidratação em **toda** tabela com ordenação. Agora o componente devolve
  só o conteúdo (texto ou botão).
- **`<button>` dentro de `<button>`** (`components/ui/dropdown.tsx`): o `Dropdown`
  envolvia o `trigger` num `<button>` mesmo quando o chamador já passava um `Button`
  (caso dos `RowActions` em todas as listagens). Adicionada a prop **`triggerAsChild`**,
  que injeta o toggle no próprio elemento passado.
- **Ações da linha invisíveis** (`components/people/row-actions.tsx`): o gatilho usava
  `opacity-0 group-hover:opacity-100`, mas a classe `group` estava no `<Link>` do nome
  (outra célula) — só aparecia ao passar o mouse **no nome**, e nunca em tela de toque.
  Passou a ser sempre visível, em tom discreto.
- **HTTP 500 em `GET /api/v1/finance/balance`** (backend, `internal/finance/finance.go`):
  o SQL usava `$1` em `WHERE ($1 = '' OR type = $1)` mas `QueryRow` era chamado **sem
  argumento** — pgx devolvia `expected 1 arguments, got 0` e a tela de Visão Geral e a
  de Financeiro perdiam os KPIs de entradas/saídas/saldo. Corrigido passando `kind`
  (o filtro `?type=` voltou a funcionar).

> Correções de console/hidratação só aparecem no **build de desenvolvimento** do
> React; `npx tsc --noEmit` não as detecta, e o `next build` também não falha por
> isso. Como não há test runner no frontend, a verificação é sempre abrir a tela e
> olhar o console do browser.

### Carga do Rol de Membros do cliente (Set/2026) ✅

O ambiente demo tinha 6 membros de teste. A planilha
`Docs/exemplos/rol_membros_agrupado_por_familia.xlsx` (a mais completa das duas —
tem 4 abas: `Resumo_Familias`, `Familias`, `Nao_agrupados` e `Original`) foi
carregada para dar base real às telas.

- **Script**: `db/seed/rol_membros.sql` — **fora do git** (`.gitignore` → `db/seed/`),
  porque o conteúdo é dado pessoal real: CPF, RG, nascimento, endereço, telefone e
  e-mail de pessoas físicas. Não é migração embutida (`db/embed.go` só embute
  `migrations/`), então **não** roda no boot da API: aplica-se à mão, uma vez.
  ```bash
  docker exec -i chosen-postgres psql -U postgres -d chosenerp < db/seed/rol_membros.sql
  ```
- **Volume**: 351 membros, 88 famílias, 206 vínculos familiares, 11 cargos novos no
  catálogo (21 no total) e 37 cargos exercidos. UUIDs **determinísticos** (md5 do
  nome) + `ON CONFLICT`, então reaplicar atualiza em vez de duplicar.
- **Fontes por campo**: a ficha da aba `Original` é a confiável (CPF, RG, sexo, CEP,
  endereço, cidade/UF, telefone, celular, e-mail, **data de nascimento em serial do
  Excel**, tipo de cadastro, **CARGO** e presença). A aba `Familias` dá o agrupamento
  e o `PAPEL` de cada um. A coluna `ENDEREÇO` da aba `Familias` está **desalinhada**
  em boa parte das linhas (o número do imóvel cai em `BAIRRO`), por isso o endereço
  vem da `Original`.
- **Mapeamentos**: `MEMBROS PROFESSO` → `active`; `NÃO PROFESSO` e `NÃO É BATIZADO
  AINDA` → `member`; `INATIVO` → `inactive` (o texto original fica em
  `extra_json.tipo_cadastro` — é o gap 1.2 do backlog). `phone`/`whatsapp` em E.164
  (`+55…`). Celular/landline `0 - 0` viram `NULL`. CPF entra na coluna só com 11
  dígitos, RG só com 6+; o texto original vai para `extra_json.cpf_original` /
  `rg_original`. Presença (FREQUENTE / POUCA FREQUENCIA / NAO FREQUENTE) não tem
  coluna — ficou em `extra_json.frequencia` (gap 1.5).
- **Cargos**: a planilha usa `PASTOR(A)`, `PRESBITERO(A)` e `DIACONO(A)`, que casam
  por slug com os do seed `000020`; os outros 11 (comissão fiscal, responsáveis de
  área) foram criados no catálogo. **Não há data de início** na planilha, então o
  vínculo fica com `started_at` nulo e o texto cru em `notes`.
- **Famílias**: `name` = "Família AMARAL" e `code` = "001", exatamente como o cliente
  escreve. As duas famílias de demonstração do seed `000009` ocupavam `#001`/`#002` e
  foram movidas para **`#901`/`#902`** (nada foi apagado) para o código do Rol valer.
- **Limite conhecido**: em ~3 das 88 famílias há **mais de um casal na mesma casa**
  (BRAGA #013, MARTINS #047, ABREU #084). Nessas, o `PAPEL` da planilha descreve o
  papel da pessoa na **própria** família nuclear, não um parentesco com o chefe, e a
  aba Família mostra um rótulo aproximado. A própria planilha avisa: *"Sugestão
  automática; confirmar parentesco no cadastro"*.
- Duas pessoas aparecem só na aba de famílias (sem ficha na `Original`) e foram
  criadas sem data de nascimento, com a idade da planilha em `extra_json.idade_planilha`.
  `JOSE CARLOS MORELLI` está em **duas** famílias na própria planilha (MORELLI #087 e
  FONSECA #088) — a confirmar com o cliente.

> O gerador e o extrator do `.xlsx` são descartáveis e ficaram em `%TEMP%/plan_x/`
> (`xlsx2tsv.js` e `gen_seed.js`): não fazem parte do projeto e o `.sql` gerado é o
> artefato que importa.

### Webadmin — Etapa C: cadastros ricos e Drawer ✅
- **Formulários completos**: membros passaram de 7 para **16 campos**
  (apelido, CPF, RG, nascimento, sexo, estado civil, profissão, e-mail,
  telefone, WhatsApp, status, cargo, **batismo**, **membro desde**), agrupados
  em seções (Identificação / Documentos / Contato / Vida eclesiástica).
- **Modal → Drawer** nos cadastros (membros, visitantes, benfeitores), com
  tamanho `lg`; componente `Section` para agrupar campos.
- **Editar na lista**: membros podem ser editados por Drawer (PATCH) sem sair
  da listagem; `MemberForm` reutilizado em criar/editar.
- **KPIs (StatCard)** e filtros extras em Membros, Visitantes e Benfeitores;
  tabela de membros com idade e data de ingresso.
- **Visitantes**: campo WhatsApp (migração 000015), filtro por etapa e
  **trilha visual** em passos; **Benfeitores**: CPF e observações em textarea.
- Correção: aba "Espiritual" exibia `birth_date` no campo **Batismo**.

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

> ✅ **Critério de saída da Fase 0 atendido (Set/2026).** O isolamento multi-tenant
> passou a ser verificado por **testes automatizados** em `internal/store/rls_test.go`
> — 31 testes / 84 casos (as varreduras genéricas cobrem 25 tabelas no eixo tenant
> e 20 no eixo filial, já incluindo `cargos`). Como rodar: ver `AGENTS.md` → "Testes".

- [x] Isolamento RLS: Sede vê só seu branch; filial só a dela; Sede vê todo o tenant. *(automático)*
- [x] Escrita cross-branch bloqueada (`WITH CHECK`). *(automático)*
- [x] **Isolamento cross-tenant** — corrigido pela migração 000016. *(automático)*
- [x] Append-only de `financial_transactions` e `audit_log`. *(automático)*
- [x] API conecta como `chosenerp_app` (não-superuser) ⇒ RLS aplicado de verdade. *(automático)*
- [x] Login, RBAC e refresh token funcionais. *(manual)*
- [x] Série mensal e DRE corretos; RLS também nos relatórios. *(manual)*
- [x] Recibo HTML renderizado; envio registrado (`status=sent`) e histórico listado. *(manual)*
- [x] `go build ./...`, `go vet ./...` e smoke test de todas as rotas HTTP. *(manual)*

### 5.1 Vulnerabilidade corrigida: vazamento multi-tenant (migração 000016)

Os testes automatizados encontraram uma falha **crítica** que a validação manual
não pegou:

- `is_headquarters()` verificava apenas `app.branch_id IS NULL` + papel
  (`super_admin`/`admin_sede`/`system`), **ignorando o tenant**.
- Consequência: qualquer usuário com escopo "Sede" (`users.branch_id IS NULL`,
  exatamente a persona "Admin da Sede" do PRD §3) lia e gravava em dados de
  **todos os tenants** — membros, financeiro, atas, prontuário, tudo.
- Agravante nas tabelas que admitem `branch_id IS NULL` (categorias, documentos,
  avisos, benfeitores, entregas): uma filial comum via registros globais de
  outros tenants.

Correção (`db/migrations/000016_tenant_scope_rls.up.sql`):
- `is_headquarters()` passou a exigir `current_tenant() IS NOT NULL`;
- novo `is_system()` para os workers internos (mantém acesso total);
- helpers `rls_read()` / `rls_write()` / `rls_hq()` que checam **tenant e filial**;
- todas as políticas recriadas com esses helpers.

> ⚠️ `is_system()` concede acesso total. Ele é usado por `store.WithSystem()`
> (worker de entrega, endpoint público da carteirinha) e pelo worker de
> recorrências (`role: "system"` com tenant/filial definidos). Qualquer código
> novo que use escopo de sistema precisa filtrar por tenant explicitamente.

### 5.2 Outros defeitos encontrados na mesma rodada

- **`docker compose` não subia (chave `volumes:` duplicada)**: um bloco `volumes:`
  de nível superior foi inserido **no meio** de `infra/docker-compose.yml` (entre os
  serviços `api` e `redis`), o que aninhava `redis`/`rabbitmq`/`prometheus`/`grafana`
  dentro dele e tornava o bloco real (no fim do arquivo) uma chave repetida:
  `yaml: construct errors: line 131: mapping key "volumes" already defined at line 69`.
  O `start.ps1` abortava na etapa 3 (`start.ps1:70`), **antes** de subir qualquer
  container — e como a migração `000020_cargos` é aplicada no boot da API, ela ficou
  sem aplicar por toda a rodada. Corrigido movendo `uploads:` para o bloco correto,
  no fim do arquivo (`docker compose config -q` valida).
- **Upload de arquivo falhava com "permission denied" — sem nada no log**: a imagem é
  `FROM scratch` e o container roda como UID `65532`; como `/uploads` **não existia na
  imagem**, o Docker criava o ponto de montagem do volume nomeado como `root:root` e a
  API não conseguia gravar. `POST /members/{id}/photo` respondia
  `{"error":"não foi possível salvar a imagem"}` e o anexo do financeiro falhava pelo
  mesmo motivo; o handler não loga o erro, então o container parecia saudável.
  Corrigido com `COPY --chown=65532:65532 infra/api/uploads/ /uploads/` no
  `infra/api/Dockerfile` — ao montar um volume nomeado **vazio**, o Docker inicializa
  o volume com o dono/permissões do diretório correspondente na imagem. Verificado:
  foto e anexo sobrevivem a `restart`/`recreate` do container e são servidos sem
  sessão em `GET /api/v1/attachments/{nome}` (o que a carteirinha pública exige).

> ⚠️ O `infra/api/uploads/` existe só para o dono do diretório ir para dentro da
> imagem. Apagar esse diretório (ou o `--chown` do `COPY`) faz o upload voltar a
> falhar em produção, silenciosamente.

- **Tela branca no webadmin ao recarregar logado** (`Cannot read properties of
  undefined (reading 'includes')`): `GET /api/v1/me` devolvia apenas
  `user_id/tenant_id/branch_id/role`, **sem `permissions`** (e sem `id`, `email`,
  `full_name`). Como `AuthProvider` reidrata a sessão por esse endpoint, o
  `user.permissions.includes(...)` do `hasPerm` quebrava e derrubava o layout.
  Corrigido: `auth.Service.Me()` (`internal/auth/service.go`) remonta o perfil
  completo dentro do escopo RLS, e `handleMe` passou a devolver a mesma forma do
  login. O frontend também ficou defensivo (`user.permissions ?? []`), porque o
  `localStorage` pode conter um cache gravado por uma versão anterior.
- **HTTP 500 em `GET /ministries` e `GET /groups`**: `LEFT JOIN members` devolvia
  `full_name` NULL quando não há líder, e o scan usava `string` (não anulável).
  Corrigido em `internal/ministries/ministries.go` e `internal/groups/groups.go`.
- **`GRANT` é por database**: ao criar um banco novo, os `ALTER DEFAULT PRIVILEGES`
  de `db/init/setup.sql` não são herdados. O SQL de setup agora está embutido em
  `db.Setup` (`db/embed.go`) e é reaplicado pelo harness de testes.
- **Usuário `pastor.norte@demo.local` não existe**: `AGENTS.md`, `README.md` e
  `start.ps1` o documentam, mas **nenhuma migração o cria** (nem a filial Norte).
  O seed (000009) cria apenas tenant `demo`, a filial "Sede Matriz" e o
  `admin@demo.local`. Itens #40/#41 no backlog.

### 5.3 Incidente: o build do webadmin amarrado a um endereço (Set/2026)

- **Sintoma**: o usuário não conseguia entrar **em lugar nenhum**, nem em
  `http://localhost:33000` — e as credenciais estavam certas (`POST` direto em
  `localhost:38080/api/v1/auth/login` devolvia **200**).
- **Causa**: o build feito para publicar o app no túnel gravou o domínio público no
  **bundle do cliente** (`env: { API_URL }` no `next.config.ts`, substituído em build
  time). Prova: o chunk tinha **1** ocorrência de `chosenerp.mgmconsultoria.com` e
  **zero** de `localhost:38080`. Resultado: até quem abria o `localhost` tinha o
  browser mandado para a API pública — que na fase interina responde **403** naquele
  caminho. O erro parecia senha errada, mas era roteamento.
- **Agravante**: o formulário de login pré-preenchia a senha `admin123` fixa no
  código; depois da rotação das senhas demo, abrir a tela já vinha com uma senha
  inválida e o 401 resultante parecia erro do usuário. O campo de senha agora começa
  vazio nos dois modos.
- **Correção (arquitetura)**: a API passou a ser **same-origin por caminho relativo**
  — o browser chama `/api/v1/...` sem host e o `rewrites()` do `next.config.ts`
  (`/api/:path*` → `API_ORIGIN`, config de **servidor**, default
  `http://localhost:38080`) faz o proxy. Nenhum endereço entra no bundle: **o mesmo
  build serve o localhost e o domínio público**, e apontar para outra API é mudar
  `API_ORIGIN` e reiniciar o `next start`. As rotas públicas da carteirinha são
  server-side no Go, então nunca dependeram do bundle — o link público ficou de pé
  durante todo o incidente.
- **Verificação**: bundle novo com **0** endereços absolutos e 74 `/api/v1`;
  `routes-manifest.json` com `source: /api/:path*`; `tsc --noEmit` exit 0; smoke test
  em 33011 antes da troca; depois da troca, login **200** em `localhost:33000` e
  `/api/v1/members` sem token **401** (prova que o rewrite chega na API e ela valida,
  em vez de o Next responder 404). Build anterior guardado em
  `.next.bak-20260922-210231` (rollback).
- **Armadilha**: build com `NEXT_DIST_DIR` **reescreve `tsconfig.json` e
  `next-env.d.ts`**, gravando o distDir alternativo dentro deles (o `next-env.d.ts`
  fica com um `/// <reference path>` pendurado). Reverter com
  `git checkout -- apps/webadmin/tsconfig.json apps/webadmin/next-env.d.ts` depois de
  todo build de validação.

### Dívida técnica conhecida
- **Sem testes automatizados no frontend** (Next) — só o backend está coberto.
- **Sem CI/CD** ~~sem CI~~ — **CI adicionado** (`.github/workflows/ci.yml`: Go
  build/vet/testes de RLS com `CHOSEN_TESTS_REQUIRED=1` + typecheck/build do
  webadmin). Faltam ainda **ambientes staging/prod** definidos.
- **Sem object storage (S3)** — anexos de comprovantes e fotos de membros **já
  funcionam**, mas em **disco local** (`UPLOAD_DIR`, volume `uploads` do Compose).
  Isso amarra os arquivos a uma única instância: escalar horizontalmente (ou trocar de
  host) exige migrar para S3/equivalente. Também não há varredura de vírus nem
  expiração dos arquivos órfãos.
- ~~**Sem MFA**~~ — **MFA/TOTP implementado** (23/09/2026), sem dependência
  externa; login pede o código quando a conta tem MFA ligado.
- **Sem tracing distribuído** (só `/metrics` + Prometheus/Grafana).
- ~~**Sem CRUD de usuários/perfis no webadmin**~~ — **implementado**
  (23/09/2026): API `/api/v1/users`, `/roles`, `/permissions` e tela
  "Usuários e Acessos".
- **Sem rota DELETE alguma** na API (exclusão/anomização LGPD dependerá disso).

### Correções relevantes
- Postgres nativo (`D:\Postgres`) ocupa IPv4:5432 ⇒ Docker movido para **35432**.
- Docker Hub com TLS instável ⇒ serviços opcionais sob `--profile full`;
  imagem da API usa `FROM scratch` (sem pull).
- Conflito de rotas no ServeMux: recibos movidos para `/api/v1/receipts/…`.

## 6. Notas de ambiente

- `python` não está no PATH (usar `py`).
- Build da imagem exige binário Linux pré-compilado (ver `start.ps1`).
- Credenciais dev: **`admin@demo.local`** (Sede, `super_admin`) e
  `pastor.norte@demo.local`. As senhas foram rotacionadas em 2026-09-22 e vivem
  no `.env` (`DEMO_ADMIN_PASSWORD` / `DEMO_NORTE_PASSWORD`), não na documentação.
  O `pastor.norte@demo.local` **não é criado por nenhuma migração** — ver 5.2;
  a conta existe no banco atual porque foi criada à mão.
- O banco `chosenerp_test` é recriado do zero a cada execução dos testes de RLS
  (`DROP ... WITH (FORCE)`); o banco de desenvolvimento não é tocado.

## 7. Próximos passos

### Concluído
- [x] **Testes automatizados de RLS** (`internal/store/*_test.go`) — critério de
      saída da Fase 0. Inclui correção do vazamento multi-tenant (migração 000016).
- [x] Correção do HTTP 500 em `/ministries` e `/groups` (scan de `leader_name` NULL).
- [x] Correção da tela branca no reload do webadmin (`/me` sem `permissions`).
- [x] Envio real (SMTP + WhatsApp Business API), com worker de retry — *ainda
      requer credenciais de produção via `.env` (`SMTP_*`, `WHATSAPP_TOKEN`,
      `WHATSAPP_PHONE_ID`)*.
- [x] Exportação de relatórios (CSV/PDF) e DRE anual (`?year=`).
- [x] Carteirinha pública + avisos (escopo mínimo do app do membro).
- [x] Fase 2 parcial: repasses manuais entre filiais, ministérios (vínculos),
      grupos/células com frequência, doações recorrentes (worker + recibo automático).

### A fazer — ver backlog priorizado em `Docs/02_Backlog.md`
- [ ] **P0** LGPD: consentimento, portabilidade, exclusão/anonimização.
- [x] **P0** CI/CD executando `go test` + `go vet` (com `CHOSEN_TESTS_REQUIRED=1`)
      — `.github/workflows/ci.yml` (23/09/2026).
- [x] **P0** Seed real da filial Norte e do usuário `pastor.norte@demo.local`
      — migração `000021_seed_north` (resolvido em 23/09/2026).
- [x] **P1** Etapas 1–3 do plano (`Docs/03_Plano_de_Execucao_Pendencias.md`):
      núcleo do Rol de Membros, aniversariantes/demográficos + relatórios
      exportáveis, usuários/acessos + MFA.
- [ ] **P1** Seguir o plano a partir da **Etapa 8+** (Fase 2: escalas, split,
      check-in infantil, WhatsApp em massa, OFX; depois Fase 3 restante).
- [ ] **P1** Fechar Fase 1: certificados/cartas, contas a pagar, centro de custo,
          relatórios demográficos e de inadimplência.
- [ ] **P1** S3 (destrava anexo de comprovantes e fotos).
- [ ] **P2** Fechar Fase 2: escalas, sub-congregações, split automático de repasses,
          check-in infantil real, WhatsApp em massa, OFX.
- [x] **P3** Fase 3 (Módulo 6): atas/votações, assinatura interna, mandatos e convênios — **Etapa 7**.
- [ ] **P3** Fase 3 (restante): patrimônio, folha, portal do contador,
          discipulado, prontuário pastoral E2EE.
- [ ] **P4** Fase 4: IA de evasão, Open Finance, facial, IoT, marketplace,
          missões, casamentos/funerais/capelania.

---

*Documento vivo — atualizar a cada fechamento de fase.*
