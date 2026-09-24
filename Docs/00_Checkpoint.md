# Chosen ERP - Checkpoint (historico consolidado)

**Ultima atualizacao:** Setembro/2026
**Responsavel:** tompavanello (proprietario) + opencode

> Este e o **unico** checkpoint do projeto. Ele consolida e substitui os arquivos
> `00_Checkpoint_Detalhado.md`, `00_Checkpoint_Fase0_Concluida.md`,
> `00_Checkpoint_Fase1.md`, `00_Checkpoint_Fase1_Pessoas.md` e
> `00_Checkpoint_Fase1_Recibos_Relatorios.md`.

---

## Resumo do status

| Fase | Status | Escopo principal |
|---|---|---|
| **Fase 0 - Fundacao tecnica** |  ~90% | Postgres+RLS, migracoes, auth/RBAC, gateway, API containerizada, **testes automatizados de RLS** + correcao do vazamento multi-tenant (000016). **Falta:** S3, CI/CD, tracing, MFA |
| **Fase 1 - MVP** |  ~60% | Secretaria/pessoas, financeiro, recibos, relatorios, exportacao, carteirinha publica, **nucleo do Rol de Membros** (endereco/situacao/motivo/historico). **Falta:** LGPD, certificados/cartas, contas a pagar, centro de custo, relatorios demograficos e de aniversariantes, eventos/frequencia, usuarios do sistema |
| **Fase 2 - Multi-filial** |  ~55% | Repasses manuais, ministerios, celulas c/ frequencia, doacoes recorrentes, **sub-congregacoes + consolidado**, **escalas**. **Falta:** split automatico, WhatsApp, OFX, check-in infantil real |
| Fase 3 - Governanca |  parcial | **Modulo 6 entregue (Etapa 7):** atas, votacao (quorum/secreto), assinatura interna, mandatos e convenios. Falta: patrimonio, folha, portal do contador, discipulado, E2EE |
| Fase 4 - IA/IoT |  0% | Predicao de evasao, Open Finance, facial, IoT, marketplace, missoes, casamentos/funerais |

> **Revisao de status (Set/2026):** as Fases 1 e 2 estavam marcadas como "Concluidas",
> mas o confronto com `Chosen_ERP_Documentacao_Completa.md` mostrou lacunas relevantes
> (detalhadas nas secoes 4.4 e 4.6). Os percentuais acima sao estimativa de escopo
> entregue, nao de tempo. Backlog priorizado em **`Docs/02_Backlog.md`**.

---

## 1. Decisoes de arquitetura (confirmadas)

| Tema | Decisao |
|---|---|
| Stack | **Go** (monolito modular) + **Python/FastAPI** (futuro: relatorios/IA) + **PostgreSQL 17** |
| Frontend | **Next.js 15** (App Router) + Tailwind v4 |
| Multi-tenant | Banco compartilhado + **RLS** por `branch_id`/`tenant_id` |
| Papeis de banco | **migrador** (`postgres`, dono do schema) e **aplicacao** (`chosenerp_app`, nao-superuser  RLS ativo) |
| Integridade | `financial_transactions` e `audit_log` **append-only** com **hash-chain** |
| Login | `auth_lookup_user` - funcao **SECURITY DEFINER** (fora do escopo RLS) |
| Mensageria/cache | RabbitMQ + Redis (profile `full`, opcionais) |

## 2. Portas (todas > 30000)

| Servico | Porta host |
|---|---|
| PostgreSQL | **35432** |
| API (Go) | **38080** |
| Webadmin (Next) | **33000** |
| Redis / RabbitMQ / RabbitMQ-mgmt | 36379 / 35672 / 31572 (profile `full`) |
| Prometheus / Grafana | 39090 / 33001 (profile `full`) |

## 3. Fase 0 - Fundacao tecnica 

- `postgres:17-alpine` em Docker (volume `pgdata`), healthcheck, init script de roles.
- **10 migracoes** versionadas, embutidas no binario via `go:embed`, aplicadas no boot
  (`schema_migrations`).
- Schema: tenants, branches, users, roles/permissions, members, families,
  member_relationships, visitors, benefactors, ministries, ministry_members,
  small_groups, group_attendance, finance, transfers, documents, consent, audit_log.
- **RLS** com politicas por tabela + helpers `current_branch/current_tenant/is_headquarters`.
- **Append-only + hash-chain** (triggers) em `financial_transactions` e `audit_log`.
- Auth (Go): JWT access/refresh, RBAC, MFA previsto, resolucao de tenant/branch.
- Gateway (Go): router, middleware Bearer, transacao com `set_config` RLS, auditoria, CORS.
- API containerizada em imagem `FROM scratch` (sem pull de imagens externas).
- Observabilidade: `/metrics` + Prometheus/Grafana (profile `full`).

**Criterio de aceite cumprido:** Filial A nunca le dados da Filial B via API
(isolamento de leitura e escrita validado; Sede enxerga todo o seu tenant).
Agora **automatizado** em `internal/store/rls_test.go` (31 testes) - ver secao 5.
O mesmo esforco revelou e corrigiu um vazamento **cross-tenant** (secao 5.1).

## 4. Fase 1 - MVP  (parcial - ver 4.4)

### Secretaria / Pessoas 
- Membros: CRUD, **edicao de perfil** (`PATCH /members/{id}`, incluindo nome desde a
  migracao do formulario unificado).
- **Arvore genealogica + discipulado** (`GET /members/{id}/tree`).
- **Vinculos** conjuge/pai/filho/discipulo (`POST /members/{id}/relationships`).
- **Familias**: criar/listar (com contagem)/vincular/listar membros, **mais** detalhe,
  edicao (nome, chefe, endereco jsonb) e desvinculo; **codigo `#NNN`** por tenant
  (migracao 000020) e `GET /members/{id}/families`.
- **Cargos e mandatos** (migracao 000020): catalogo `cargos` customizavel pela igreja
  (CRUD + seed dos 8 cargos do requisito) e vinculo `member_cargos` com **N cargos por
  membro**, cada um com inicio, vencimento e situacao Ativo/Encerrado.
- **Foto do membro**: upload real em disco local (`POST/DELETE /members/{id}/photo`),
  exibida no grid, no perfil e na carteirinha publica.
- **Visitantes**: CRUD + trilha de acolhimento (`journey_stage`).
- **Benfeitores**: CRUD.
- Carteirinha QR (`POST /members/{id}/card`) - **idempotente** (indice unico parcial
  em `documents(member_id) WHERE kind='membership_card'`) e agora **devolve o token**,
  o que permite abrir/imprimir a carteirinha direto do grid.

### Financeiro 
- Plano de contas (`GET/POST /finance/categories`) + seed de categorias.
- Lancamentos de dizimo/oferta/despesa (`POST /finance/transactions`), append-only.
- Balancete (`GET /finance/balance`).

### Recibos 
- **Recibo digital automatico** por lancamento (ref + token p/ QR).
- **Layout HTML** pronto p/ impressao (`GET /receipts/{id}`).
- **Envio (outbox)**: `POST /receipts/{id}/send` (e-mail/WhatsApp) +
  `GET /receipts/{id}/deliveries`. Tabela `document_deliveries` (migracao 000012).
- **Envio real** (`internal/delivery`): e-mail via **SMTP** (`net/smtp`) e
  **WhatsApp Business Cloud API** (texto com link de validacao). Sem credencial
  configurada, cai em modo **simulado** (mantem dev). Worker de outbox com
  **retry** de pendentes/falhos (intervalo via `DELIVERY_POLL_SECONDS`, max. 5
  tentativas), processando cada entrega **dentro do escopo RLS** da filial.

### Relatorios 
- `GET /reports/balance?from&to` -> balancete **por mes**.
- `GET /reports/dre?from&to` -> **DRE por categoria** + **comparativo** (delta %).
- **Exportacao** (`GET /reports/balance/export` e `GET /reports/dre/export?format=csv|pdf&year=`)
  - CSV (BOM, `;`) e **HTML de impressao** (PDF via navegador).

### App do membro v1 
- **Carteirinha publica** (`GET /api/v1/public/card/{token}`, sem auth): resolve a
  carteirinha pelo token do QR e devolve membro + avisos no escopo da filial dele.
- **Impressao** (`GET /api/v1/public/card/{token}/print`) - HTML.
- **Avisos** (`announcements`, migracao 000013) + gestao (`GET/POST /api/v1/announcements`).
- Pagina publica `/member/[token]` no webadmin, com **QR real** (`qrcode.react`)
  apontando para a propria pagina, **foto**, numero da carteirinha e filial.

### 4.4 Pendencias da Fase 1 (escopo do PRD ainda nao entregue) 

Secretaria digital (1.4)
- [ ] Cartas de transferencia e recomendacao.
- [ ] Certificados (batismo, casamento, apresentacao de bebes).
- [ ] Transferencia digital de cadastro de membro entre filiais.
- [ ] Hierarquia Sede > Congregacoes > **Sub-congregacoes**: a coluna `branches.parent_id`
      e o `kind='sub_congregation'` **ja existem** (migracao 000002); falta usar
      (politicas RLS ignoram `parent_id`, e nao ha CRUD/UI de hierarquia).

Compliance LGPD (1.5) - *risco legal, bloqueia comercializacao*
- [ ] API/UI de termo de consentimento (tabelas `consent_terms` e `member_consents` existem, **sem endpoint nem tela**).
- [ ] Exportacao/portabilidade dos dados do titular.
- [ ] Anonimizacao e exclusao sob solicitacao.

Financeiro (2.1 / 2.2)
- [ ] Contas a pagar **vs. pagas** (hoje so ha lancamento de despesa; sem quitacao/vencimento).
- [ ] Anexo de comprovantes digitais (**depende de S3**, tambem pendente da Fase 0).
- [ ] Centro de custo por ministerio/celula (nao existe coluna `cost_center`).
- [ ] Orcamento inteligente com workflow de aprovacao do tesoureiro.
- [ ] Campanha de crowdfunding com pagina publica e termometro (o enum preve, sem implementacao).
- [ ] Recibo com validade juridica para deducao de IR.

Relatorios (Modulo 12)
- [ ] Relatorio de inadimplencia / queda de contribuicao.
- [ ] Dashboard demografico: piramide etaria, distribuicao geografica.
- [ ] Aniversariantes do mes (membros e casamentos).
- [ ] Engajamento e frequencia por ministerio/celula.
  > A Visao Geral hoje mostra apenas: entradas, saidas, saldo, total de membros,
  > total de visitantes e grafico mensal de saldo.

App do membro
- [ ] O item "App do membro - versao basica" do PRD foi atendido apenas como
      **pagina publica** de carteirinha + avisos. Nao ha app nativo nem login de membro.

## 4.5 Fase 2 - Multi-filial  (parcial - ver 4.6)
- **Repasses entre filiais** (`finance/transfers` + migracao 000013 com policy de INSERT):
  `GET/POST /api/v1/finance/transfers`, `GET /api/v1/branches`. RLS: origem/destino/Sede.
- **Ministerios / escalas** (`ministries`, `ministry_members`): `GET/POST /api/v1/ministries`,
  `GET/POST /api/v1/ministries/{id}/members` (voluntarios/coordenadores/lideres).
- **Grupos / celulas + check-in infantil** (`small_groups`, `group_attendance`):
  `GET/POST /api/v1/groups`, `POST/GET /api/v1/groups/{id}/attendance`.
- **Doacoes recorrentes** (`recurring_donations`, migracao 000014):
  `GET/POST /api/v1/finance/recurring`, `PATCH /api/v1/finance/recurring/{id}`;
  worker `finance.RecurringWorker` (intervalo via `RECURRING_POLL_SECONDS`) gera
  lancamentos automaticos **dentro do RLS** de cada filial, com **recibo automatico**.
- **Envio automatico do recibo recorrente**: o worker enfileira o recibo na outbox
  para o doador (e-mail preferencial; senao WhatsApp do membro/benfeitor);
  o worker de entrega (`delivery.Worker`) envia de fato (SMTP/WhatsApp ou simulado).
  `finance.Create` passou a devolver o **id do documento** para permitir o enfileiramento.
- **WhatsApp em massa (Set/2026, #31/#32)**: a outbox `announcement_deliveries`
  ganhou snapshot de titulo/corpo, `source` e `dedupe_key` (migracao
  `000040_whatsapp_mass`); resolvedor de publico segmentado
  (`internal/announcements/audience.go`) com filtros de filial, sexo, estado
  civil, faixa etaria e situacao; `NotificationWorker` enfileira aniversarios,
  lembretes de escala e boas-vindas a visitantes, configuraveis por tenant em
  `notification_settings`. Tela **Comunicados** com pre-visualizacao de publico
  e painel de automacoes.
- **Comunicado editavel com segmentacao e agendamento (Set/2026, #32b)**:
  `000041_announcement_scheduling` guarda `audience_filter`, `channel` e o
  agendamento (`schedule_type` = manual/once/daily/event, com `schedule_at`,
  `schedule_time`, `schedule_event_id` + `schedule_offset_minutes`). `PATCH
  /announcements/{id}` edita o comunicado; `delivery.ScheduleWorker` dispara uma
  vez, diariamente (fuso da igreja) ou N minutos antes/depois de um evento, com
  dedupe por destinatario/periodo. Na tela, o formulario de criacao/edicao traz
  a segmentacao e o agendamento.
- **Excluir comunicado + historico de execucoes (Set/2026)**: `DELETE
  /announcements/{id}` (cascata em entregas/execucoes) e
  `000042_announcement_runs` + `GET /notification-runs` - o `ScheduleWorker`
  grava cada disparo (tipo, periodo, no de destinatarios, data) e a tela
  Comunicados expoe a aba **Execucoes**.
- Frontend: paginas **Repasses**, **Ministerios/Grupos** (abas, modais, check-in) e aba
  **Recorrencias** no Financeiro.

### 4.6 Pendencias da Fase 2 (escopo do PRD ainda nao entregue) 

- [x] **Escalas (`rosters`)**: `rosters` + `roster_assignments` (`000038`), CRUD,
      convocacao com funcao, **confirmacao/recusa de presenca**, **conflito de
      agenda** (`/rosters/{id}/conflicts`) e **sugestao** de voluntarios
      (`/rosters/suggestions`, por ministerio e frequencia, marcando conflito).
      Tela **Escalas** no menu Organizacao. Faltam ferias/disponibilidade
      explicita (#26).
- [ ] **Motor de repasses dinamico (split automatico)**: o `POST /finance/transfers`
      e **manual**. Nao ha regras configuraveis (ex.: 10% filial -> sede, 5% missoes)
      executadas automaticamente no lancamento.
- [x] **Visao consolidada Sede > Filiais**: `GET /api/v1/reports/consolidated` +
      pagina "Consolidado Sede > Filiais" (por filial, respeitando a hierarquia).
- [x] **Sub-congregacoes** (3o nivel hierarquico): RLS com leitura hierarquica
      (`rls_read_scope`, migracao `000036`) e correcao das politicas `*_sel`
      (`000037`); CRUD de filiais com `parent_id` e guarda contra ciclo.
- [x] **Check-in infantil real / Ministerio Kids (Set/2026, #29)**: modulo `internal/kids`
      + migracoes `000043_kids` e `000044_kids_hq_write`. **Trilha** de conteudo com
      **licoes ordenadas** (objetivo, versiculo, texto, materiais); **turmas** por
      faixa etaria ligadas a uma trilha; **participantes** = membros criancas
      matriculados com **responsaveis** (membros) e restricoes alimentares;
      **encontros** que ministram uma licao; **check-in/check-out** com
      responsavel, **codigo de seguranca** e etiqueta para impressao; e
      **relatorio de evolucao** por crianca (frequencia, licoes e progresso na
      trilha). Tela **Kids** no menu Organizacao com as abas Turmas, Conteudo,
      Participantes, Encontros, Check-in e Evolucao. Falta o fluxo tablet/totem.
- [ ] **Mapa de calor geografico** de celulas (ha lat/long em `small_groups`, sem uso).
- [x] **WhatsApp alem do recibo**: aniversariantes, lembretes de escala,
      boas-vindas a visitantes (**#31**), **disparo em massa segmentado (#32)**.
      Migracao `000040_whatsapp_mass` (outbox com snapshot/dedupe + `notification_settings`),
      `internal/delivery/notification_worker.go` (enfileira as automacoes) e
      `internal/announcements/audience.go` (filtros de filial, sexo, estado civil,
      faixa etaria e situacao). Endpoints `POST /announcements/audience/preview`,
      `GET/PATCH /notifications/settings` e `POST /notifications/run`; tela
      **Comunicados** com segmentacao e painel de automacoes.
- [ ] **Trilha de acolhimento automatizada**: `journey_stage` e avancado
      manualmente (`PATCH /visitors/{id}/stage`); sem gatilhos nem agendamento.
- [ ] **Conciliacao bancaria via OFX** (importacao e vinculacao de lancamentos).
- [ ] **Doacoes recorrentes com gateway real**: o worker gera lancamentos internos;
      nao ha debito automatico em cartao/PIX de verdade (sem integracao de pagamentos).
- [ ] **Banco de dons e talentos** (3.1).
- [ ] **Background check** de voluntarios (3.4).

### Plano do cliente - Etapas 0, 1 e 2 (Set/2026) 

Execucao do plano `Docs/03_Plano_de_Execucao_Pendencias.md` (decisoes do cliente
de 23/09/2026).
- **Etapa 0:** `.github/workflows/ci.yml` (Go build/vet/testes de RLS com
  `CHOSEN_TESTS_REQUIRED=1` + typecheck/build do webadmin); migracao
  `000021_seed_north` (filial Norte + `pastor.norte@demo.local` + 9 permissoes do
  `pastor_filial`) - resolve o item #40, que era documentado mas inexistente.
- **Etapa 1 (migracao `000022_member_lifecycle`):** `members.address` (jsonb,
  **endereco por membro**), `CHECK` fechando `membership_status`
  (`active|member|inactive|dismissed|transferred|deceased|other`),
  `members.exit_reason`/`exited_at`, e `member_history` **append-only com
  hash-chain** e RLS. A **classificacao Professos/Nao Professos e derivada**
  (`active`), sem coluna (nao existe "professo inativo").
- **API/UI:** `GET/POST /members/{id}/history`, campos de endereco/motivo/data no
  formulario unico, badge de classificacao e **aba Historico** no perfil do
  membro. O historico e gravado automaticamente no cadastro e nas mudancas de
  situacao.
- **Testes:** `TestMemberHistory_AppendOnly`,
  `TestMemberHistory_CascadeDeleteWhenMemberRemoved` e
  `TestRLS_MemberHistory_InsertCrossBranchBlocked` (a varredura generica de RLS
  passou a cobrir `member_history` sozinha).
- **Etapa 2:** migracao `000023_member_marriage` (`members.marriage_date`);
  `GET /api/v1/reports/birthdays?month=` (nascimentos + casamentos) e
  `GET /api/v1/reports/demographics` (piramide etaria, situacao, estado civil,
  sexo, UF/cidade); **widget de aniversariantes** na Visao Geral e **painel
  demografico** na pagina de Relatorios.
- **Relatorios com menu e exportacao:** cada relatorio virou pagina propria no
  grupo **Relatorios** (Balancete mensal, DRE, Aniversariantes, Demograficos),
  com exportacao em **CSV, Excel (XLSX) e PDF** - `internal/xlsx` gera OOXML sem
  dependencia externa e `internal/httpapi/report_export.go` centraliza os tres
  formatos.
- **Etapa 3 - Usuarios e Acessos:** migracao `000024_users_roles_mfa` (permissoes
  `users.*`, perfis `lider`/`pastor`/`contador`/`visitante`), API de usuarios e
  catalogos (`/users`, `/roles`, `/permissions`), tela "Usuarios e Acessos" e
  **MFA/TOTP** (`internal/auth/totp.go`) com login por codigo.
- **Etapa 4 - Financeiro do cliente:** migracao `000025_legacy_chart_of_accounts`
  (plano de contas do legado com codigos 101-111 e 1-32, re-apontando os
  lancamentos do seed); **Demonstrativo Mensal** (`GET /reports/monthly-statement`
  + export CSV/XLSX/PDF) e pagina **Entradas x Saidas**.
- **Ajustes do financeiro:** removida a tabela redundante
  `financial_classification_types` (`000026`); **conta contabil obrigatoria** no
  lancamento (casando o tipo entrada/saida) com o rotulo renomeado de
  "categoria" para **"Conta contabil"**; **anexo no ato do lancamento**; e
  **importacao em lote por CSV** (`POST /finance/transactions/import`). Corrigido
  o bug do hash-chain (`000027`) que quebrava lancamento sem forma de pagamento.
- **Etapa 5 - Eventos e Frequencia:** migracao `000028_church_events`
  (`event_kinds` com 10 tipos, `church_events`, `event_attendance`,
  `member_frequency_history`). API de tipos/eventos, **chamada nominal** +
  total digitado e **frequencia com historico** (`/members/{id}/frequency`).
  Tela **Eventos** (com checklist de presenca) e aba **Frequencia** no membro.
- **Melhorias de eventos (`000029`):** modo de presenca (chamada **ou** numero),
  custo estimado, eventos multi-dia e **convocados** por pessoa/ministerio
  (`event_invitees`); visao de **calendario**.
- **UI de eventos/financeiro:** cor por tipo de evento (`000030`), calendario com
  clique-no-dia para criar, **arrastar para mover** (preservando a duracao),
  convocados no hover, drawers mais largos e descricao do lancamento com 2000
  caracteres (icones de envio removidos do grid).
- **Etapa 6 - LGPD:** `internal/lgpd` com **consentimento** (termos + registro),
  **portabilidade** (`/members/{id}/export` em JSON) e **anonimizacao**
  (`/members/{id}/anonymize`); aba **LGPD** no membro. Exclusao fisica nao e
  oferecida (retencao fiscal do livro financeiro).
- **Etapa 7 - Governanca (Modulo 6):** migracao `000033_governance` e pacote
  `internal/governance`. **Atas digitais** (`minutes`) com **assinatura
  eletronica interna** (`POST /minutes/{id}/sign`; a ata e congelada ao assinar)
  e `minute_signatures` **append-only com hash-chain**. **Votacao eletronica**
  (`votes`/`vote_options`): **quorum obrigatorio** e **voto secreto** - a
  participacao (`vote_registrations`) fica separada da escolha (`vote_ballots`,
  tambem hash-chain), entao a apuracao devolve so contagens. **Apuracao
  automatica** e **ata automatica** (ao encerrar, `POST /votes/{id}/close`
  apura e anexa o resultado a ata vinculada). **Painel de mandatos**
  (`GET /governance/mandates`) e **convenios/documentacao legal**
  (`legal_documents`) com alerta de vencimento. Tela **Governanca** (abas Atas,
  Votacoes, Mandatos, Convenios). Testes de RLS novos para as 6 tabelas +
  append-only de `minute_signatures`/`vote_ballots`.
- **Ajustes de UI (Set/2026):** lancamento com **estorno** e correcao
  (append-only via `000031`), indicador/atalho de **anexo** no grid (varios anexos
  abrem o detalhe), recibo corrigido (pop-up aberto antes do fetch); **grades mais
  compactas**; **aniversariantes separados** em relatorios de nascimento e
  casamento; card-resumo na Visao Geral (total do mes + quantos hoje);
  **ministerios e grupos** com edicao, exclusao e responsavel.
- **Rateio e importacao (Set/2026):** `financial_event_allocations` (`000032`)
  associa lancamentos a um ou mais eventos (custo real por evento, com divisao
  igualitaria automatica); importacao de lancamentos por **planilha XLSX ou CSV**
  com **mapeamento de colunas** e **linha inicial** (`import/preview` +
  `import`); leitor de XLSX proprio em `internal/xlsx`.
- **Meu perfil, igreja e filiais (Set/2026):** migracao `000034_tenant_settings`
  (permissoes `settings.read/write` + politica de `UPDATE` em `tenants`).
  `PATCH /me` e `POST /me/password` permitem ao usuario editar o proprio
  nome/e-mail e trocar a senha; `GET/PATCH /tenant` edita os dados da igreja
  (razao social, CNPJ, plano, fuso); `POST/PATCH/DELETE /branches` faz o CRUD de
  filiais/congregacoes (pacote `internal/org`; `DELETE` responde **409** quando
  ha membros). A migracao `000035_branch_cnpj` adiciona **`branches.cnpj`**
  (unico por tenant quando informado). Telas **Meu perfil** (acessivel pelo
  avatar/topbar, com MFA) e **Configuracoes** (abas Igreja e Filiais, com CNPJ),
  no menu Organizacao (Sede).
- **Sub-congregacoes e consolidado (Set/2026):** migracao `000036_branch_hierarchy`
  - o gateway grava `app.branch_scope` (filial + descendentes) e `rls_read` passa
  a ler a subarvore (escrita continua exata). A `000037_sel_policies_select_only`
  corrigiu um defeito latente: as politicas `*_sel` eram `FOR ALL`, entao a
  leitura concedia escrita; agora sao `FOR SELECT`. `internal/org` valida
  hierarquia (sem ciclo) e expoe `GET /api/v1/reports/consolidated`; tela
  **Consolidado Sede > Filiais** (membros, visitantes, entradas, saidas e saldo
  por filial). Testes: `TestRLS_BranchSeesSubCongregation` e
  `TestRLS_BranchWriteStaysExact`.
- **Escalas de voluntarios (Set/2026):** migracao `000038_rosters`
  (`rosters` + `roster_assignments`) e pacote `internal/rosters`. CRUD da escala
  (**evento existente OU tipo de evento**), **convocacao** de membros com funcao
  e **confirmacao/recusa** de presenca; **deteccao de conflito** de agenda
  (`GET /rosters/{id}/conflicts`) e **sugestao** de voluntarios
  (`GET /rosters/suggestions`) ordenada por frequencia e com marca de conflito.
  A migracao `000039_roster_event` adiciona `event_kind_id` e `generated_event`:
  a opcao **"Gerar evento automatico"** cria o evento na **grade de eventos** e
  sincroniza os escalados (+ o ministerio) como **convocados/responsaveis**
  (`event_invitees`), refletindo as mudancas dos escalados. Ao excluir a escala,
  a UI pergunta se o **evento gerado** tambem deve sair da grade
  (`DELETE /rosters/{id}?delete_event=true`). Tela **Escalas** no menu
  Organizacao.

>  O guard do append-only libera DELETE quando `pg_trigger_depth() > 1`, isto
> e, quando a linha some por **cascade** da exclusao do membro. Sem isso, apagar
> um membro (ou reverter a conversao de um visitante) quebrava no trigger.

### Webadmin - Etapa D: membros, cargos, familia e layout  (Set/2026)
Rodada disparada pelos requisitos do cliente (`Docs/requisitos_basicos.txt`, CAD100/CAD107)
e pelas planilhas do Rol de Membros (`Docs/exemplos/`).
- **Grid de membros enxuto e informativo**: 12 colunas -> **7** (Membro com foto +
  apelido/idade/profissao/filial, Cargos, Contato fundido, **Carteirinha com numero**,
  Status, Desde, Acoes em dropdown). Sairam a coluna morta "Ultima doacao", o CPF
  isolado, as tres colunas de contato separadas e o **UUID cru** da filial (agora nome).
- **Formulario unico** para incluir e editar (`PersonForm` via `MemberForm`), com
  **upload de foto** e **seletor de N cargos**; os dois formularios artesanais por aba
  do detalhe foram apagados.
- **Carteirinha no grid**: numero + "Emitir"/ver/imprimir, com o token buscado sob
  demanda (`GET /members/{id}/card`).
- **Aba Cargos** no detalhe: mandatos com inicio -> vencimento, situacao, alerta de
  vencimento (`CARGO_EXPIRY_WINDOW_DAYS`) e botao "Gerenciar cargos".
- **Aba Familia** no detalhe substituiu a pagina `/dashboard/families` (removida do
  menu e do disco): criar/renomear/editar endereco/definir chefe/vincular/desvincular.
- **Sidebar** corporativa: `w-56`, itens `text-[13px]`, agrupada em
  Pessoas/Financeiro/Organizacao, acento azul da marca, item ativo por prefixo.
- **Correcoes de base**: `@custom-variant dark` + bloco `.dark` (o toggle de tema
  estava **inerte**), utilitarias movidas para `@layer components` (regra sem camada
  vencia o Tailwind - `p-*` e `pl-*` eram ignorados), `DataTable` com scroll
  horizontal e sem tela em branco abaixo de 1024px, paginacao com janela + elipses.
- Documentados no backlog os gaps do CAD100 (1.2, 1.5-1.8) e o CAD107 - itens
  **#43 a #49**, com modelo de dados sugerido e 4 decisoes a confirmar com o cliente.

### Erros de console corrigidos na revisao da Etapa D (Set/2026)

- **`<th>` dentro de `<th>`** (`components/ui/data-table.tsx`): o `SortableHeader`
  devolvia um `<th>` proprio e o `DataTable` ja criava o `<th>` da coluna - HTML
  invalido, com o React acusando *"In HTML, `<th>` cannot be a child of `<th>`"* e
  quebra de hidratacao em **toda** tabela com ordenacao. Agora o componente devolve
  so o conteudo (texto ou botao).
- **`<button>` dentro de `<button>`** (`components/ui/dropdown.tsx`): o `Dropdown`
  envolvia o `trigger` num `<button>` mesmo quando o chamador ja passava um `Button`
  (caso dos `RowActions` em todas as listagens). Adicionada a prop **`triggerAsChild`**,
  que injeta o toggle no proprio elemento passado.
- **Acoes da linha invisiveis** (`components/people/row-actions.tsx`): o gatilho usava
  `opacity-0 group-hover:opacity-100`, mas a classe `group` estava no `<Link>` do nome
  (outra celula) - so aparecia ao passar o mouse **no nome**, e nunca em tela de toque.
  Passou a ser sempre visivel, em tom discreto.
- **HTTP 500 em `GET /api/v1/finance/balance`** (backend, `internal/finance/finance.go`):
  o SQL usava `$1` em `WHERE ($1 = '' OR type = $1)` mas `QueryRow` era chamado **sem
  argumento** - pgx devolvia `expected 1 arguments, got 0` e a tela de Visao Geral e a
  de Financeiro perdiam os KPIs de entradas/saidas/saldo. Corrigido passando `kind`
  (o filtro `?type=` voltou a funcionar).

> Correcoes de console/hidratacao so aparecem no **build de desenvolvimento** do
> React; `npx tsc --noEmit` nao as detecta, e o `next build` tambem nao falha por
> isso. Como nao ha test runner no frontend, a verificacao e sempre abrir a tela e
> olhar o console do browser.

### Carga do Rol de Membros do cliente (Set/2026) 

O ambiente demo tinha 6 membros de teste. A planilha
`Docs/exemplos/rol_membros_agrupado_por_familia.xlsx` (a mais completa das duas -
tem 4 abas: `Resumo_Familias`, `Familias`, `Nao_agrupados` e `Original`) foi
carregada para dar base real as telas.

- **Script**: `db/seed/rol_membros.sql` - **fora do git** (`.gitignore` -> `db/seed/`),
  porque o conteudo e dado pessoal real: CPF, RG, nascimento, endereco, telefone e
  e-mail de pessoas fisicas. Nao e migracao embutida (`db/embed.go` so embute
  `migrations/`), entao **nao** roda no boot da API: aplica-se a mao, uma vez.
  ```bash
  docker exec -i chosen-postgres psql -U postgres -d chosenerp < db/seed/rol_membros.sql
  ```
- **Volume**: 351 membros, 88 familias, 206 vinculos familiares, 11 cargos novos no
  catalogo (21 no total) e 37 cargos exercidos. UUIDs **deterministicos** (md5 do
  nome) + `ON CONFLICT`, entao reaplicar atualiza em vez de duplicar.
- **Fontes por campo**: a ficha da aba `Original` e a confiavel (CPF, RG, sexo, CEP,
  endereco, cidade/UF, telefone, celular, e-mail, **data de nascimento em serial do
  Excel**, tipo de cadastro, **CARGO** e presenca). A aba `Familias` da o agrupamento
  e o `PAPEL` de cada um. A coluna `ENDERECO` da aba `Familias` esta **desalinhada**
  em boa parte das linhas (o numero do imovel cai em `BAIRRO`), por isso o endereco
  vem da `Original`.
- **Mapeamentos**: `MEMBROS PROFESSO` -> `active`; `NAO PROFESSO` e `NAO E BATIZADO
  AINDA` -> `member`; `INATIVO` -> `inactive` (o texto original fica em
  `extra_json.tipo_cadastro` - e o gap 1.2 do backlog). `phone`/`whatsapp` em E.164
  (`+55...`). Celular/landline `0 - 0` viram `NULL`. CPF entra na coluna so com 11
  digitos, RG so com 6+; o texto original vai para `extra_json.cpf_original` /
  `rg_original`. Presenca (FREQUENTE / POUCA FREQUENCIA / NAO FREQUENTE) nao tem
  coluna - ficou em `extra_json.frequencia` (gap 1.5).
- **Cargos**: a planilha usa `PASTOR(A)`, `PRESBITERO(A)` e `DIACONO(A)`, que casam
  por slug com os do seed `000020`; os outros 11 (comissao fiscal, responsaveis de
  area) foram criados no catalogo. **Nao ha data de inicio** na planilha, entao o
  vinculo fica com `started_at` nulo e o texto cru em `notes`.
- **Familias**: `name` = "Familia AMARAL" e `code` = "001", exatamente como o cliente
  escreve. As duas familias de demonstracao do seed `000009` ocupavam `#001`/`#002` e
  foram movidas para **`#901`/`#902`** (nada foi apagado) para o codigo do Rol valer.
- **Limite conhecido**: em ~3 das 88 familias ha **mais de um casal na mesma casa**
  (BRAGA #013, MARTINS #047, ABREU #084). Nessas, o `PAPEL` da planilha descreve o
  papel da pessoa na **propria** familia nuclear, nao um parentesco com o chefe, e a
  aba Familia mostra um rotulo aproximado. A propria planilha avisa: *"Sugestao
  automatica; confirmar parentesco no cadastro"*.
- Duas pessoas aparecem so na aba de familias (sem ficha na `Original`) e foram
  criadas sem data de nascimento, com a idade da planilha em `extra_json.idade_planilha`.
  `JOSE CARLOS MORELLI` esta em **duas** familias na propria planilha (MORELLI #087 e
  FONSECA #088) - a confirmar com o cliente.

> O gerador e o extrator do `.xlsx` sao descartaveis e ficaram em `%TEMP%/plan_x/`
> (`xlsx2tsv.js` e `gen_seed.js`): nao fazem parte do projeto e o `.sql` gerado e o
> artefato que importa.

### Webadmin - Etapa C: cadastros ricos e Drawer 
- **Formularios completos**: membros passaram de 7 para **16 campos**
  (apelido, CPF, RG, nascimento, sexo, estado civil, profissao, e-mail,
  telefone, WhatsApp, status, cargo, **batismo**, **membro desde**), agrupados
  em secoes (Identificacao / Documentos / Contato / Vida eclesiastica).
- **Modal -> Drawer** nos cadastros (membros, visitantes, benfeitores), com
  tamanho `lg`; componente `Section` para agrupar campos.
- **Editar na lista**: membros podem ser editados por Drawer (PATCH) sem sair
  da listagem; `MemberForm` reutilizado em criar/editar.
- **KPIs (StatCard)** e filtros extras em Membros, Visitantes e Benfeitores;
  tabela de membros com idade e data de ingresso.
- **Visitantes**: campo WhatsApp (migracao 000015), filtro por etapa e
  **trilha visual** em passos; **Benfeitores**: CPF e observacoes em textarea.
- Correcao: aba "Espiritual" exibia `birth_date` no campo **Batismo**.

### Webadmin - UI rica (Etapa A + B) 
- **Design system** proprio em `components/ui/*` (Button, Card, Badge, Table, Modal/Drawer,
  Tabs, Toast, Skeleton, EmptyState, Pagination, Avatar, StatCard, PageHeader).
- **Tema dark** (toggle) + layout responsivo (sidebar colapsavel, topbar, breadcrumbs).
- **Sessao/RBAC**: `AuthProvider` + `useAuth` - carrega `/me`, **auto-refresh do JWT em 401**,
  logout automatico; **menu filtrado por permissao**.
- Membros: busca/filtro/paginacao + tabela rica; nova pagina **Perfil 360o** (`/members/[id]`)
  com abas (Dados, Contato, Vinculos, Espiritual, Documentos) e edicao completa.
- Familias: pagina nova (criar, vincular, listar).
- Financeiro: KPIs, **grafico Recharts**, filtros, **detalhe do lancamento** (drawer),
  tela de **Plano de Contas**.
- Relatorios: **graficos Recharts** (area/barras), seletor de periodo, **exportacao CSV**.
- Visitantes/Benfeitores/Visao Geral: colunas completas, busca, KPIs e grafico de saldo.

## 5. Validacao acumulada

>  **Criterio de saida da Fase 0 atendido (Set/2026).** O isolamento multi-tenant
> passou a ser verificado por **testes automatizados** em `internal/store/rls_test.go`
> - 31 testes / 84 casos (as varreduras genericas cobrem 25 tabelas no eixo tenant
> e 20 no eixo filial, ja incluindo `cargos`). Como rodar: ver `AGENTS.md` -> "Testes".

- [x] Isolamento RLS: Sede ve so seu branch; filial so a dela; Sede ve todo o tenant. *(automatico)*
- [x] Escrita cross-branch bloqueada (`WITH CHECK`). *(automatico)*
- [x] **Isolamento cross-tenant** - corrigido pela migracao 000016. *(automatico)*
- [x] Append-only de `financial_transactions` e `audit_log`. *(automatico)*
- [x] API conecta como `chosenerp_app` (nao-superuser)  RLS aplicado de verdade. *(automatico)*
- [x] Login, RBAC e refresh token funcionais. *(manual)*
- [x] Serie mensal e DRE corretos; RLS tambem nos relatorios. *(manual)*
- [x] Recibo HTML renderizado; envio registrado (`status=sent`) e historico listado. *(manual)*
- [x] `go build ./...`, `go vet ./...` e smoke test de todas as rotas HTTP. *(manual)*

### 5.1 Vulnerabilidade corrigida: vazamento multi-tenant (migracao 000016)

Os testes automatizados encontraram uma falha **critica** que a validacao manual
nao pegou:

- `is_headquarters()` verificava apenas `app.branch_id IS NULL` + papel
  (`super_admin`/`admin_sede`/`system`), **ignorando o tenant**.
- Consequencia: qualquer usuario com escopo "Sede" (`users.branch_id IS NULL`,
  exatamente a persona "Admin da Sede" do PRD 3) lia e gravava em dados de
  **todos os tenants** - membros, financeiro, atas, prontuario, tudo.
- Agravante nas tabelas que admitem `branch_id IS NULL` (categorias, documentos,
  avisos, benfeitores, entregas): uma filial comum via registros globais de
  outros tenants.

Correcao (`db/migrations/000016_tenant_scope_rls.up.sql`):
- `is_headquarters()` passou a exigir `current_tenant() IS NOT NULL`;
- novo `is_system()` para os workers internos (mantem acesso total);
- helpers `rls_read()` / `rls_write()` / `rls_hq()` que checam **tenant e filial**;
- todas as politicas recriadas com esses helpers.

>  `is_system()` concede acesso total. Ele e usado por `store.WithSystem()`
> (worker de entrega, endpoint publico da carteirinha) e pelo worker de
> recorrencias (`role: "system"` com tenant/filial definidos). Qualquer codigo
> novo que use escopo de sistema precisa filtrar por tenant explicitamente.

### 5.2 Outros defeitos encontrados na mesma rodada

- **`docker compose` nao subia (chave `volumes:` duplicada)**: um bloco `volumes:`
  de nivel superior foi inserido **no meio** de `infra/docker-compose.yml` (entre os
  servicos `api` e `redis`), o que aninhava `redis`/`rabbitmq`/`prometheus`/`grafana`
  dentro dele e tornava o bloco real (no fim do arquivo) uma chave repetida:
  `yaml: construct errors: line 131: mapping key "volumes" already defined at line 69`.
  O `start.ps1` abortava na etapa 3 (`start.ps1:70`), **antes** de subir qualquer
  container - e como a migracao `000020_cargos` e aplicada no boot da API, ela ficou
  sem aplicar por toda a rodada. Corrigido movendo `uploads:` para o bloco correto,
  no fim do arquivo (`docker compose config -q` valida).
- **Upload de arquivo falhava com "permission denied" - sem nada no log**: a imagem e
  `FROM scratch` e o container roda como UID `65532`; como `/uploads` **nao existia na
  imagem**, o Docker criava o ponto de montagem do volume nomeado como `root:root` e a
  API nao conseguia gravar. `POST /members/{id}/photo` respondia
  `{"error":"nao foi possivel salvar a imagem"}` e o anexo do financeiro falhava pelo
  mesmo motivo; o handler nao loga o erro, entao o container parecia saudavel.
  Corrigido com `COPY --chown=65532:65532 infra/api/uploads/ /uploads/` no
  `infra/api/Dockerfile` - ao montar um volume nomeado **vazio**, o Docker inicializa
  o volume com o dono/permissoes do diretorio correspondente na imagem. Verificado:
  foto e anexo sobrevivem a `restart`/`recreate` do container e sao servidos sem
  sessao em `GET /api/v1/attachments/{nome}` (o que a carteirinha publica exige).

>  O `infra/api/uploads/` existe so para o dono do diretorio ir para dentro da
> imagem. Apagar esse diretorio (ou o `--chown` do `COPY`) faz o upload voltar a
> falhar em producao, silenciosamente.

- **Tela branca no webadmin ao recarregar logado** (`Cannot read properties of
  undefined (reading 'includes')`): `GET /api/v1/me` devolvia apenas
  `user_id/tenant_id/branch_id/role`, **sem `permissions`** (e sem `id`, `email`,
  `full_name`). Como `AuthProvider` reidrata a sessao por esse endpoint, o
  `user.permissions.includes(...)` do `hasPerm` quebrava e derrubava o layout.
  Corrigido: `auth.Service.Me()` (`internal/auth/service.go`) remonta o perfil
  completo dentro do escopo RLS, e `handleMe` passou a devolver a mesma forma do
  login. O frontend tambem ficou defensivo (`user.permissions ?? []`), porque o
  `localStorage` pode conter um cache gravado por uma versao anterior.
- **HTTP 500 em `GET /ministries` e `GET /groups`**: `LEFT JOIN members` devolvia
  `full_name` NULL quando nao ha lider, e o scan usava `string` (nao anulavel).
  Corrigido em `internal/ministries/ministries.go` e `internal/groups/groups.go`.
- **`GRANT` e por database**: ao criar um banco novo, os `ALTER DEFAULT PRIVILEGES`
  de `db/init/setup.sql` nao sao herdados. O SQL de setup agora esta embutido em
  `db.Setup` (`db/embed.go`) e e reaplicado pelo harness de testes.
- **Usuario `pastor.norte@demo.local` nao existe**: `AGENTS.md`, `README.md` e
  `start.ps1` o documentam, mas **nenhuma migracao o cria** (nem a filial Norte).
  O seed (000009) cria apenas tenant `demo`, a filial "Sede Matriz" e o
  `admin@demo.local`. Itens #40/#41 no backlog.

### 5.3 Incidente: o build do webadmin amarrado a um endereco (Set/2026)

- **Sintoma**: o usuario nao conseguia entrar **em lugar nenhum**, nem em
  `http://localhost:33000` - e as credenciais estavam certas (`POST` direto em
  `localhost:38080/api/v1/auth/login` devolvia **200**).
- **Causa**: o build feito para publicar o app no tunel gravou o dominio publico no
  **bundle do cliente** (`env: { API_URL }` no `next.config.ts`, substituido em build
  time). Prova: o chunk tinha **1** ocorrencia de `chosenerp.mgmconsultoria.com` e
  **zero** de `localhost:38080`. Resultado: ate quem abria o `localhost` tinha o
  browser mandado para a API publica - que na fase interina responde **403** naquele
  caminho. O erro parecia senha errada, mas era roteamento.
- **Agravante**: o formulario de login pre-preenchia a senha `admin123` fixa no
  codigo; depois da rotacao das senhas demo, abrir a tela ja vinha com uma senha
  invalida e o 401 resultante parecia erro do usuario. O campo de senha agora comeca
  vazio nos dois modos.
- **Correcao (arquitetura)**: a API passou a ser **same-origin por caminho relativo**
  - o browser chama `/api/v1/...` sem host e o `rewrites()` do `next.config.ts`
  (`/api/:path*` -> `API_ORIGIN`, config de **servidor**, default
  `http://localhost:38080`) faz o proxy. Nenhum endereco entra no bundle: **o mesmo
  build serve o localhost e o dominio publico**, e apontar para outra API e mudar
  `API_ORIGIN` e reiniciar o `next start`. As rotas publicas da carteirinha sao
  server-side no Go, entao nunca dependeram do bundle - o link publico ficou de pe
  durante todo o incidente.
- **Verificacao**: bundle novo com **0** enderecos absolutos e 74 `/api/v1`;
  `routes-manifest.json` com `source: /api/:path*`; `tsc --noEmit` exit 0; smoke test
  em 33011 antes da troca; depois da troca, login **200** em `localhost:33000` e
  `/api/v1/members` sem token **401** (prova que o rewrite chega na API e ela valida,
  em vez de o Next responder 404). Build anterior guardado em
  `.next.bak-20260922-210231` (rollback).
- **Armadilha**: build com `NEXT_DIST_DIR` **reescreve `tsconfig.json` e
  `next-env.d.ts`**, gravando o distDir alternativo dentro deles (o `next-env.d.ts`
  fica com um `/// <reference path>` pendurado). Reverter com
  `git checkout -- apps/webadmin/tsconfig.json apps/webadmin/next-env.d.ts` depois de
  todo build de validacao.

### Divida tecnica conhecida
- **Sem testes automatizados no frontend** (Next) - so o backend esta coberto.
- **Sem CI/CD** ~~sem CI~~ - **CI adicionado** (`.github/workflows/ci.yml`: Go
  build/vet/testes de RLS com `CHOSEN_TESTS_REQUIRED=1` + typecheck/build do
  webadmin). Faltam ainda **ambientes staging/prod** definidos.
- **Sem object storage (S3)** - anexos de comprovantes e fotos de membros **ja
  funcionam**, mas em **disco local** (`UPLOAD_DIR`, volume `uploads` do Compose).
  Isso amarra os arquivos a uma unica instancia: escalar horizontalmente (ou trocar de
  host) exige migrar para S3/equivalente. Tambem nao ha varredura de virus nem
  expiracao dos arquivos orfaos.
- ~~**Sem MFA**~~ - **MFA/TOTP implementado** (23/09/2026), sem dependencia
  externa; login pede o codigo quando a conta tem MFA ligado.
- **Sem tracing distribuido** (so `/metrics` + Prometheus/Grafana).
- ~~**Sem CRUD de usuarios/perfis no webadmin**~~ - **implementado**
  (23/09/2026): API `/api/v1/users`, `/roles`, `/permissions` e tela
  "Usuarios e Acessos".
- **Sem rota DELETE alguma** na API (exclusao/anomizacao LGPD dependera disso).

### Correcoes relevantes
- Postgres nativo (`D:\Postgres`) ocupa IPv4:5432  Docker movido para **35432**.
- Docker Hub com TLS instavel  servicos opcionais sob `--profile full`;
  imagem da API usa `FROM scratch` (sem pull).
- Conflito de rotas no ServeMux: recibos movidos para `/api/v1/receipts/...`.

## 6. Notas de ambiente

- `python` nao esta no PATH (usar `py`).
- Build da imagem exige binario Linux pre-compilado (ver `start.ps1`).
- Credenciais dev: **`admin@demo.local`** (Sede, `super_admin`) e
  `pastor.norte@demo.local`. As senhas foram rotacionadas em 2026-09-22 e vivem
  no `.env` (`DEMO_ADMIN_PASSWORD` / `DEMO_NORTE_PASSWORD`), nao na documentacao.
  O `pastor.norte@demo.local` **nao e criado por nenhuma migracao** - ver 5.2;
  a conta existe no banco atual porque foi criada a mao.
- O banco `chosenerp_test` e recriado do zero a cada execucao dos testes de RLS
  (`DROP ... WITH (FORCE)`); o banco de desenvolvimento nao e tocado.

## 7. Proximos passos

### Concluido
- [x] **Testes automatizados de RLS** (`internal/store/*_test.go`) - criterio de
      saida da Fase 0. Inclui correcao do vazamento multi-tenant (migracao 000016).
- [x] Correcao do HTTP 500 em `/ministries` e `/groups` (scan de `leader_name` NULL).
- [x] Correcao da tela branca no reload do webadmin (`/me` sem `permissions`).
- [x] Envio real (SMTP + WhatsApp Business API), com worker de retry - *ainda
      requer credenciais de producao via `.env` (`SMTP_*`, `WHATSAPP_TOKEN`,
      `WHATSAPP_PHONE_ID`)*.
- [x] Exportacao de relatorios (CSV/PDF) e DRE anual (`?year=`).
- [x] Carteirinha publica + avisos (escopo minimo do app do membro).
- [x] Fase 2 parcial: repasses manuais entre filiais, ministerios (vinculos),
      grupos/celulas com frequencia, doacoes recorrentes (worker + recibo automatico).

### A fazer - ver backlog priorizado em `Docs/02_Backlog.md`
- [ ] **P0** LGPD: consentimento, portabilidade, exclusao/anonimizacao.
- [x] **P0** CI/CD executando `go test` + `go vet` (com `CHOSEN_TESTS_REQUIRED=1`)
      - `.github/workflows/ci.yml` (23/09/2026).
- [x] **P0** Seed real da filial Norte e do usuario `pastor.norte@demo.local`
      - migracao `000021_seed_north` (resolvido em 23/09/2026).
- [x] **P1** Etapas 1-3 do plano (`Docs/03_Plano_de_Execucao_Pendencias.md`):
      nucleo do Rol de Membros, aniversariantes/demograficos + relatorios
      exportaveis, usuarios/acessos + MFA.
- [ ] **P1** Seguir o plano a partir da **Etapa 8+** (Fase 2: escalas, split,
      check-in infantil, WhatsApp em massa, OFX; depois Fase 3 restante).
- [ ] **P1** Fechar Fase 1: certificados/cartas, contas a pagar, centro de custo,
          relatorios demograficos e de inadimplencia.
- [ ] **P1** S3 (destrava anexo de comprovantes e fotos).
- [ ] **P2** Fechar Fase 2: escalas, sub-congregacoes, split automatico de repasses,
          check-in infantil real, WhatsApp em massa, OFX.
- [x] **P3** Fase 3 (Modulo 6): atas/votacoes, assinatura interna, mandatos e convenios - **Etapa 7**.
- [ ] **P3** Fase 3 (restante): patrimonio, folha, portal do contador,
          discipulado, prontuario pastoral E2EE.
- [ ] **P4** Fase 4: IA de evasao, Open Finance, facial, IoT, marketplace,
          missoes, casamentos/funerais/capelania.

---

*Documento vivo - atualizar a cada fechamento de fase.*
