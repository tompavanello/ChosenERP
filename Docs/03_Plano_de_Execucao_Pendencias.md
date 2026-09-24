# Chosen ERP — Plano de Execução e Pendências (Gap Analysis)

**Data:** 23/09/2026
**Base de comparação:**
- `Docs/requisitos_basicos.txt` — e-mail do cliente (22/09/2026): Rol de Membros e Registro de Eventos/Frequência.
- `Docs/exemplos/` — planilhas do Rol de Membros e relatórios financeiros legados.
- `Docs/Chosen_ERP_Documentacao_Completa.md` (PRD) e `Docs/01_Blueprint_Arquitetura.md`.
- `Docs/00_Checkpoint.md` e `Docs/02_Backlog.md` (estado real do código).

**Objetivo:** consolidar, num único lugar, **tudo que ainda falta** para avançar,
com prioridade, modelo de dados, dependências, decisões já tomadas e ordem de execução.

> Status global herdado do checkpoint: Fase 0 ~90% · Fase 1 ~55% · Fase 2 ~45% ·
> Fases 3 e 4 = 0%. Este documento **não substitui** o `02_Backlog.md`; ele
> reorganiza os gaps pela ótica do que o cliente pediu + do que as planilhas
> revelam, e registra as decisões de produto.

---

## 0. Plano de início — passo a passo (recomendado)

> **Decisões já tomadas (§9):** endereço **por membro** · professorate **derivada**
> da situação (não existe "professo inativo") · plano de contas importado **com os
> mesmos códigos** · votação com **quórum obrigatório, voto secreto e assinatura
> interna** · eventos aceitam **total digitado E chamada nominal**.
>
> **Status de execução (23/09/2026):** ✅ Etapas **0 a 7** concluídas ·
> Etapa 8+ pendente.

A ordem abaixo é a de **menor risco e maior valor** por etapa. Cada etapa é um
bloco entregável (migração + API + tela + teste). Não comece a próxima antes de
fechar a anterior.

### Etapa 0 — Rede de segurança (antes de qualquer migração) ✅
> **Entregue (23/09/2026):** `.github/workflows/ci.yml` (Go build/vet/testes de
> RLS + typecheck/build do webadmin, com `CHOSEN_TESTS_REQUIRED=1`); migração
> `000021_seed_north` (filial Norte + `pastor.norte@demo.local` + permissões do
> `pastor_filial`); suíte de RLS rodando como baseline.
1. **CI/CD mínimo**: GitHub Actions com `go build ./... && go vet ./... && go test ./... -count=1`
   e `CHOSEN_TESTS_REQUIRED=1`; `npm run build` no `webadmin`.
2. **Seed da filial Norte** + usuário `pastor.norte@demo.local` (hoje documentado e inexistente).
3. Rodar `go test ./internal/store` **antes** de tocar em RLS; é a rede que valida as migrações seguintes.
   → *Por quê primeiro: as etapas 1 em diante mexem em `members`, que tem RLS; sem isso você altera schema sem rede.*

### Etapa 1 — Núcleo do Rol de Membros (a base de tudo) ✅
> **Entregue (23/09/2026):** migração `000022_member_lifecycle`
> (`members.address` jsonb, `CHECK` de situação, `exit_reason`/`exited_at`,
> `member_history` append-only com hash-chain e RLS, classificação derivada);
> endpoints `GET/POST /members/{id}/history`; UI com endereço, motivo/data de
> baixa e **aba Histórico**; histórico automático nas mudanças de situação e no
> cadastro; testes novos (`TestMemberHistory_*`). O guard do append-only libera
> a exclusão em cascata quando o membro é apagado (`pg_trigger_depth() > 1`).
Uma migração (`000021_member_lifecycle`) + API + tela:
1. `members.address jsonb` (CEP, logradouro, número, complemento, bairro, cidade, UF) — **endereço por membro**.
2. Fechar `members.membership_status` com `CHECK` (Ativo / Inativo / Baixado / Transferido / Falecido / Outros) e migrar valores legados.
3. `members.exit_reason` (CHECK) + `members.exited_at` — obrigatórios quando situação = baixa/transferência/falecimento.
4. `member_history` **append-only com hash-chain** (data, hora, tipo de evento, observação).
5. **Classificação Professos / Não Professos é derivada** da situação (`CASE`), sem coluna nova.
6. UI: campos de situação/motivo/data e a **aba Histórico** no perfil do membro.
   → *Por quê agora: histórico, aniversariantes, demográficos e frequência todos dependem de `members`.*

### Etapa 2 — Aniversariantes + demográficos (vitória rápida) ✅
> **Entregue (23/09/2026):** migração `000023_member_marriage`
> (`members.marriage_date`); endpoints `GET /api/v1/reports/birthdays?month=`
> (nascimentos + aniversários de casamento, com dedupe de casal) e
> `GET /api/v1/reports/demographics` (pirâmide etária, situação, estado civil,
> sexo e distribuição por UF/cidade); **widget de aniversariantes** na Visão Geral
> e **painel demográfico** na página de Relatórios.
>
> **Menu e exportação (23/09/2026):** cada relatório ganhou **página e item de
> menu próprios** no grupo **Relatórios** (Balancete mensal, DRE, Aniversariantes,
> Demográficos) e **exportação em CSV, Excel (XLSX) e PDF** (versão de impressão
> que o navegador salva em PDF). O XLSX é gerado por `internal/xlsx` (OOXML
> mínimo, sem dependência externa); o CSV/PDF são centralizados em
> `internal/httpapi/report_export.go`.
1. `GET /api/v1/reports/birthdays?month=` (membros e casamentos) + widget no dashboard.
2. Pirâmide etária e distribuição geográfica (agora com o endereço da Etapa 1).
   → *Barato, e é o pedido explícito do cliente; aparece rápido na tela.*

### Etapa 3 — Usuários e Acessos ✅
> **Entregue (23/09/2026):** migração `000024_users_roles_mfa` (permissões
> `users.read/users.write`, perfis `lider`/`pastor`/`contador`/`visitante` e
> permissões-base de cada perfil; `auth_lookup_user` devolve MFA). API:
> `GET/POST /api/v1/users`, `PATCH /api/v1/users/{id}`,
> `POST /api/v1/users/{id}/password`, `GET /api/v1/roles`,
> `GET /api/v1/permissions` (restrito a `super_admin`/`admin_sede`). **MFA/TOTP**
> (`internal/auth/totp.go`, sem dependência externa): `GET /api/v1/auth/mfa`,
> `POST /auth/mfa/{setup,enable,disable}` e login com `code`. Tela
> **Usuários e Acessos** no menu (criar, editar perfil/filial, ativar/desativar,
> redefinir senha) e card de MFA com QR para o próprio usuário.
1. `GET/POST/PATCH /api/v1/users` + `GET /api/v1/roles` e `/permissions`.
2. Perfis faltantes no seed (`pastor_filial`, `lider`, `pastor`, `contador`, `membro`, `visitante`).
3. Tela "Usuários e Acessos" (criar, editar papel, desativar, resetar senha, escopo Sede × Filial).
4. **MFA/TOTP** sobre `mfa_secret`/`mfa_enabled` (PRD RNF).
   → *Sem isso a operação real da igreja fica no usuário único admin.*

### Etapa 4 — Financeiro no formato do cliente ✅
> **Entregue (23/09/2026):** migração `000025_legacy_chart_of_accounts` importa o
> plano de contas do legado com os **mesmos códigos** (receitas 101–111,
> despesas 1–32) em `financial_categories` e `financial_classification_types`,
> re-aponta os lançamentos do seed e desativa as contas genéricas antigas.
> Endpoints `GET /api/v1/reports/monthly-statement?month=YYYY-MM` (Demonstrativo
> Mensal em regime de caixa: contas com totais por semana + saldo inicial/final)
> e `/monthly-statement/export?format=csv|xlsx|pdf`. Tela **Demonstrativo Mensal**
> e **Entradas × Saídas** (pizzas de composição, barras mensais e comparativo com
> o período anterior) no grupo Relatórios.
1. Migração que **substitui** o plano de contas atual pelos **mesmos códigos do legado** (receitas 101–111, despesas 1–32, saldos 501/502).
2. **Demonstrativo Mensal** (regime de caixa): entradas/saídas por semana, saldo inicial/final, aplicações financeiras.
3. **Entradas de Recursos × Saídas de Recursos** (pizza + barras + comparativo anual).
   → *Reproduz as planilhas de `Docs/exemplos/`; é o que o cliente já usa hoje.*

> **Ajustes do financeiro (23/09/2026):** removida a tabela **redundante**
> `financial_classification_types` (migração `000026`) — o lançamento sempre usou
> `financial_categories` (plano de contas); a **conta contábil** virou
> **obrigatória** no lançamento (com casamento de tipo entrada/saída) e o rótulo
> "categoria" foi trocado para **"Conta contábil"**. O formulário de lançamento
> agora aceita **anexo** (comprovante) no momento do registro, e a tela de
> lançamentos ganhou **importação em lote** por CSV (`POST /finance/transactions/import`).
> Corrigido de quebra um bug do hash-chain (`000027`): `payment_method` nulo
> gerava hash NULL e quebrava o INSERT.

### Etapa 5 — Eventos e Frequência ✅
> **Entregue (23/09/2026):** migração `000028_church_events` (`event_kinds` com
> seed dos 10 tipos, `church_events`, `event_attendance` e
> `member_frequency_history`). API: CRUD de tipos (`/event-kinds`), CRUD de
> eventos (`/events`) com filtro por período/tipo, chamada nominal
> (`GET/POST /events/{id}/attendance`) e frequência do membro
> (`GET/POST /members/{id}/frequency`). O **total digitado e a chamada nominal
> coexistem**; a frequência guarda **histórico** (fecha a vigente e insere a
> nova). Tela **Eventos** (eventos + tipos, com checklist de presença) e aba
> **Frequência** no perfil do membro.
>
> **Melhorias de eventos (23/09/2026, migração `000029`):** modo de presença
> (**chamada nominal** OU **apenas o número**), **custo estimado** (opcional),
> **eventos multi-dia** (retiro/acampamento, com data de término) e
> **convocados** (`event_invitees`) — pessoas e/ou **ministérios** obrigados a
> participar. Visão de **calendário** (grade mensal com eventos, inclusive os de
> vários dias).
1. `church_events` + catálogo `event_kinds` (EBD, Culto, Culto Especial, Reunião, Pequeno Grupo, Jovens, Escola Bíblica, Santa Ceia, Vigília, Outros).
2. **Total de participantes digitado E chamada nominal** (`event_attendance`) — as duas opções.
3. `member_frequency_history` (Frequente / Pouco frequente / Não frequente com histórico), alimentada pela chamada quando houver.

### Etapa 6 — LGPD (risco legal) ✅
> **Entregue (23/09/2026):** pacote `internal/lgpd`. **Consentimento**:
> `GET/POST /api/v1/consent-terms` (versão automática) e
> `GET/POST /api/v1/members/{id}/consents` (registro/revogação com IP e
> user-agent). **Portabilidade**: `GET /api/v1/members/{id}/export` devolve em
> JSON todos os dados do titular (ficha, cargos, vínculos, famílias, histórico,
> frequência, consentimentos, doações e presenças). **Anonimização**:
> `POST /api/v1/members/{id}/anonymize` (apaga PII e registra no histórico).
> Aba **LGPD** no perfil do membro. A exclusão física não é oferecida de
> propósito: o livro financeiro é retido por obrigação fiscal — a anonimização
> atende ao art. 18 mantendo a integridade dos lançamentos.

### Etapa 7 — Governança ✅
> **Entregue (23/09/2026):** migração `000033_governance` com o pacote
> `internal/governance`. **Atas digitais** (`minutes` + `minute_signatures`
> append-only com **hash-chain**): CRUD, **assinatura eletrônica interna** (hash
> do conteúdo + credenciais do usuário; a ata é congelada ao assinar) e leitura
> das assinaturas. **Votação eletrônica** (`votes` + `vote_options` +
> `vote_registrations` + `vote_ballots`): **quórum obrigatório**, **voto
> secreto** — a participação fica separada da escolha, então a apuração só
> devolve contagens —, abertura/encerramento, **apuração automática** e **ata
> automática** (o resultado é anexado à ata vinculada ao encerrar). **Painel de
> mandatos** (`GET /governance/mandates`, vigentes/vencendo em 60 dias) e
> **convênios/documentação legal** (`legal_documents`, com alerta de
> vencimento). Tela **Governança** no menu (abas Atas, Votações, Mandatos,
> Convênios). `vote_ballots` também é append-only com hash-chain (G7).
Atas digitais + **votação eletrônica** (quórum obrigatório, voto secreto, assinatura interna) + apuração automática + painel de mandatos/convênios.

### Etapa 8+ — Fase 2 / 3 / 4
Escalas, split automático, check-in infantil, WhatsApp em massa, OFX, patrimônio, folha, portal do contador, IA, IoT e ecossistema (ver §7).

---

## 1. Resumo executivo — os pedidos explícitos + o que as planilhas revelam

| Pedido / evidência | Situação hoje | Onde está |
|---|---|---|
| **Relatório de aniversariantes** (membros e casamentos) | ❌ Não existe endpoint nem tela | Backlog #21 · PRD Mód. 12 · Etapa 2 |
| **Registro de votação em assembleias** | ✅ Feito (Etapa 7) — atas, votação com quórum e voto secreto, apuração e painel de mandatos/convênios | Backlog P3 · PRD Mód. 6 · Etapa 7 |
| **Cadastro de usuários do sistema** | ❌ RBAC existe no banco (`users`, `roles`, `permissions`), mas não há `GET/POST/PATCH /api/v1/users` nem tela | Backlog #23 · Etapa 3 |
| **Rol de Membros** | 🟡 Parcial: 1.1, 1.3 e 1.4 feitos; faltam 1.2 (derivada), 1.5, 1.6, 1.7, 1.8 e endereço | Backlog #43–#47 · Etapa 1 |
| **Registro de Eventos e Frequência** | ❌ Módulo inteiro inexistente | Backlog #48 · Etapa 5 |
| **Relatórios financeiros do exemplo** (Demonstrativo Mensal, Entradas×Saídas, comparativo anual) | 🟡 Existem balancete/DRE; **não** existem o Demonstrativo Mensal semanal, o comparativo por período no formato do cliente nem as classificações de receita/despesa do plano de contas legado | `Docs/exemplos/*.jpeg` · Etapa 4 |

---

## 2. Rol de Membros (pessoas)

Fonte: `Docs/requisitos_basicos.txt` §1.1–§1.8. Já coberto: 1.1 (cadastro,
exceto **endereço**), 1.3 (funções/múltiplas funções) e 1.4 (mandato) —
migração `000020_cargos`.

| ID | Req. | Gap | Prior. | Modelo |
|---|---|---|---|---|
| **C1** | 1.1 | **Endereço do membro** — hoje só existe `families.address` (jsonb) | P1 | **`members.address jsonb`** (endereço é da pessoa, decisão §9) |
| **C2** | 1.2 | **Classificação no Rol** (Professos / Não Professos) | P1 | **Derivada** de `membership_status` (`CASE`) — não existe "professo inativo" (decisão §9) |
| **C3** | 1.5 | **Frequência** (Frequente / Pouco frequente / Não frequente) **com histórico** | P2 | ✅ **Feito (000028)** — `member_frequency_history`; a linha com `ended_at IS NULL` é a vigente |
| **C4** | 1.6 | **Situação do membro** — ampliar `members.membership_status` (hoje **sem CHECK**) para Ativo / Inativo / Baixado / Transferido / Falecido / Outros | P1 | `CHECK` fechando o domínio + migração dos valores legados (`extra_json.tipo_cadastro`) |
| **C5** | 1.7 | **Motivo da baixa** (+ data) | P1 | `members.exit_reason text CHECK (...)`, `members.exited_at date`, obrigatórios quando situação = baixa/transferência/falecimento |
| **C6** | 1.8 | **Histórico eclesiástico** (data, hora, tipo de evento, observação) | P1 | `member_history (id, tenant_id, branch_id, member_id, occurred_at, kind, notes, created_by, created_at)` — **append-only com hash-chain** (reaproveitar trigger de `audit_log`) |
| **C7** | §1.4 | **Alertas de vencimento de mandato** já há badge na UI; falta **relatório/painel** consolidado de mandatos vigentes/vencendo | P3 | combina com governança (§5) |

> **Ordem obrigatória:** C4 → C5 → C6. O histórico registra as transições de
> situação e de motivo; gravá-las automaticamente evita digitação dupla
> (ex.: "Profissão de fé" muda a situação e vira evento no histórico).

---

## 3. Registro de Eventos e Frequência

Fonte: `Docs/requisitos_basicos.txt` §2. **Módulo inteiro inexistente.**

| ID | Gap | Prior. | Modelo |
|---|---|---|---|
| **C8** | Cadastro de eventos (data, hora início/fim, tipo, **qtd. de participantes**, observações) | P2 | ✅ **Feito (000028)** — `church_events` + catálogo `event_kinds` customizável |
| **C9** | Tipos de evento (EBD, Culto, Culto Especial, Reunião, Pequeno Grupo, Jovens, Escola Bíblica, Santa Ceia, Vigília, Outros) | P2 | ✅ **Feito (000028)** — seed de 10 tipos |
| **C10** | **As duas opções** (decisão §9): total digitado **e** chamada nominal por pessoa | P2 | ✅ **Feito (000028)** — `participants_count` **e** `event_attendance` coexistem |
| **C11** | Painéis: Rol de Membros, Frequência e Eventos (totais por classificação/situação, entradas/baixas/transferências/falecimentos, médias de culto/EBD, maior/menor público, comparativo mês a mês e anual) | P3 | somente leitura sobre C2–C8 + relatórios existentes |

---

## 4. Relatórios que faltam (PRD Mód. 12 + planilhas do cliente)

As imagens em `Docs/exemplos/` são telas do sistema legado e revelam o
**layout esperado** dos relatórios financeiros.

| ID | Relatório | Evidência | Prior. | Observação |
|---|---|---|---|---|
| **R1** | **Aniversariantes do mês** (membros e casamentos) | PRD Mód. 12 · Backlog #21 | **P1** | `GET /api/v1/reports/birthdays?month=` + widget no dashboard; casamento depende de nova data em `member_relationships`/ficha |
| **R2** | **Dashboard demográfico**: pirâmide etária + distribuição geográfica | PRD Mód. 12 · Backlog #20 | P1 | Recharts; usa o endereço de C1 |
| **R3** | **Demonstrativo Mensal** (regime de caixa): entradas/saídas por semana, saldo inicial/final, aplicações financeiras | `10.55.30.jpeg` | P1 | ✅ **Feito (000025)** — `GET /reports/monthly-statement` + export CSV/XLSX/PDF |
| **R4** | **Entradas de Recursos × Saídas de Recursos** (pizza + barras + comparativo anual) | `11.00.34.jpeg`, `11.01.10.jpeg`, `11.01.29.jpeg` | P1 | ✅ **Feito** — página "Entradas × Saídas" (pizzas + barras + comparativo); export via DRE |
| **R5** | **Relatório de inadimplência / queda de contribuição** | PRD Mód. 12 · Backlog #19 | P1 | com sigilo por perfil |
| **R6** | **Engajamento e frequência** por ministério/célula | PRD Mód. 12 · Backlog #22 | P2 | depende de `group_attendance` (já existe) + registro de eventos |
| **R7** | **Painel de mandatos vigentes/vencendo** e **histórico de votações/atas** | PRD Mód. 12 (Governança) | P3 | depende de C7 e §5 |

### 4.1 Plano de contas do cliente (evidência `10.54.11.jpeg` / `10.55.30.jpeg`)

**Decisão (§9): importar com os mesmos códigos**, substituindo o seed atual
(`financial_classification_types` hoje tem `1.1 Dízimo`, `1.2 Oferta`, …).

**Receitas (101–111):** Dízimos, Ofertas, Oferta Missionária, Rendimentos de
Aplicações Financeiras, Venda de Equipamentos, Aluguel de Bens Imóveis, Outras
Receitas, Resgate de Aplicação Financeira, Empréstimos, Receitas Central Kids,
Receitas Diaconia. Inclui **Saldo inicial (501)** e **Saldo final (502)**.

**Despesas (1–32):** Desp. Benfeitorias, Juros, Energia Elétrica, Água e Esgoto,
Comunicação, Impostos/taxas/contribuições, Pastores (Congressos, Fundo de
Garantia, Plano de Saúde, Demais Despesas), Folha de Pagamento, Encargos
Trabalhistas/sociais s/ FOLHA, Ajuda de custo (permanente/eventual), Viagem e
locomoção, Locação de bens imóveis, Aquisição de insumos/medicamentos, Pró-labore,
Assembleia Geral, Serviços de Conservação/limpeza/vigilância, Demais serviços de
conservação, Materiais e serviços p/ reforma, Reembolsos (M. Diaconia, Min.
Depoimentos, Min. de Missões), Confraternizações/eventos, Instrumentos/músicas/
computadores/projetores, Materiais p/ ensino bíblico, Despesas com funeral.

**Ação:** migração de seed (por tenant) que replica o plano legado como
`financial_classification_types`, preservando o `code` para importação/migração,
permitindo edição pela igreja.

---

## 5. Governança e Compliance Institucional (PRD Mód. 6) — ✅ Etapa 7 (23/09/2026)

O cliente pediu **registro de votação em assembleias**. Implementado pelo pacote
`internal/governance` e pela migração `000033_governance`; a permissão
`governance.read/write` (semeada na `000009_seed_demo`) passou a ter uso.
**Decisões (§9): quórum obrigatório, voto secreto e assinatura interna.**

| ID | Gap | Prior. | Situação |
|---|---|---|---|
| **G1** | **Livro de atas digital** (pauta, deliberações, presença) | P3 | ✅ `minutes` + tela/API (CRUD, bloqueio de edição após assinar) |
| **G2** | **Votação eletrônica** com **quórum obrigatório** e **voto secreto** | P3 | ✅ `votes` + `vote_options`; participação em `vote_registrations` e escolha em `vote_ballots` (sem vínculo eleitor↔voto) |
| **G3** | **Apuração + ata automática** da votação | P3 | ✅ `POST /votes/{id}/close` apura, grava `result_summary` e anexa o resultado à ata vinculada |
| **G4** | **Assinatura eletrônica interna** da ata | P3 | ✅ `POST /minutes/{id}/sign` (hash do conteúdo + credenciais); `minute_signatures` append-only |
| **G5** | **Gestão de mandatos** consolidada + alertas de vencimento | P3 | ✅ `GET /governance/mandates` (vigentes/vencendo em 60 dias) + aba Mandatos |
| **G6** | **Convênios e documentação legal** com alertas de vencimento | P3 | ✅ `legal_documents` + alerta na tela |
| **G7** | **Trilha de auditoria imutável também na governança** | P3 | ✅ `minute_signatures` e `vote_ballots` append-only **com hash-chain** |

> Critério de aceite da Fase 3: uma assembleia real vota a diretoria 100% pelo
> sistema, com quórum validado, voto secreto, assinatura interna e ata gerada
> automaticamente.

---

## 6. Cadastro de usuários e perfis do sistema (Backlog #23)

RBAC existe no banco (`users`, `roles`, `permissions`, `role_permissions`,
seed `000009_seed_demo`), mas **não há API nem tela**.

| ID | Gap | Prior. |
|---|---|---|
| **U1** | `GET/POST/PATCH /api/v1/users` (+ `is_active`, reset de senha, troca de papel) | P1 |
| **U2** | `GET /api/v1/roles` e `GET /api/v1/permissions` (catálogo para a UI) | P1 |
| **U3** | Tela webadmin "Usuários e Acessos" (criar, editar papel, desativar, resetar senha) | P1 |
| **U4** | Vínculo de filial (escopo Sede × Filial) respeitando RLS | P1 |
| **U5** | **MFA/TOTP** aproveitando `mfa_secret`/`mfa_enabled` (hoje só colunas) | P0 (PRD RNF) |
| **U6** | Perfis faltantes no seed: `pastor_filial`, `lider`, `pastor`, `contador`, `membro`, `visitante` (PRD §3) | P1 |

---

## 7. Demais lacunas por fase (PRD e checklist de escopo)

### Fase 0 — Fundação (faltam)
- [ ] **CI/CD** (build + `go test`/`go vet` + lint Next) com `CHOSEN_TESTS_REQUIRED=1` (#2).
- [ ] Ambientes dev/staging/prod definidos.
- [ ] **S3/object storage** (#11) — anexos/fotos hoje em disco local (amarra a 1 instância).
- [ ] **MFA** (ver U5), **tracing distribuído** (OpenTelemetry, #38).
- [ ] **Seed da filial Norte + `pastor.norte@demo.local`** (#40) — documentado e inexistente.
- [ ] **Testes no frontend** (#41) e **testes de contrato da API** (#42).

### Fase 1 — MVP (faltam)
- [x] LGPD: consentimento (#3), portabilidade (#4), exclusão/anonimização (#5) — *risco legal* (feito na Etapa 6).
- [~] Secretaria: certificados (#7), cartas de transferência/recomendação (#8), transferência entre filiais (#9). *(sub-congregações #10 ✅ feitas na Fase 2)*
- [ ] Financeiro: contas a pagar × pagas (#14), centro de custo (#15), orçamento com aprovação (#16), recibo com validade de IR (#17), crowdfunding (#18).
- [ ] Relatórios da §4 (R1–R5) e **usuários do sistema** (§6).

### Fase 2 — Multi-filial (faltam)
- [x] **Escalas (`rosters`)** + confirmação de presença (#24) e **conflito de agenda** (#25) — entregues (Set/2026). Escalas inteligentes (#26) **parciais**: sugestão por ministério/frequência com marca de conflito; faltam férias/disponibilidade explícita.
- [ ] Split automático de repasses (#27) — *maior risco financeiro*.
- [x] **Painel consolidado Sede > Filiais** (#28) e **sub-congregações** (#10) — entregues (Set/2026): RLS com leitura hierárquica (`rls_read_scope`), CRUD de filiais com hierarquia e relatório consolidado por filial.
- [x] Check-in infantil real (criança↔responsável, etiqueta, restrições, totem) (#29) — entregue (Set/2026) como **módulo Kids**: trilha/lições, turmas, matrícula + responsáveis, encontros, check-in/check-out com código de segurança e etiqueta, e relatório de evolução. Falta o totem dedicado.
- [ ] Mapa de calor geográfico de células (#30).
- [x] **WhatsApp amplo** (aniversariantes, escalas, boas-vindas) (#31) e **disparo em massa segmentado** (#32) — entregues (Set/2026): `000040_whatsapp_mass`, `NotificationWorker` e segmentação em `internal/announcements/audience.go`.
- [ ] Trilha de acolhimento automatizada (#33), conciliação OFX (#34), gateway real de doações (#35).
- [ ] Banco de dons e talentos (#36), background check (#37).

### Fase 3 — Governança (parcial: Módulo 6 entregue na Etapa 7)
- [x] Governança (Módulo 6): atas + votação (quórum/secreto/assinatura) + apuração, painel de mandatos e convênios.
- [ ] Demais itens da Fase 3: **patrimônio** (`assets` com QR/depreciação), **folha de pagamento** (CLT + prebenda), **portal do contador**, **escola bíblica/discipulado**, **prontuário pastoral E2EE**.

### Fase 4 — IA/IoT/Ecossistema (0%)
- [ ] IA de evasão, Open Finance, reconhecimento facial, IoT, Chosen Network, marketplace/API pública, streaming, Mód. 7 (missões), Mód. 8 (casamentos/funerais/capelania), Mód. 11 (pânico/estacionamento), multi-idioma/moeda, SuperApp, Mód. 4 (eventos/liturgia).

---

## 8. Backlog priorizado (itens novos e reafirmados)

| # | Item | Fase | Prior. | Depende de |
|---|---|---|---|---|
| N1 | Relatório de **aniversariantes** + widget | 1 | **P1** | — |
| N2 | **Usuários e acessos** (API + tela) | 1 | **P1** | — |
| N3 | **Situação do membro** (C4) | 1 | P1 | — |
| N4 | **Motivo da baixa** (C5) | 1 | P1 | N3 |
| N5 | **Histórico eclesiástico** (C6) | 1 | P1 | N3, N4 |
| N6 | **Classificação no Rol** (C2, derivada) | 1 | P1 | N3 |
| N7 | **Endereço do membro** (C1, por pessoa) | 1 | P1 | — |
| N8 | Plano de contas do cliente (mesmos códigos) + **Demonstrativo Mensal** (R3) | 1 | P1 | — |
| N9 | **Entradas × Saídas** e comparativo anual (R4) | 1 | P1 | N8 |
| N10 | **Dashboard demográfico** (R2) | 1 | P1 | N7 |
| N11 | **Inadimplência/queda de contribuição** (R5) | 1 | P1 | — |
| N12 | **MFA/TOTP** | 0 | P0 | N2 |
| N13 | LGPD (consentimento/portabilidade/exclusão) | 1 | P0 | — |
| N14 | CI/CD | 0 | P0 | — |
| N15 | Seed Norte + usuário da filial | 0 | P0 | — |
| N16 | **Eventos** (C8/C9) | 2 | P2 | — |
| N17 | **Total digitado + chamada nominal** (C10) | 2 | P2 | N16 |
| N18 | **Frequência com histórico** (C3) | 2 | P2 | N17 (p/ automatizar) ou manual |
| N19 | ✅ **Governança: atas + votação (quórum/secreto) + apuração** (G1–G3) | 3 | P3 | — |
| N20 | ✅ Assinatura interna + mandatos + convênios (G4–G6) | 3 | P3 | N19 |
| N21 | **Painéis de Membros e Eventos** (C11, R7) | 3 | P3 | N1–N18 |

> Os itens já catalogados no `02_Backlog.md` (#7–#38) permanecem válidos; aqui
> foram reafirmados ou detalhados. Use **este** documento para a visão do
> cliente e o `02_Backlog.md` para o histórico de execução.

---

## 9. Decisões tomadas (23/09/2026)

1. **Endereço é do membro** → nova coluna `members.address jsonb` (não só `families.address`).
2. **Não existe "professo inativo"** → a classificação Professos/Não Professos é
   **derivada** da situação; **não** criar coluna própria (evita duas fontes de verdade).
3. **Plano de contas** → importar o legado **com os mesmos códigos** (receitas
   101–111, despesas 1–32, saldos 501/502), substituindo o seed atual.
4. **Votação em assembleia** → **quórum obrigatório**, **voto secreto** e
   **assinatura interna** (não ICP-Brasil nesta fase).
5. **Frequência** → o registro de eventos aceita **as duas formas**: total de
   participantes digitado **e** chamada nominal por pessoa.

---

## 10. Ordem de execução recomendada

```
Etapa 0  CI/CD (N14) → seed Norte (N15) → rodar testes RLS
 └─ Etapa 1  Membros: endereço (N7) → situação (N3) → motivo (N4)
    │        → histórico (N5) → classificação derivada (N6)
    ├─ Etapa 2  Aniversariantes (N1) → demográficos (N10)
    ├─ Etapa 3  Usuários (N2) → MFA (N12)
    ├─ Etapa 4  Plano de contas (N8) → Demonstrativo Mensal (N9)
    ├─ Etapa 5  Eventos (N16) → total+nominal (N17) → frequência (N18)
    ├─ Etapa 6  LGPD (N13)
    └─ Etapa 7  ✅ Governança: atas+votação (N19) → assinatura/mandatos (N20)
        └─ Etapa 8+  Fase 2 (escalas, split, check-in, WhatsApp, OFX) → Fase 3 → Fase 4
```

**Racional:** primeiro a **rede de segurança** (CI/CD + testes RLS) e o
**núcleo de membros**, que é dependência de quase tudo; depois as **vitórias
rápidas** que o cliente pediu (aniversariantes, usuários, financeiro do
exemplo); por fim os módulos verticais (eventos, governança) e a inovação.

---

## 11. Critérios de aceite sugeridos

- **Aniversariantes:** lista do mês corrente (e do próximo) com membros e
  casamentos, exportável e filtrável por filial.
- **Usuários:** admin cria usuário, define papel/filial e desativa; o login
  reflete as permissões imediatamente; nenhum tenant enxerga usuário de outro.
- **Votação:** assembleia cria votação com quórum obrigatório, membros votam em
  segredo, apuração fecha sozinha, ata assinada internamente sai com o resultado
  — em um caso real de eleição de diretoria.
- **Rol de Membros:** o rol fecha com totais por professos/não professos
  (derivados) e por situação, com frequência e histórico imutável por membro.
- **Eventos:** registrar EBD e Culto com total e/ou chamada nominal e ver médias,
  maior/menor público e comparativos mês a mês e anual.
- **Financeiro do exemplo:** reproduzir o Demonstrativo Mensal e o comparativo
  Entradas×Saídas nos moldes das planilhas do cliente, com o plano de contas
  legado.

---

*Documento vivo — ao concluir um item, marcá-lo aqui e refletir o avanço em
`Docs/00_Checkpoint.md` e `Docs/02_Backlog.md`.*
