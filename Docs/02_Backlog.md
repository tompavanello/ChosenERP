# Chosen ERP - Backlog priorizado

**Fonte:** confronto entre `Docs/Chosen_ERP_Documentacao_Completa.md` (PRD/roadmap)
e o estado real do codigo em Set/2026.
**Status geral:** Fase 0 ~80% - Fase 1 ~55% - Fase 2 ~45% - Fases 3 e 4 = 0%.

Legenda de prioridade:
- **P0** - bloqueia comercializacao ou criterio de aceite de fase ja vencido.
- **P1** - fecha o escopo de MVP declarado (Fase 1).
- **P2** - fecha o escopo multi-filial declarado (Fase 2).
- **P3/P4** - Fases 3 e 4, conforme roadmap.

---

## P0 - Divida tecnica e risco legal

| # | Item | Fase | Referencia no PRD | Status |
|---|---|---|---|---|
| 1 | **Testes automatizados de RLS** (Go) | 0 | Criterio de aceite da Fase 0 |  **Feito** - ver nota abaixo |
| 2 | **CI/CD** (build + testes + lint em Go e Next) e definicao dos ambientes dev/staging/prod. | 0 | Fase 0 |  |
| 3 | **LGPD - consentimento**: endpoint + tela para `consent_terms` / `member_consents` (tabelas ja existem, sem uso). | 1 | Modulo 1.5 |  **Feito** (Etapa 6) |
| 4 | **LGPD - portabilidade**: exportacao total dos dados do titular (JSON). | 1 | Modulo 1.5 - RNF Portabilidade |  **Feito** (Etapa 6) |
| 5 | **LGPD - exclusao/anonimizacao**: anonimizacao de `members` (mantem integridade do financeiro). | 1 | Modulo 1.5 |  **Feito** (Etapa 6 - anonimizacao; exclusao fisica nao, por retencao fiscal) |
| 6 | **MFA para admins** (TOTP), aproveitando a coluna `mfa_secret` ja existente. | 0 | RNF Seguranca |  **Feito (23/09/2026)** - TOTP em `internal/auth/totp.go`, login com `code` e tela de ativacao com QR |
| 40 | **Seed da filial Norte + usuario `pastor.norte@demo.local`** - documentados em `AGENTS.md`/`README.md`/`start.ps1` mas criados por nenhuma migracao. | 0 | Fase 0 |  **Feito** (migracao `000021_seed_north`) |
| 41 | **Testes automatizados no frontend** (Next): componentes de UI, `lib/api.ts` (refresh em 401), filtragem do menu por permissao. | 0 | - |  |
| 42 | **Testes de contrato da API** (Go): garantir que `/auth/login` e `/me` devolvem a mesma forma de `user` (com `permissions`) e que os envelopes `{members:[]}`, `{transactions:[]}` nao mudem sem aviso. Falhas desse tipo derrubam o webadmin em runtime sem erro de compilacao. | 0 | - |  |

---

## Item #1 - concluido (Set/2026)

**Entregue:** `internal/store/rls_harness_test.go` (harness) e
`internal/store/rls_test.go` (31 testes / 84 casos, Set/2026). O harness cria um banco
descartavel `chosenerp_test`, aplica as migracoes do binario, reaplica `db/init/setup.sql`
(GRANT e por database) e semeia fixtures de 2 tenants x 3 filiais.

Cobertura:
- leitura por filial, por id cross-branch, escopo Sede e **cross-tenant**;
- varredura generica: `TestRLS_NoCrossTenantReadForAnyTable` (25 tabelas) e
  `TestRLS_NoCrossBranchReadForAnyTable` (20 tabelas) - a lista vem do proprio
  catalogo (`columnsLike`), entao tabela nova entra na varredura sozinha;
- escrita cross-branch bloqueada (membros, financeiro, repasses, recorrencias);
- `member_relationships` / `ministry_members` / `member_cargos` herdam o escopo da
  tabela pai (lista fixa no teste - **e o unico ponto que exige edicao manual** ao
  criar uma tabela-filha nova, porque essas tabelas nao tem `tenant_id` proprio);
- append-only de `financial_transactions` e `audit_log`;
- `audit_log` ilegivel para perfis de filial;
- `TestRLS_AllTenantScopedTablesHaveRLS` - falha se alguma tabela nova com
  `tenant_id`/`branch_id` ficar sem RLS.

**Vulnerabilidade encontrada e corrigida:** vazamento **cross-tenant** -
`is_headquarters()` ignorava o tenant, entao qualquer usuario "Sede" lia e gravava
em todos os tenants. Corrigido pela migracao `000016_tenant_scope_rls` (detalhes em
`00_Checkpoint.md` 5.1).

**Como rodar:** ver `AGENTS.md` -> "Testes". Sem os DSNs de teste, a suite pula
(use `CHOSEN_TESTS_REQUIRED=1` no CI para falhar em vez de pular).

---

## P1 - Fechar a Fase 1 (MVP)

### Secretaria digital (Modulo 1.4)
| # | Item | Notas |
|---|---|---|
| 7 | **Certificados** (batismo, casamento, apresentacao de bebes) | reaproveitar `documents` + `internal/documents/render.go` |
| 8 | **Cartas de transferencia e recomendacao** | idem |
| 9 | **Transferencia de membro entre filiais** | mutacao de `branch_id` - exige atencao ao RLS (`WITH CHECK`) e ao `audit_log` |
| 10 | **Sub-congregacoes** (3o nivel) |  **Feito (Set/2026)** - leitura hierarquica no RLS (`rls_read_scope`, migracao `000036`); CRUD ja existia com `parent_id`/`kind`; guarda contra ciclo na API; item de menu e validacao. Correcao `000037`: politicas `*_sel` passaram a ser `FOR SELECT` (antes eram `ALL` e davam escrita pela leitura) |

### Financeiro (Modulo 2.1 / 2.2)
| # | Item | Notas |
|---|---|---|
| 11 | **S3 / object storage** | item pendente da Fase 0. **Nao e mais bloqueio de #12/#13**: ambos foram entregues sobre disco local |
| 12 | **Anexo de comprovantes** nos lancamentos |  **Feito** (Set/2026) - `financial_attachments` (000019) + `POST /finance/transactions/{id}/attachments` e `GET /attachments/{filename}` em disco local |
| 13 | **Upload de foto** do membro |  **Feito** (Set/2026) - `POST/DELETE /members/{id}/photo`, mesmo padrao de disco local de #12; foto no grid, no perfil e na carteirinha publica |
| 14 | **Contas a pagar vs. pagas** | hoje so existe lancamento de despesa; falta `due_date`, `paid_at`, quitacao |
| 15 | **Centro de custo** por ministerio/celula | adicionar `cost_center` em `financial_transactions` + migracao |
| 16 | **Orcamento (budgeting)** com workflow de aprovacao do tesoureiro | novo dominio `internal/budget` |
| 17 | **Recibo com validade juridica para deducao de IR** | dados do doador (CPF/CNPJ) + layout especifico |
| 18 | **Crowdfunding** com pagina publica e termometro de meta | o enum de `type` ja preve `crowdfunding` |

### Relatorios (Modulo 12)
| # | Item | Notas |
|---|---|---|
| 19 | **Relatorio de inadimplencia / queda de contribuicao** | com sigilo preservado (perfil) |
| 20 | **Dashboard demografico**: piramide etaria + distribuicao geografica |  **Feito (23/09/2026)** - `GET /reports/demographics` + painel Recharts em Relatorios |
| 21 | **Aniversariantes do mes** (membros e casamentos) |  **Feito (23/09/2026)** - `GET /reports/birthdays` + widget na Visao Geral; casamento usa `members.marriage_date` (`000023`) |
| 22 | **Engajamento e frequencia** por ministerio/celula | reaproveitar `group_attendance` |

### Acessos
| # | Item | Notas |
|---|---|---|
| 23 | **CRUD de usuarios/perfis** (`GET/POST/PATCH /api/v1/users`) + tela no webadmin |  **Feito (23/09/2026)** - API + tela "Usuarios e Acessos" (perfis, filial, ativar/desativar, redefinir senha) |

---

## P2 - Fechar a Fase 2 (multi-filial)

| # | Item | Notas |
|---|---|---|
| 24 | **Escalas (`rosters`)** |  **Feito (Set/2026)** - `rosters` + `roster_assignments` (`000038`), CRUD, convocacao com funcao, confirmacao/recusa de presenca (`PATCH /rosters/{id}/assignments/{id}`) e tela Escalas. (`000039`) associa **evento OU tipo de evento** e a opcao **gerar evento automatico**: cria o evento na grade e sincroniza os escalados como **convocados/responsaveis** |
| 25 | **Deteccao de conflito de agenda** |  **Feito (Set/2026)** - `GET /rosters/{id}/conflicts` (sobreposicao de horario do mesmo voluntario em outra escala) + alerta na tela |
| 26 | **Escalas inteligentes** (disponibilidade, ferias, frequencia minima na celula) |  **Parcial (Set/2026)** - `GET /rosters/suggestions` sugere membros do ministerio, ordena por frequencia recente e marca conflito. Faltam ferias/disponibilidade explicita |
| 27 | **Motor de repasses dinamico (split automatico)** | hoje `POST /finance/transfers` e manual; criar regras configuraveis (% por destino) executadas no lancamento |
| 28 | **Painel consolidado Sede > Filiais** |  **Feito (Set/2026)** - `GET /api/v1/reports/consolidated` + pagina "Consolidado Sede > Filiais" (membros, visitantes, entradas, saidas e saldo por filial, respeitando o escopo hierarquico) |
| 29 | **Check-in infantil real** |  **Feito (Set/2026)** - modulo **Kids** (`000043`/`000044`): trilha com licoes ordenadas, turmas por faixa etaria, matricula de membro crianca + responsaveis, encontros, **check-in/check-out com codigo de seguranca e etiqueta de impressao**, restricoes alimentares e **relatorio de evolucao**. Falta so o fluxo dedicado tablet/totem |
| 30 | **Mapa de calor geografico de celulas** | `small_groups` ja tem lat/long |
| 31 | **WhatsApp**: aniversariantes, lembretes de escala, boas-vindas a visitantes |  **Feito (23/09/2026)** - `notification_settings` (`000040`) + `NotificationWorker` enfileira as automacoes na outbox (dedupe por destinatario); painel de automacoes na tela Comunicados |
| 32 | **Disparo em massa segmentado** (filtros: filial, sexo, estado civil, faixa etaria) |  **Feito (23/09/2026)** - filtros em `ResolveRecipients`/`CountRecipients` (`internal/announcements/audience.go`); `POST /announcements/audience/preview` + segmentacao na tela Comunicados |
| 32b | **Comunicado com segmentacao salva + agendamento** (editar, uma vez/diario/relativo a evento) |  **Feito (23/09/2026)** - `000041_announcement_scheduling`; `PATCH /announcements/{id}`; `ScheduleWorker` dispara uma vez, diariamente ou N min antes/depois de um evento; editavel na tela Comunicados. Inclui **excluir comunicado** (`DELETE /announcements/{id}`) e **historico de execucoes** (`000042_announcement_runs`, `GET /notification-runs` + aba Execucoes) |
| 33 | **Trilha de acolhimento automatizada** | hoje `journey_stage` e manual; gatilhos + agendamento |
| 34 | **Conciliacao bancaria via OFX** | import + vinculacao de lancamentos |
| 35 | **Doacoes recorrentes com gateway real** (debito automatico cartao/PIX) | hoje o worker so gera lancamentos internos |
| 36 | **Banco de dons e talentos** (Modulo 3.1) | teste de dons + habilidades profissionais |
| 37 | **Background check** de voluntarios (Modulo 3.4) | integracao externa |
| 38 | **Tracing distribuido (OpenTelemetry)** | pendente da Fase 0 |

---

## P3 - Fase 3: governanca, patrimonio, discipulado

- [x] **Modulo 6**: atas digitais com assinatura eletronica (`minutes`) -  **Feito (Etapa 7, `000033`)**.
- [x] **Modulo 6**: votacao eletronica para assembleias (`votes`), com apuracao e ata automatica -  **Feito (Etapa 7)**.
- [x] **Modulo 6**: gestao de mandatos + alertas de vencimento -  **Feito (Etapa 7)**.
- [x] **Modulo 6**: gestao de convenios e documentacao legal (escrituras, alvaras, contratos, seguros) com alertas -  **Feito (Etapa 7)**.
- [ ] **Modulo 2.5**: patrimonio (`assets`) com QR Code, depreciacao e historico de manutencao.
- [ ] **Modulo 2.4**: folha de pagamento (`payroll`) - CLT + prebenda/pro-labore.
- [ ] **Modulo 2.4**: portal do contador (acesso externo restrito, sem dados pastorais).
- [ ] **Modulo 5**: escola biblica/discipulado com trilhas certificadas e pre-requisitos.
- [ ] Prontuario pastoral criptografado (`pastoral_records`, E2EE com chave do pastor).
- [x] Trilha de auditoria imutavel (hash-chain) - *ja existe em `financial_transactions` e `audit_log`; estendida a governanca (`minute_signatures` e `vote_ballots`) na Etapa 7*.

> Criterio de aceite da Fase 3: uma assembleia real de igreja parceira realiza a
> votacao de diretoria 100% pelo sistema, com ata gerada automaticamente.
> **Modulo 6 entregue (Etapa 7, 23/09/2026):** atas, votacao com quorum e voto
> secreto, apuracao/ata automatica, assinatura interna, mandatos e convenios.

---

## P4 - Fase 4: inteligencia, IoT e ecossistema

- [ ] Motor de IA para predicao de evasao de membros.
- [ ] Conciliacao via Open Finance (substituindo OFX).
- [ ] Reconhecimento facial no check-in infantil.
- [ ] Smart Facilities (IoT: fechaduras, climatizacao).
- [ ] Chosen Network (marketplace interno de servicos entre membros).
- [ ] Marketplace de API publica (OpenAPI documentada) para integracoes de terceiros.
- [ ] Modulo de streaming integrado ao app (pedido de oracao e oferta na live).
- [ ] **Modulo 7**: missoes e expansao (missionarios, sustento, prestacao de contas).
- [ ] **Modulo 8**: casamentos, funerais/cuidado ao luto (follow-up 30/60/90), capelania.
- [ ] **Modulo 11**: botao de panico, gestao de estacionamento.
- [ ] Multi-idioma e multi-moeda.
- [ ] SuperApp white-label (Biblia, hinario, devocional gamificado) - hoje existe apenas a pagina publica `/member/[token]`.
- [ ] **Modulo 4**: cultos e eventos (`events`, `service_liturgy`), venda/reserva de vagas, reserva de espacos.

---

## Lacunas do CAD100 / CAD107 - requisitos do cliente (22/09/2026)

Origem: `Docs/requisitos_basicos.txt` (e-mail do cliente) confrontado com o codigo.
O cliente organiza o Rol de Membros em torno de **frequencia, situacao e historico
eclesiastico**. Ja coberto:
1.1 dados cadastrais (incluindo **endereco por membro**, migracao `000022`),
1.2 classificacao (derivada), 1.3 funcoes/multiplas funcoes e 1.4 mandato
(**000020**), 1.6 situacao, 1.7 motivo da baixa e **1.8 historico**
(`000022`, append-only). Faltam **1.5 frequencia** e todo o **registro de
eventos** (ver itens abaixo).

| # | Requisito | Item | Prioridade | Modelo de dados sugerido |
|---|---|---|---|---|
| 43 | 1.2 | **Classificacao no Rol** (Professos / Nao Professos) | P1 |  **Feito (000022)** - **derivada** da situacao (`active` = professo), sem coluna propria (nao existe "professo inativo") |
| 44 | 1.5 | **Frequencia** (Frequente / Pouco frequente / Nao frequente) **com historico** | P2 |  **Feito (000028)** - `member_frequency_history` (vigente = `ended_at IS NULL`); API `GET/POST /members/{id}/frequency` |
| 45 | 1.6 | **Situacao do membro** (Ativo / Inativo / Baixado / Transferido / Falecido / Outros) | P1 |  **Feito (000022)** - `CHECK` fechou o dominio: `active\|member\|inactive\|dismissed\|transferred\|deceased\|other` |
| 46 | 1.7 | **Motivo da baixa** | P1 |  **Feito (000022)** - `members.exit_reason` (CHECK) + `members.exited_at` |
| 47 | 1.8 | **Historico eclesiastico** (data, hora, tipo de evento, observacao) | P1 |  **Feito (000022)** - `member_history` append-only com hash-chain + RLS; API `GET/POST /members/{id}/history` e aba Historico |
| 48 | Registro de Eventos | **Registro de Eventos e Frequencia** (modulo inteiro) | P2 |  **Feito (000028)** - `church_events`, `event_kinds`, `event_attendance` e `member_frequency_history`; tela Eventos |
| 49 | 3 | **Paineis** - Rol de Membros, Frequencia, Eventos | P3 | Somente leitura sobre #43-#48; combinar com os itens #20/#21/#22 ja existentes |

### Decisoes que o cliente precisa confirmar antes da implementacao

1. **1.2 vs. 1.6 se sobrepoem.** Na propria lista de 1.6 o cliente descreve
   "Ativo **(membro professo)**" e "Inativo **(membro nao professo)**" - ou seja, a
   professorate pode ser *derivada* da situacao em vez de ser um campo proprio.
   Implementar as duas colunas do jeito literal cria duas fontes de verdade que
   podem divergir. **Perguntar:** professo e atributo independente da situacao
   (existe "professo inativo"?) ou e a mesma coisa? A resposta define se #43 e
   coluna ou `CASE` sobre #45.
2. **Frequencia automatica e impossivel so com o CAD107.** O CAD107 registra
   **total de participantes por evento**, nao presenca por pessoa - com esses
   dados nao ha como classificar ninguem como "Pouco frequente". Para automatizar
   (item 44) seria preciso `event_attendance (event_id, member_id, presente)` por
   pessoa, que **nao esta no requisito**. Recomendacao: criar a tabela de presenca
   desde ja (check-in ja existe para celulas em `group_attendance`) e deixar o
   total do CAD107 como campo digitado, para nao bloquear o cadastro de eventos
   quando a igreja nao fizer chamada nominal.
3. **O historico (1.8) deve ser consequencia, nao digitacao.** Profissao de fe,
   transferencia, baixa e falecimento ja sao mudancas de situacao (#45/#46):
   grava-las automaticamente em `member_history` evita que a secretaria registre
   o mesmo fato duas vezes e mantem o historico confiavel. O lancamento manual
   fica para os eventos que nao mudam situacao (batismo infantil, recebido por
   jurisdicao).
4. **Endereco do membro (1.1) ainda nao existe.** Hoje o endereco mora so em
   `families.address` (jsonb), entao um membro sem familia nao tem endereco.
   Definir se o endereco e da pessoa ou da familia antes de modelar - a planilha
   do cliente (`Docs/exemplos/`) sugere **por familia**, mas o requisito 1.1 pede
   no cadastro de membros.

### Observacao de sequenciamento

#47 (historico) depende de #45 (situacao) e #46 (motivo) - e o ultimo da fila
porque registra as transicoes dos outros dois. #48 e #44 podem ser feitos em
paralelo a #43/#45. O **#49 (paineis) so faz sentido depois de todos eles**: e
agregacao, nao cadastro.

---

## Ordem sugerida de execucao

```
P0 (#2 CI/CD -> #3..#5 LGPD -> #6 MFA -> #40 seed Norte -> #41 testes frontend)
  P1 (#14..#18 financeiro -> #7..#10 secretaria -> #19..#22 relatorios -> #23 usuarios)
      CAD100: #45 situacao -> #46 motivo -> #47 historico   (nesta ordem)
      CAD100: #43 classificacao no Rol  (em paralelo)
      CAD107: #48 eventos + #44 frequencia -> #49 paineis
          P2 (#24..#26 escalas -> #27 split -> #28 consolidado -> #29 check-in
                -> #30..#33 engajamento -> #34 OFX -> #35 gateway)
              P3 -> P4
```

### Observacoes de sequenciamento
- **#11 (S3)** deixou de ser pre-requisito: #12 (anexos) e #13 (foto do membro)
  foram entregues sobre disco local. Migrar para S3 depois e troca de backend de
  storage, nao refacao de feature.
- **#45 -> #46 -> #47** e a unica ordem obrigatoria entre os itens do CAD100: o
  historico registra as transicoes de situacao e de motivo da baixa.
- **#10 (sub-congregacoes)** altera RLS e afeta #27/#28 - prefira fazer junto.
- **#27 (split automatico)** e o item de maior risco financeiro do roadmap;
  exige testes exaustivos e dupla checagem (o PRD lista "erro em calculo de
  repasses" como risco alto).
- ~~**#1** era pre-requisito de tudo~~ - **concluido**. Qualquer refatoracao de
  RLS (#10, #27) agora tem rede de seguranca: rode `go test ./internal/store`
  antes e depois.

---

*Documento vivo - atualizar a cada fechamento de item. Ao concluir um item,
marca-lo aqui e refletir o avanco em `Docs/00_Checkpoint.md`.*
