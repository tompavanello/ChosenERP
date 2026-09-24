# Chosen ERP — Backlog priorizado

**Fonte:** confronto entre `Docs/Chosen_ERP_Documentacao_Completa.md` (PRD/roadmap)
e o estado real do código em Set/2026.
**Status geral:** Fase 0 ~80% · Fase 1 ~55% · Fase 2 ~45% · Fases 3 e 4 = 0%.

Legenda de prioridade:
- **P0** — bloqueia comercialização ou critério de aceite de fase já vencido.
- **P1** — fecha o escopo de MVP declarado (Fase 1).
- **P2** — fecha o escopo multi-filial declarado (Fase 2).
- **P3/P4** — Fases 3 e 4, conforme roadmap.

---

## P0 — Dívida técnica e risco legal

| # | Item | Fase | Referência no PRD | Status |
|---|---|---|---|---|
| 1 | **Testes automatizados de RLS** (Go) | 0 | Critério de aceite da Fase 0 | ✅ **Feito** — ver nota abaixo |
| 2 | **CI/CD** (build + testes + lint em Go e Next) e definição dos ambientes dev/staging/prod. | 0 | Fase 0 | ⏳ |
| 3 | **LGPD — consentimento**: endpoint + tela para `consent_terms` / `member_consents` (tabelas já existem, sem uso). | 1 | Módulo 1.5 | ✅ **Feito** (Etapa 6) |
| 4 | **LGPD — portabilidade**: exportação total dos dados do titular (JSON). | 1 | Módulo 1.5 · RNF Portabilidade | ✅ **Feito** (Etapa 6) |
| 5 | **LGPD — exclusão/anonimização**: anonimização de `members` (mantém integridade do financeiro). | 1 | Módulo 1.5 | ✅ **Feito** (Etapa 6 — anonimização; exclusão física não, por retenção fiscal) |
| 6 | **MFA para admins** (TOTP), aproveitando a coluna `mfa_secret` já existente. | 0 | RNF Segurança | ✅ **Feito (23/09/2026)** — TOTP em `internal/auth/totp.go`, login com `code` e tela de ativação com QR |
| 40 | **Seed da filial Norte + usuário `pastor.norte@demo.local`** — documentados em `AGENTS.md`/`README.md`/`start.ps1` mas criados por nenhuma migração. | 0 | Fase 0 | ✅ **Feito** (migração `000021_seed_north`) |
| 41 | **Testes automatizados no frontend** (Next): componentes de UI, `lib/api.ts` (refresh em 401), filtragem do menu por permissão. | 0 | — | ⏳ |
| 42 | **Testes de contrato da API** (Go): garantir que `/auth/login` e `/me` devolvem a mesma forma de `user` (com `permissions`) e que os envelopes `{members:[]}`, `{transactions:[]}` não mudem sem aviso. Falhas desse tipo derrubam o webadmin em runtime sem erro de compilação. | 0 | — | ⏳ |

---

## Item #1 — concluído (Set/2026)

**Entregue:** `internal/store/rls_harness_test.go` (harness) e
`internal/store/rls_test.go` (31 testes / 84 casos, Set/2026). O harness cria um banco
descartável `chosenerp_test`, aplica as migrações do binário, reaplica `db/init/setup.sql`
(GRANT é por database) e semeia fixtures de 2 tenants × 3 filiais.

Cobertura:
- leitura por filial, por id cross-branch, escopo Sede e **cross-tenant**;
- varredura genérica: `TestRLS_NoCrossTenantReadForAnyTable` (25 tabelas) e
  `TestRLS_NoCrossBranchReadForAnyTable` (20 tabelas) — a lista vem do próprio
  catálogo (`columnsLike`), então tabela nova entra na varredura sozinha;
- escrita cross-branch bloqueada (membros, financeiro, repasses, recorrências);
- `member_relationships` / `ministry_members` / `member_cargos` herdam o escopo da
  tabela pai (lista fixa no teste — **é o único ponto que exige edição manual** ao
  criar uma tabela-filha nova, porque essas tabelas não têm `tenant_id` próprio);
- append-only de `financial_transactions` e `audit_log`;
- `audit_log` ilegível para perfis de filial;
- `TestRLS_AllTenantScopedTablesHaveRLS` — falha se alguma tabela nova com
  `tenant_id`/`branch_id` ficar sem RLS.

**Vulnerabilidade encontrada e corrigida:** vazamento **cross-tenant** —
`is_headquarters()` ignorava o tenant, então qualquer usuário "Sede" lia e gravava
em todos os tenants. Corrigido pela migração `000016_tenant_scope_rls` (detalhes em
`00_Checkpoint.md` §5.1).

**Como rodar:** ver `AGENTS.md` → "Testes". Sem os DSNs de teste, a suíte pula
(use `CHOSEN_TESTS_REQUIRED=1` no CI para falhar em vez de pular).

---

## P1 — Fechar a Fase 1 (MVP)

### Secretaria digital (Módulo 1.4)
| # | Item | Notas |
|---|---|---|
| 7 | **Certificados** (batismo, casamento, apresentação de bebês) | reaproveitar `documents` + `internal/documents/render.go` |
| 8 | **Cartas de transferência e recomendação** | idem |
| 9 | **Transferência de membro entre filiais** | mutação de `branch_id` — exige atenção ao RLS (`WITH CHECK`) e ao `audit_log` |
| 10 | **Sub-congregações** (3º nível) | ✅ **Feito (Set/2026)** — leitura hierárquica no RLS (`rls_read_scope`, migração `000036`); CRUD já existia com `parent_id`/`kind`; guarda contra ciclo na API; item de menu e validação. Correção `000037`: políticas `*_sel` passaram a ser `FOR SELECT` (antes eram `ALL` e davam escrita pela leitura) |

### Financeiro (Módulo 2.1 / 2.2)
| # | Item | Notas |
|---|---|---|
| 11 | **S3 / object storage** | item pendente da Fase 0. **Não é mais bloqueio de #12/#13**: ambos foram entregues sobre disco local |
| 12 | **Anexo de comprovantes** nos lançamentos | ✅ **Feito** (Set/2026) — `financial_attachments` (000019) + `POST /finance/transactions/{id}/attachments` e `GET /attachments/{filename}` em disco local |
| 13 | **Upload de foto** do membro | ✅ **Feito** (Set/2026) — `POST/DELETE /members/{id}/photo`, mesmo padrão de disco local de #12; foto no grid, no perfil e na carteirinha pública |
| 14 | **Contas a pagar vs. pagas** | hoje só existe lançamento de despesa; falta `due_date`, `paid_at`, quitação |
| 15 | **Centro de custo** por ministério/célula | adicionar `cost_center` em `financial_transactions` + migração |
| 16 | **Orçamento (budgeting)** com workflow de aprovação do tesoureiro | novo domínio `internal/budget` |
| 17 | **Recibo com validade jurídica para dedução de IR** | dados do doador (CPF/CNPJ) + layout específico |
| 18 | **Crowdfunding** com página pública e termômetro de meta | o enum de `type` já prevê `crowdfunding` |

### Relatórios (Módulo 12)
| # | Item | Notas |
|---|---|---|
| 19 | **Relatório de inadimplência / queda de contribuição** | com sigilo preservado (perfil) |
| 20 | **Dashboard demográfico**: pirâmide etária + distribuição geográfica | ✅ **Feito (23/09/2026)** — `GET /reports/demographics` + painel Recharts em Relatórios |
| 21 | **Aniversariantes do mês** (membros e casamentos) | ✅ **Feito (23/09/2026)** — `GET /reports/birthdays` + widget na Visão Geral; casamento usa `members.marriage_date` (`000023`) |
| 22 | **Engajamento e frequência** por ministério/célula | reaproveitar `group_attendance` |

### Acessos
| # | Item | Notas |
|---|---|---|
| 23 | **CRUD de usuários/perfis** (`GET/POST/PATCH /api/v1/users`) + tela no webadmin | ✅ **Feito (23/09/2026)** — API + tela "Usuários e Acessos" (perfis, filial, ativar/desativar, redefinir senha) |

---

## P2 — Fechar a Fase 2 (multi-filial)

| # | Item | Notas |
|---|---|---|
| 24 | **Escalas (`rosters`)** | ✅ **Feito (Set/2026)** — `rosters` + `roster_assignments` (`000038`), CRUD, convocação com função, confirmação/recusa de presença (`PATCH /rosters/{id}/assignments/{id}`) e tela Escalas. (`000039`) associa **evento OU tipo de evento** e a opção **gerar evento automático**: cria o evento na grade e sincroniza os escalados como **convocados/responsáveis** |
| 25 | **Detecção de conflito de agenda** | ✅ **Feito (Set/2026)** — `GET /rosters/{id}/conflicts` (sobreposição de horário do mesmo voluntário em outra escala) + alerta na tela |
| 26 | **Escalas inteligentes** (disponibilidade, férias, frequência mínima na célula) | 🟡 **Parcial (Set/2026)** — `GET /rosters/suggestions` sugere membros do ministério, ordena por frequência recente e marca conflito. Faltam férias/disponibilidade explícita |
| 27 | **Motor de repasses dinâmico (split automático)** | hoje `POST /finance/transfers` é manual; criar regras configuráveis (% por destino) executadas no lançamento |
| 28 | **Painel consolidado Sede > Filiais** | ✅ **Feito (Set/2026)** — `GET /api/v1/reports/consolidated` + página "Consolidado Sede > Filiais" (membros, visitantes, entradas, saídas e saldo por filial, respeitando o escopo hierárquico) |
| 29 | **Check-in infantil real** | ✅ **Feito (Set/2026)** — módulo **Kids** (`000043`/`000044`): trilha com lições ordenadas, turmas por faixa etária, matrícula de membro criança + responsáveis, encontros, **check-in/check-out com código de segurança e etiqueta de impressão**, restrições alimentares e **relatório de evolução**. Falta só o fluxo dedicado tablet/totem |
| 30 | **Mapa de calor geográfico de células** | `small_groups` já tem lat/long |
| 31 | **WhatsApp**: aniversariantes, lembretes de escala, boas-vindas a visitantes | ✅ **Feito (23/09/2026)** — `notification_settings` (`000040`) + `NotificationWorker` enfileira as automações na outbox (dedupe por destinatário); painel de automações na tela Comunicados |
| 32 | **Disparo em massa segmentado** (filtros: filial, sexo, estado civil, faixa etária) | ✅ **Feito (23/09/2026)** — filtros em `ResolveRecipients`/`CountRecipients` (`internal/announcements/audience.go`); `POST /announcements/audience/preview` + segmentação na tela Comunicados |
| 32b | **Comunicado com segmentação salva + agendamento** (editar, uma vez/diário/relativo a evento) | ✅ **Feito (23/09/2026)** — `000041_announcement_scheduling`; `PATCH /announcements/{id}`; `ScheduleWorker` dispara uma vez, diariamente ou N min antes/depois de um evento; editável na tela Comunicados. Inclui **excluir comunicado** (`DELETE /announcements/{id}`) e **histórico de execuções** (`000042_announcement_runs`, `GET /notification-runs` + aba Execuções) |
| 33 | **Trilha de acolhimento automatizada** | hoje `journey_stage` é manual; gatilhos + agendamento |
| 34 | **Conciliação bancária via OFX** | import + vinculação de lançamentos |
| 35 | **Doações recorrentes com gateway real** (débito automático cartão/PIX) | hoje o worker só gera lançamentos internos |
| 36 | **Banco de dons e talentos** (Módulo 3.1) | teste de dons + habilidades profissionais |
| 37 | **Background check** de voluntários (Módulo 3.4) | integração externa |
| 38 | **Tracing distribuído (OpenTelemetry)** | pendente da Fase 0 |

---

## P3 — Fase 3: governança, patrimônio, discipulado

- [x] **Módulo 6**: atas digitais com assinatura eletrônica (`minutes`) — ✅ **Feito (Etapa 7, `000033`)**.
- [x] **Módulo 6**: votação eletrônica para assembleias (`votes`), com apuração e ata automática — ✅ **Feito (Etapa 7)**.
- [x] **Módulo 6**: gestão de mandatos + alertas de vencimento — ✅ **Feito (Etapa 7)**.
- [x] **Módulo 6**: gestão de convênios e documentação legal (escrituras, alvarás, contratos, seguros) com alertas — ✅ **Feito (Etapa 7)**.
- [ ] **Módulo 2.5**: patrimônio (`assets`) com QR Code, depreciação e histórico de manutenção.
- [ ] **Módulo 2.4**: folha de pagamento (`payroll`) — CLT + prebenda/pró-labore.
- [ ] **Módulo 2.4**: portal do contador (acesso externo restrito, sem dados pastorais).
- [ ] **Módulo 5**: escola bíblica/discipulado com trilhas certificadas e pré-requisitos.
- [ ] Prontuário pastoral criptografado (`pastoral_records`, E2EE com chave do pastor).
- [x] Trilha de auditoria imutável (hash-chain) — *já existe em `financial_transactions` e `audit_log`; estendida à governança (`minute_signatures` e `vote_ballots`) na Etapa 7*.

> Critério de aceite da Fase 3: uma assembleia real de igreja parceira realiza a
> votação de diretoria 100% pelo sistema, com ata gerada automaticamente.
> **Módulo 6 entregue (Etapa 7, 23/09/2026):** atas, votação com quórum e voto
> secreto, apuração/ata automática, assinatura interna, mandatos e convênios.

---

## P4 — Fase 4: inteligência, IoT e ecossistema

- [ ] Motor de IA para predição de evasão de membros.
- [ ] Conciliação via Open Finance (substituindo OFX).
- [ ] Reconhecimento facial no check-in infantil.
- [ ] Smart Facilities (IoT: fechaduras, climatização).
- [ ] Chosen Network (marketplace interno de serviços entre membros).
- [ ] Marketplace de API pública (OpenAPI documentada) para integrações de terceiros.
- [ ] Módulo de streaming integrado ao app (pedido de oração e oferta na live).
- [ ] **Módulo 7**: missões e expansão (missionários, sustento, prestação de contas).
- [ ] **Módulo 8**: casamentos, funerais/cuidado ao luto (follow-up 30/60/90), capelania.
- [ ] **Módulo 11**: botão de pânico, gestão de estacionamento.
- [ ] Multi-idioma e multi-moeda.
- [ ] SuperApp white-label (Bíblia, hinário, devocional gamificado) — hoje existe apenas a página pública `/member/[token]`.
- [ ] **Módulo 4**: cultos e eventos (`events`, `service_liturgy`), venda/reserva de vagas, reserva de espaços.

---

## Lacunas do CAD100 / CAD107 — requisitos do cliente (22/09/2026)

Origem: `Docs/requisitos_basicos.txt` (e-mail do cliente) confrontado com o código.
O cliente organiza o Rol de Membros em torno de **frequência, situação e histórico
eclesiástico**. Já coberto:
1.1 dados cadastrais (incluindo **endereço por membro**, migração `000022`),
1.2 classificação (derivada), 1.3 funções/múltiplas funções e 1.4 mandato
(**000020**), 1.6 situação, 1.7 motivo da baixa e **1.8 histórico**
(`000022`, append-only). Faltam **1.5 frequência** e todo o **registro de
eventos** (ver itens abaixo).

| # | Requisito | Item | Prioridade | Modelo de dados sugerido |
|---|---|---|---|---|
| 43 | 1.2 | **Classificação no Rol** (Professos / Não Professos) | P1 | ✅ **Feito (000022)** — **derivada** da situação (`active` = professo), sem coluna própria (não existe "professo inativo") |
| 44 | 1.5 | **Frequência** (Frequente / Pouco frequente / Não frequente) **com histórico** | P2 | ✅ **Feito (000028)** — `member_frequency_history` (vigente = `ended_at IS NULL`); API `GET/POST /members/{id}/frequency` |
| 45 | 1.6 | **Situação do membro** (Ativo / Inativo / Baixado / Transferido / Falecido / Outros) | P1 | ✅ **Feito (000022)** — `CHECK` fechou o domínio: `active\|member\|inactive\|dismissed\|transferred\|deceased\|other` |
| 46 | 1.7 | **Motivo da baixa** | P1 | ✅ **Feito (000022)** — `members.exit_reason` (CHECK) + `members.exited_at` |
| 47 | 1.8 | **Histórico eclesiástico** (data, hora, tipo de evento, observação) | P1 | ✅ **Feito (000022)** — `member_history` append-only com hash-chain + RLS; API `GET/POST /members/{id}/history` e aba Histórico |
| 48 | Registro de Eventos | **Registro de Eventos e Frequência** (módulo inteiro) | P2 | ✅ **Feito (000028)** — `church_events`, `event_kinds`, `event_attendance` e `member_frequency_history`; tela Eventos |
| 49 | §3 | **Painéis** — Rol de Membros, Frequência, Eventos | P3 | Somente leitura sobre #43–#48; combinar com os itens #20/#21/#22 já existentes |

### Decisões que o cliente precisa confirmar antes da implementação

1. **1.2 vs. 1.6 se sobrepõem.** Na própria lista de 1.6 o cliente descreve
   "Ativo **(membro professo)**" e "Inativo **(membro não professo)**" — ou seja, a
   professorate pode ser *derivada* da situação em vez de ser um campo próprio.
   Implementar as duas colunas do jeito literal cria duas fontes de verdade que
   podem divergir. **Perguntar:** professo é atributo independente da situação
   (existe "professo inativo"?) ou é a mesma coisa? A resposta define se #43 é
   coluna ou `CASE` sobre #45.
2. **Frequência automática é impossível só com o CAD107.** O CAD107 registra
   **total de participantes por evento**, não presença por pessoa — com esses
   dados não há como classificar ninguém como "Pouco frequente". Para automatizar
   (item 44) seria preciso `event_attendance (event_id, member_id, presente)` por
   pessoa, que **não está no requisito**. Recomendação: criar a tabela de presença
   desde já (check-in já existe para células em `group_attendance`) e deixar o
   total do CAD107 como campo digitado, para não bloquear o cadastro de eventos
   quando a igreja não fizer chamada nominal.
3. **O histórico (1.8) deve ser consequência, não digitação.** Profissão de fé,
   transferência, baixa e falecimento já são mudanças de situação (#45/#46):
   gravá-las automaticamente em `member_history` evita que a secretaria registre
   o mesmo fato duas vezes e mantém o histórico confiável. O lançamento manual
   fica para os eventos que não mudam situação (batismo infantil, recebido por
   jurisdição).
4. **Endereço do membro (1.1) ainda não existe.** Hoje o endereço mora só em
   `families.address` (jsonb), então um membro sem família não tem endereço.
   Definir se o endereço é da pessoa ou da família antes de modelar — a planilha
   do cliente (`Docs/exemplos/`) sugere **por família**, mas o requisito 1.1 pede
   no cadastro de membros.

### Observação de sequenciamento

#47 (histórico) depende de #45 (situação) e #46 (motivo) — é o último da fila
porque registra as transições dos outros dois. #48 e #44 podem ser feitos em
paralelo a #43/#45. O **#49 (painéis) só faz sentido depois de todos eles**: é
agregação, não cadastro.

---

## Ordem sugerida de execução

```
P0 (#2 CI/CD → #3..#5 LGPD → #6 MFA → #40 seed Norte → #41 testes frontend)
 └─ P1 (#14..#18 financeiro → #7..#10 secretaria → #19..#22 relatórios → #23 usuários)
     ├─ CAD100: #45 situação → #46 motivo → #47 histórico   (nesta ordem)
     ├─ CAD100: #43 classificação no Rol  (em paralelo)
     └─ CAD107: #48 eventos + #44 frequência → #49 painéis
         └─ P2 (#24..#26 escalas → #27 split → #28 consolidado → #29 check-in
                → #30..#33 engajamento → #34 OFX → #35 gateway)
             └─ P3 → P4
```

### Observações de sequenciamento
- **#11 (S3)** deixou de ser pré-requisito: #12 (anexos) e #13 (foto do membro)
  foram entregues sobre disco local. Migrar para S3 depois é troca de backend de
  storage, não refação de feature.
- **#45 → #46 → #47** é a única ordem obrigatória entre os itens do CAD100: o
  histórico registra as transições de situação e de motivo da baixa.
- **#10 (sub-congregações)** altera RLS e afeta #27/#28 — prefira fazer junto.
- **#27 (split automático)** é o item de maior risco financeiro do roadmap;
  exige testes exaustivos e dupla checagem (o PRD lista "erro em cálculo de
  repasses" como risco alto).
- ~~**#1** era pré-requisito de tudo~~ — **concluído**. Qualquer refatoração de
  RLS (#10, #27) agora tem rede de segurança: rode `go test ./internal/store`
  antes e depois.

---

*Documento vivo — atualizar a cada fechamento de item. Ao concluir um item,
marcá-lo aqui e refletir o avanço em `Docs/00_Checkpoint.md`.*
