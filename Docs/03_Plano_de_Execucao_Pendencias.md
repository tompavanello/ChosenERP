# Chosen ERP - Plano de Execucao e Pendencias (Gap Analysis)

**Data:** 23/09/2026
**Base de comparacao:**
- `Docs/requisitos_basicos.txt` - e-mail do cliente (22/09/2026): Rol de Membros e Registro de Eventos/Frequencia.
- `Docs/exemplos/` - planilhas do Rol de Membros e relatorios financeiros legados.
- `Docs/Chosen_ERP_Documentacao_Completa.md` (PRD) e `Docs/01_Blueprint_Arquitetura.md`.
- `Docs/00_Checkpoint.md` e `Docs/02_Backlog.md` (estado real do codigo).

**Objetivo:** consolidar, num unico lugar, **tudo que ainda falta** para avancar,
com prioridade, modelo de dados, dependencias, decisoes ja tomadas e ordem de execucao.

> Status global herdado do checkpoint: Fase 0 ~90% - Fase 1 ~55% - Fase 2 ~45% -
> Fases 3 e 4 = 0%. Este documento **nao substitui** o `02_Backlog.md`; ele
> reorganiza os gaps pela otica do que o cliente pediu + do que as planilhas
> revelam, e registra as decisoes de produto.

---

## 0. Plano de inicio - passo a passo (recomendado)

> **Decisoes ja tomadas (9):** endereco **por membro** - professorate **derivada**
> da situacao (nao existe "professo inativo") - plano de contas importado **com os
> mesmos codigos** - votacao com **quorum obrigatorio, voto secreto e assinatura
> interna** - eventos aceitam **total digitado E chamada nominal**.
>
> **Status de execucao (23/09/2026):**  Etapas **0 a 7** concluidas -
> Etapa 8+ pendente.

A ordem abaixo e a de **menor risco e maior valor** por etapa. Cada etapa e um
bloco entregavel (migracao + API + tela + teste). Nao comece a proxima antes de
fechar a anterior.

### Etapa 0 - Rede de seguranca (antes de qualquer migracao) 
> **Entregue (23/09/2026):** `.github/workflows/ci.yml` (Go build/vet/testes de
> RLS + typecheck/build do webadmin, com `CHOSEN_TESTS_REQUIRED=1`); migracao
> `000021_seed_north` (filial Norte + `pastor.norte@demo.local` + permissoes do
> `pastor_filial`); suite de RLS rodando como baseline.
1. **CI/CD minimo**: GitHub Actions com `go build ./... && go vet ./... && go test ./... -count=1`
   e `CHOSEN_TESTS_REQUIRED=1`; `npm run build` no `webadmin`.
2. **Seed da filial Norte** + usuario `pastor.norte@demo.local` (hoje documentado e inexistente).
3. Rodar `go test ./internal/store` **antes** de tocar em RLS; e a rede que valida as migracoes seguintes.
   -> *Por que primeiro: as etapas 1 em diante mexem em `members`, que tem RLS; sem isso voce altera schema sem rede.*

### Etapa 1 - Nucleo do Rol de Membros (a base de tudo) 
> **Entregue (23/09/2026):** migracao `000022_member_lifecycle`
> (`members.address` jsonb, `CHECK` de situacao, `exit_reason`/`exited_at`,
> `member_history` append-only com hash-chain e RLS, classificacao derivada);
> endpoints `GET/POST /members/{id}/history`; UI com endereco, motivo/data de
> baixa e **aba Historico**; historico automatico nas mudancas de situacao e no
> cadastro; testes novos (`TestMemberHistory_*`). O guard do append-only libera
> a exclusao em cascata quando o membro e apagado (`pg_trigger_depth() > 1`).
Uma migracao (`000021_member_lifecycle`) + API + tela:
1. `members.address jsonb` (CEP, logradouro, numero, complemento, bairro, cidade, UF) - **endereco por membro**.
2. Fechar `members.membership_status` com `CHECK` (Ativo / Inativo / Baixado / Transferido / Falecido / Outros) e migrar valores legados.
3. `members.exit_reason` (CHECK) + `members.exited_at` - obrigatorios quando situacao = baixa/transferencia/falecimento.
4. `member_history` **append-only com hash-chain** (data, hora, tipo de evento, observacao).
5. **Classificacao Professos / Nao Professos e derivada** da situacao (`CASE`), sem coluna nova.
6. UI: campos de situacao/motivo/data e a **aba Historico** no perfil do membro.
   -> *Por que agora: historico, aniversariantes, demograficos e frequencia todos dependem de `members`.*

### Etapa 2 - Aniversariantes + demograficos (vitoria rapida) 
> **Entregue (23/09/2026):** migracao `000023_member_marriage`
> (`members.marriage_date`); endpoints `GET /api/v1/reports/birthdays?month=`
> (nascimentos + aniversarios de casamento, com dedupe de casal) e
> `GET /api/v1/reports/demographics` (piramide etaria, situacao, estado civil,
> sexo e distribuicao por UF/cidade); **widget de aniversariantes** na Visao Geral
> e **painel demografico** na pagina de Relatorios.
>
> **Menu e exportacao (23/09/2026):** cada relatorio ganhou **pagina e item de
> menu proprios** no grupo **Relatorios** (Balancete mensal, DRE, Aniversariantes,
> Demograficos) e **exportacao em CSV, Excel (XLSX) e PDF** (versao de impressao
> que o navegador salva em PDF). O XLSX e gerado por `internal/xlsx` (OOXML
> minimo, sem dependencia externa); o CSV/PDF sao centralizados em
> `internal/httpapi/report_export.go`.
1. `GET /api/v1/reports/birthdays?month=` (membros e casamentos) + widget no dashboard.
2. Piramide etaria e distribuicao geografica (agora com o endereco da Etapa 1).
   -> *Barato, e e o pedido explicito do cliente; aparece rapido na tela.*

### Etapa 3 - Usuarios e Acessos 
> **Entregue (23/09/2026):** migracao `000024_users_roles_mfa` (permissoes
> `users.read/users.write`, perfis `lider`/`pastor`/`contador`/`visitante` e
> permissoes-base de cada perfil; `auth_lookup_user` devolve MFA). API:
> `GET/POST /api/v1/users`, `PATCH /api/v1/users/{id}`,
> `POST /api/v1/users/{id}/password`, `GET /api/v1/roles`,
> `GET /api/v1/permissions` (restrito a `super_admin`/`admin_sede`). **MFA/TOTP**
> (`internal/auth/totp.go`, sem dependencia externa): `GET /api/v1/auth/mfa`,
> `POST /auth/mfa/{setup,enable,disable}` e login com `code`. Tela
> **Usuarios e Acessos** no menu (criar, editar perfil/filial, ativar/desativar,
> redefinir senha) e card de MFA com QR para o proprio usuario.
1. `GET/POST/PATCH /api/v1/users` + `GET /api/v1/roles` e `/permissions`.
2. Perfis faltantes no seed (`pastor_filial`, `lider`, `pastor`, `contador`, `membro`, `visitante`).
3. Tela "Usuarios e Acessos" (criar, editar papel, desativar, resetar senha, escopo Sede x Filial).
4. **MFA/TOTP** sobre `mfa_secret`/`mfa_enabled` (PRD RNF).
   -> *Sem isso a operacao real da igreja fica no usuario unico admin.*

### Etapa 4 - Financeiro no formato do cliente 
> **Entregue (23/09/2026):** migracao `000025_legacy_chart_of_accounts` importa o
> plano de contas do legado com os **mesmos codigos** (receitas 101-111,
> despesas 1-32) em `financial_categories` e `financial_classification_types`,
> re-aponta os lancamentos do seed e desativa as contas genericas antigas.
> Endpoints `GET /api/v1/reports/monthly-statement?month=YYYY-MM` (Demonstrativo
> Mensal em regime de caixa: contas com totais por semana + saldo inicial/final)
> e `/monthly-statement/export?format=csv|xlsx|pdf`. Tela **Demonstrativo Mensal**
> e **Entradas x Saidas** (pizzas de composicao, barras mensais e comparativo com
> o periodo anterior) no grupo Relatorios.
1. Migracao que **substitui** o plano de contas atual pelos **mesmos codigos do legado** (receitas 101-111, despesas 1-32, saldos 501/502).
2. **Demonstrativo Mensal** (regime de caixa): entradas/saidas por semana, saldo inicial/final, aplicacoes financeiras.
3. **Entradas de Recursos x Saidas de Recursos** (pizza + barras + comparativo anual).
   -> *Reproduz as planilhas de `Docs/exemplos/`; e o que o cliente ja usa hoje.*

> **Ajustes do financeiro (23/09/2026):** removida a tabela **redundante**
> `financial_classification_types` (migracao `000026`) - o lancamento sempre usou
> `financial_categories` (plano de contas); a **conta contabil** virou
> **obrigatoria** no lancamento (com casamento de tipo entrada/saida) e o rotulo
> "categoria" foi trocado para **"Conta contabil"**. O formulario de lancamento
> agora aceita **anexo** (comprovante) no momento do registro, e a tela de
> lancamentos ganhou **importacao em lote** por CSV (`POST /finance/transactions/import`).
> Corrigido de quebra um bug do hash-chain (`000027`): `payment_method` nulo
> gerava hash NULL e quebrava o INSERT.

### Etapa 5 - Eventos e Frequencia 
> **Entregue (23/09/2026):** migracao `000028_church_events` (`event_kinds` com
> seed dos 10 tipos, `church_events`, `event_attendance` e
> `member_frequency_history`). API: CRUD de tipos (`/event-kinds`), CRUD de
> eventos (`/events`) com filtro por periodo/tipo, chamada nominal
> (`GET/POST /events/{id}/attendance`) e frequencia do membro
> (`GET/POST /members/{id}/frequency`). O **total digitado e a chamada nominal
> coexistem**; a frequencia guarda **historico** (fecha a vigente e insere a
> nova). Tela **Eventos** (eventos + tipos, com checklist de presenca) e aba
> **Frequencia** no perfil do membro.
>
> **Melhorias de eventos (23/09/2026, migracao `000029`):** modo de presenca
> (**chamada nominal** OU **apenas o numero**), **custo estimado** (opcional),
> **eventos multi-dia** (retiro/acampamento, com data de termino) e
> **convocados** (`event_invitees`) - pessoas e/ou **ministerios** obrigados a
> participar. Visao de **calendario** (grade mensal com eventos, inclusive os de
> varios dias).
1. `church_events` + catalogo `event_kinds` (EBD, Culto, Culto Especial, Reuniao, Pequeno Grupo, Jovens, Escola Biblica, Santa Ceia, Vigilia, Outros).
2. **Total de participantes digitado E chamada nominal** (`event_attendance`) - as duas opcoes.
3. `member_frequency_history` (Frequente / Pouco frequente / Nao frequente com historico), alimentada pela chamada quando houver.

### Etapa 6 - LGPD (risco legal) 
> **Entregue (23/09/2026):** pacote `internal/lgpd`. **Consentimento**:
> `GET/POST /api/v1/consent-terms` (versao automatica) e
> `GET/POST /api/v1/members/{id}/consents` (registro/revogacao com IP e
> user-agent). **Portabilidade**: `GET /api/v1/members/{id}/export` devolve em
> JSON todos os dados do titular (ficha, cargos, vinculos, familias, historico,
> frequencia, consentimentos, doacoes e presencas). **Anonimizacao**:
> `POST /api/v1/members/{id}/anonymize` (apaga PII e registra no historico).
> Aba **LGPD** no perfil do membro. A exclusao fisica nao e oferecida de
> proposito: o livro financeiro e retido por obrigacao fiscal - a anonimizacao
> atende ao art. 18 mantendo a integridade dos lancamentos.

### Etapa 7 - Governanca 
> **Entregue (23/09/2026):** migracao `000033_governance` com o pacote
> `internal/governance`. **Atas digitais** (`minutes` + `minute_signatures`
> append-only com **hash-chain**): CRUD, **assinatura eletronica interna** (hash
> do conteudo + credenciais do usuario; a ata e congelada ao assinar) e leitura
> das assinaturas. **Votacao eletronica** (`votes` + `vote_options` +
> `vote_registrations` + `vote_ballots`): **quorum obrigatorio**, **voto
> secreto** - a participacao fica separada da escolha, entao a apuracao so
> devolve contagens -, abertura/encerramento, **apuracao automatica** e **ata
> automatica** (o resultado e anexado a ata vinculada ao encerrar). **Painel de
> mandatos** (`GET /governance/mandates`, vigentes/vencendo em 60 dias) e
> **convenios/documentacao legal** (`legal_documents`, com alerta de
> vencimento). Tela **Governanca** no menu (abas Atas, Votacoes, Mandatos,
> Convenios). `vote_ballots` tambem e append-only com hash-chain (G7).
Atas digitais + **votacao eletronica** (quorum obrigatorio, voto secreto, assinatura interna) + apuracao automatica + painel de mandatos/convenios.

### Etapa 8+ - Fase 2 / 3 / 4
Escalas, split automatico, check-in infantil, WhatsApp em massa, OFX, patrimonio, folha, portal do contador, IA, IoT e ecossistema (ver 7).

---

## 1. Resumo executivo - os pedidos explicitos + o que as planilhas revelam

| Pedido / evidencia | Situacao hoje | Onde esta |
|---|---|---|
| **Relatorio de aniversariantes** (membros e casamentos) |  Nao existe endpoint nem tela | Backlog #21 - PRD Mod. 12 - Etapa 2 |
| **Registro de votacao em assembleias** |  Feito (Etapa 7) - atas, votacao com quorum e voto secreto, apuracao e painel de mandatos/convenios | Backlog P3 - PRD Mod. 6 - Etapa 7 |
| **Cadastro de usuarios do sistema** |  RBAC existe no banco (`users`, `roles`, `permissions`), mas nao ha `GET/POST/PATCH /api/v1/users` nem tela | Backlog #23 - Etapa 3 |
| **Rol de Membros** |  Parcial: 1.1, 1.3 e 1.4 feitos; faltam 1.2 (derivada), 1.5, 1.6, 1.7, 1.8 e endereco | Backlog #43-#47 - Etapa 1 |
| **Registro de Eventos e Frequencia** |  Modulo inteiro inexistente | Backlog #48 - Etapa 5 |
| **Relatorios financeiros do exemplo** (Demonstrativo Mensal, EntradasxSaidas, comparativo anual) |  Existem balancete/DRE; **nao** existem o Demonstrativo Mensal semanal, o comparativo por periodo no formato do cliente nem as classificacoes de receita/despesa do plano de contas legado | `Docs/exemplos/*.jpeg` - Etapa 4 |

---

## 2. Rol de Membros (pessoas)

Fonte: `Docs/requisitos_basicos.txt` 1.1-1.8. Ja coberto: 1.1 (cadastro,
exceto **endereco**), 1.3 (funcoes/multiplas funcoes) e 1.4 (mandato) -
migracao `000020_cargos`.

| ID | Req. | Gap | Prior. | Modelo |
|---|---|---|---|---|
| **C1** | 1.1 | **Endereco do membro** - hoje so existe `families.address` (jsonb) | P1 | **`members.address jsonb`** (endereco e da pessoa, decisao 9) |
| **C2** | 1.2 | **Classificacao no Rol** (Professos / Nao Professos) | P1 | **Derivada** de `membership_status` (`CASE`) - nao existe "professo inativo" (decisao 9) |
| **C3** | 1.5 | **Frequencia** (Frequente / Pouco frequente / Nao frequente) **com historico** | P2 |  **Feito (000028)** - `member_frequency_history`; a linha com `ended_at IS NULL` e a vigente |
| **C4** | 1.6 | **Situacao do membro** - ampliar `members.membership_status` (hoje **sem CHECK**) para Ativo / Inativo / Baixado / Transferido / Falecido / Outros | P1 | `CHECK` fechando o dominio + migracao dos valores legados (`extra_json.tipo_cadastro`) |
| **C5** | 1.7 | **Motivo da baixa** (+ data) | P1 | `members.exit_reason text CHECK (...)`, `members.exited_at date`, obrigatorios quando situacao = baixa/transferencia/falecimento |
| **C6** | 1.8 | **Historico eclesiastico** (data, hora, tipo de evento, observacao) | P1 | `member_history (id, tenant_id, branch_id, member_id, occurred_at, kind, notes, created_by, created_at)` - **append-only com hash-chain** (reaproveitar trigger de `audit_log`) |
| **C7** | 1.4 | **Alertas de vencimento de mandato** ja ha badge na UI; falta **relatorio/painel** consolidado de mandatos vigentes/vencendo | P3 | combina com governanca (5) |

> **Ordem obrigatoria:** C4 -> C5 -> C6. O historico registra as transicoes de
> situacao e de motivo; grava-las automaticamente evita digitacao dupla
> (ex.: "Profissao de fe" muda a situacao e vira evento no historico).

---

## 3. Registro de Eventos e Frequencia

Fonte: `Docs/requisitos_basicos.txt` 2. **Modulo inteiro inexistente.**

| ID | Gap | Prior. | Modelo |
|---|---|---|---|
| **C8** | Cadastro de eventos (data, hora inicio/fim, tipo, **qtd. de participantes**, observacoes) | P2 |  **Feito (000028)** - `church_events` + catalogo `event_kinds` customizavel |
| **C9** | Tipos de evento (EBD, Culto, Culto Especial, Reuniao, Pequeno Grupo, Jovens, Escola Biblica, Santa Ceia, Vigilia, Outros) | P2 |  **Feito (000028)** - seed de 10 tipos |
| **C10** | **As duas opcoes** (decisao 9): total digitado **e** chamada nominal por pessoa | P2 |  **Feito (000028)** - `participants_count` **e** `event_attendance` coexistem |
| **C11** | Paineis: Rol de Membros, Frequencia e Eventos (totais por classificacao/situacao, entradas/baixas/transferencias/falecimentos, medias de culto/EBD, maior/menor publico, comparativo mes a mes e anual) | P3 | somente leitura sobre C2-C8 + relatorios existentes |

---

## 4. Relatorios que faltam (PRD Mod. 12 + planilhas do cliente)

As imagens em `Docs/exemplos/` sao telas do sistema legado e revelam o
**layout esperado** dos relatorios financeiros.

| ID | Relatorio | Evidencia | Prior. | Observacao |
|---|---|---|---|---|
| **R1** | **Aniversariantes do mes** (membros e casamentos) | PRD Mod. 12 - Backlog #21 | **P1** | `GET /api/v1/reports/birthdays?month=` + widget no dashboard; casamento depende de nova data em `member_relationships`/ficha |
| **R2** | **Dashboard demografico**: piramide etaria + distribuicao geografica | PRD Mod. 12 - Backlog #20 | P1 | Recharts; usa o endereco de C1 |
| **R3** | **Demonstrativo Mensal** (regime de caixa): entradas/saidas por semana, saldo inicial/final, aplicacoes financeiras | `10.55.30.jpeg` | P1 |  **Feito (000025)** - `GET /reports/monthly-statement` + export CSV/XLSX/PDF |
| **R4** | **Entradas de Recursos x Saidas de Recursos** (pizza + barras + comparativo anual) | `11.00.34.jpeg`, `11.01.10.jpeg`, `11.01.29.jpeg` | P1 |  **Feito** - pagina "Entradas x Saidas" (pizzas + barras + comparativo); export via DRE |
| **R5** | **Relatorio de inadimplencia / queda de contribuicao** | PRD Mod. 12 - Backlog #19 | P1 | com sigilo por perfil |
| **R6** | **Engajamento e frequencia** por ministerio/celula | PRD Mod. 12 - Backlog #22 | P2 | depende de `group_attendance` (ja existe) + registro de eventos |
| **R7** | **Painel de mandatos vigentes/vencendo** e **historico de votacoes/atas** | PRD Mod. 12 (Governanca) | P3 | depende de C7 e 5 |

### 4.1 Plano de contas do cliente (evidencia `10.54.11.jpeg` / `10.55.30.jpeg`)

**Decisao (9): importar com os mesmos codigos**, substituindo o seed atual
(`financial_classification_types` hoje tem `1.1 Dizimo`, `1.2 Oferta`, ...).

**Receitas (101-111):** Dizimos, Ofertas, Oferta Missionaria, Rendimentos de
Aplicacoes Financeiras, Venda de Equipamentos, Aluguel de Bens Imoveis, Outras
Receitas, Resgate de Aplicacao Financeira, Emprestimos, Receitas Central Kids,
Receitas Diaconia. Inclui **Saldo inicial (501)** e **Saldo final (502)**.

**Despesas (1-32):** Desp. Benfeitorias, Juros, Energia Eletrica, Agua e Esgoto,
Comunicacao, Impostos/taxas/contribuicoes, Pastores (Congressos, Fundo de
Garantia, Plano de Saude, Demais Despesas), Folha de Pagamento, Encargos
Trabalhistas/sociais s/ FOLHA, Ajuda de custo (permanente/eventual), Viagem e
locomocao, Locacao de bens imoveis, Aquisicao de insumos/medicamentos, Pro-labore,
Assembleia Geral, Servicos de Conservacao/limpeza/vigilancia, Demais servicos de
conservacao, Materiais e servicos p/ reforma, Reembolsos (M. Diaconia, Min.
Depoimentos, Min. de Missoes), Confraternizacoes/eventos, Instrumentos/musicas/
computadores/projetores, Materiais p/ ensino biblico, Despesas com funeral.

**Acao:** migracao de seed (por tenant) que replica o plano legado como
`financial_classification_types`, preservando o `code` para importacao/migracao,
permitindo edicao pela igreja.

---

## 5. Governanca e Compliance Institucional (PRD Mod. 6) -  Etapa 7 (23/09/2026)

O cliente pediu **registro de votacao em assembleias**. Implementado pelo pacote
`internal/governance` e pela migracao `000033_governance`; a permissao
`governance.read/write` (semeada na `000009_seed_demo`) passou a ter uso.
**Decisoes (9): quorum obrigatorio, voto secreto e assinatura interna.**

| ID | Gap | Prior. | Situacao |
|---|---|---|---|
| **G1** | **Livro de atas digital** (pauta, deliberacoes, presenca) | P3 |  `minutes` + tela/API (CRUD, bloqueio de edicao apos assinar) |
| **G2** | **Votacao eletronica** com **quorum obrigatorio** e **voto secreto** | P3 |  `votes` + `vote_options`; participacao em `vote_registrations` e escolha em `vote_ballots` (sem vinculo eleitorvoto) |
| **G3** | **Apuracao + ata automatica** da votacao | P3 |  `POST /votes/{id}/close` apura, grava `result_summary` e anexa o resultado a ata vinculada |
| **G4** | **Assinatura eletronica interna** da ata | P3 |  `POST /minutes/{id}/sign` (hash do conteudo + credenciais); `minute_signatures` append-only |
| **G5** | **Gestao de mandatos** consolidada + alertas de vencimento | P3 |  `GET /governance/mandates` (vigentes/vencendo em 60 dias) + aba Mandatos |
| **G6** | **Convenios e documentacao legal** com alertas de vencimento | P3 |  `legal_documents` + alerta na tela |
| **G7** | **Trilha de auditoria imutavel tambem na governanca** | P3 |  `minute_signatures` e `vote_ballots` append-only **com hash-chain** |

> Criterio de aceite da Fase 3: uma assembleia real vota a diretoria 100% pelo
> sistema, com quorum validado, voto secreto, assinatura interna e ata gerada
> automaticamente.

---

## 6. Cadastro de usuarios e perfis do sistema (Backlog #23)

RBAC existe no banco (`users`, `roles`, `permissions`, `role_permissions`,
seed `000009_seed_demo`), mas **nao ha API nem tela**.

| ID | Gap | Prior. |
|---|---|---|
| **U1** | `GET/POST/PATCH /api/v1/users` (+ `is_active`, reset de senha, troca de papel) | P1 |
| **U2** | `GET /api/v1/roles` e `GET /api/v1/permissions` (catalogo para a UI) | P1 |
| **U3** | Tela webadmin "Usuarios e Acessos" (criar, editar papel, desativar, resetar senha) | P1 |
| **U4** | Vinculo de filial (escopo Sede x Filial) respeitando RLS | P1 |
| **U5** | **MFA/TOTP** aproveitando `mfa_secret`/`mfa_enabled` (hoje so colunas) | P0 (PRD RNF) |
| **U6** | Perfis faltantes no seed: `pastor_filial`, `lider`, `pastor`, `contador`, `membro`, `visitante` (PRD 3) | P1 |

---

## 7. Demais lacunas por fase (PRD e checklist de escopo)

### Fase 0 - Fundacao (faltam)
- [ ] **CI/CD** (build + `go test`/`go vet` + lint Next) com `CHOSEN_TESTS_REQUIRED=1` (#2).
- [ ] Ambientes dev/staging/prod definidos.
- [ ] **S3/object storage** (#11) - anexos/fotos hoje em disco local (amarra a 1 instancia).
- [ ] **MFA** (ver U5), **tracing distribuido** (OpenTelemetry, #38).
- [ ] **Seed da filial Norte + `pastor.norte@demo.local`** (#40) - documentado e inexistente.
- [ ] **Testes no frontend** (#41) e **testes de contrato da API** (#42).

### Fase 1 - MVP (faltam)
- [x] LGPD: consentimento (#3), portabilidade (#4), exclusao/anonimizacao (#5) - *risco legal* (feito na Etapa 6).
- [~] Secretaria: certificados (#7), cartas de transferencia/recomendacao (#8), transferencia entre filiais (#9). *(sub-congregacoes #10  feitas na Fase 2)*
- [ ] Financeiro: contas a pagar x pagas (#14), centro de custo (#15), orcamento com aprovacao (#16), recibo com validade de IR (#17), crowdfunding (#18).
- [ ] Relatorios da 4 (R1-R5) e **usuarios do sistema** (6).

### Fase 2 - Multi-filial (faltam)
- [x] **Escalas (`rosters`)** + confirmacao de presenca (#24) e **conflito de agenda** (#25) - entregues (Set/2026). Escalas inteligentes (#26) **parciais**: sugestao por ministerio/frequencia com marca de conflito; faltam ferias/disponibilidade explicita.
- [ ] Split automatico de repasses (#27) - *maior risco financeiro*.
- [x] **Painel consolidado Sede > Filiais** (#28) e **sub-congregacoes** (#10) - entregues (Set/2026): RLS com leitura hierarquica (`rls_read_scope`), CRUD de filiais com hierarquia e relatorio consolidado por filial.
- [x] Check-in infantil real (criancaresponsavel, etiqueta, restricoes, totem) (#29) - entregue (Set/2026) como **modulo Kids**: trilha/licoes, turmas, matricula + responsaveis, encontros, check-in/check-out com codigo de seguranca e etiqueta, e relatorio de evolucao. Falta o totem dedicado.
- [ ] Mapa de calor geografico de celulas (#30).
- [x] **WhatsApp amplo** (aniversariantes, escalas, boas-vindas) (#31) e **disparo em massa segmentado** (#32) - entregues (Set/2026): `000040_whatsapp_mass`, `NotificationWorker` e segmentacao em `internal/announcements/audience.go`.
- [ ] Trilha de acolhimento automatizada (#33), conciliacao OFX (#34), gateway real de doacoes (#35).
- [ ] Banco de dons e talentos (#36), background check (#37).

### Fase 3 - Governanca (parcial: Modulo 6 entregue na Etapa 7)
- [x] Governanca (Modulo 6): atas + votacao (quorum/secreto/assinatura) + apuracao, painel de mandatos e convenios.
- [ ] Demais itens da Fase 3: **patrimonio** (`assets` com QR/depreciacao), **folha de pagamento** (CLT + prebenda), **portal do contador**, **escola biblica/discipulado**, **prontuario pastoral E2EE**.

### Fase 4 - IA/IoT/Ecossistema (0%)
- [ ] IA de evasao, Open Finance, reconhecimento facial, IoT, Chosen Network, marketplace/API publica, streaming, Mod. 7 (missoes), Mod. 8 (casamentos/funerais/capelania), Mod. 11 (panico/estacionamento), multi-idioma/moeda, SuperApp, Mod. 4 (eventos/liturgia).

---

## 8. Backlog priorizado (itens novos e reafirmados)

| # | Item | Fase | Prior. | Depende de |
|---|---|---|---|---|
| N1 | Relatorio de **aniversariantes** + widget | 1 | **P1** | - |
| N2 | **Usuarios e acessos** (API + tela) | 1 | **P1** | - |
| N3 | **Situacao do membro** (C4) | 1 | P1 | - |
| N4 | **Motivo da baixa** (C5) | 1 | P1 | N3 |
| N5 | **Historico eclesiastico** (C6) | 1 | P1 | N3, N4 |
| N6 | **Classificacao no Rol** (C2, derivada) | 1 | P1 | N3 |
| N7 | **Endereco do membro** (C1, por pessoa) | 1 | P1 | - |
| N8 | Plano de contas do cliente (mesmos codigos) + **Demonstrativo Mensal** (R3) | 1 | P1 | - |
| N9 | **Entradas x Saidas** e comparativo anual (R4) | 1 | P1 | N8 |
| N10 | **Dashboard demografico** (R2) | 1 | P1 | N7 |
| N11 | **Inadimplencia/queda de contribuicao** (R5) | 1 | P1 | - |
| N12 | **MFA/TOTP** | 0 | P0 | N2 |
| N13 | LGPD (consentimento/portabilidade/exclusao) | 1 | P0 | - |
| N14 | CI/CD | 0 | P0 | - |
| N15 | Seed Norte + usuario da filial | 0 | P0 | - |
| N16 | **Eventos** (C8/C9) | 2 | P2 | - |
| N17 | **Total digitado + chamada nominal** (C10) | 2 | P2 | N16 |
| N18 | **Frequencia com historico** (C3) | 2 | P2 | N17 (p/ automatizar) ou manual |
| N19 |  **Governanca: atas + votacao (quorum/secreto) + apuracao** (G1-G3) | 3 | P3 | - |
| N20 |  Assinatura interna + mandatos + convenios (G4-G6) | 3 | P3 | N19 |
| N21 | **Paineis de Membros e Eventos** (C11, R7) | 3 | P3 | N1-N18 |

> Os itens ja catalogados no `02_Backlog.md` (#7-#38) permanecem validos; aqui
> foram reafirmados ou detalhados. Use **este** documento para a visao do
> cliente e o `02_Backlog.md` para o historico de execucao.

---

## 9. Decisoes tomadas (23/09/2026)

1. **Endereco e do membro** -> nova coluna `members.address jsonb` (nao so `families.address`).
2. **Nao existe "professo inativo"** -> a classificacao Professos/Nao Professos e
   **derivada** da situacao; **nao** criar coluna propria (evita duas fontes de verdade).
3. **Plano de contas** -> importar o legado **com os mesmos codigos** (receitas
   101-111, despesas 1-32, saldos 501/502), substituindo o seed atual.
4. **Votacao em assembleia** -> **quorum obrigatorio**, **voto secreto** e
   **assinatura interna** (nao ICP-Brasil nesta fase).
5. **Frequencia** -> o registro de eventos aceita **as duas formas**: total de
   participantes digitado **e** chamada nominal por pessoa.

---

## 10. Ordem de execucao recomendada

```
Etapa 0  CI/CD (N14) -> seed Norte (N15) -> rodar testes RLS
  Etapa 1  Membros: endereco (N7) -> situacao (N3) -> motivo (N4)
            -> historico (N5) -> classificacao derivada (N6)
     Etapa 2  Aniversariantes (N1) -> demograficos (N10)
     Etapa 3  Usuarios (N2) -> MFA (N12)
     Etapa 4  Plano de contas (N8) -> Demonstrativo Mensal (N9)
     Etapa 5  Eventos (N16) -> total+nominal (N17) -> frequencia (N18)
     Etapa 6  LGPD (N13)
     Etapa 7   Governanca: atas+votacao (N19) -> assinatura/mandatos (N20)
         Etapa 8+  Fase 2 (escalas, split, check-in, WhatsApp, OFX) -> Fase 3 -> Fase 4
```

**Racional:** primeiro a **rede de seguranca** (CI/CD + testes RLS) e o
**nucleo de membros**, que e dependencia de quase tudo; depois as **vitorias
rapidas** que o cliente pediu (aniversariantes, usuarios, financeiro do
exemplo); por fim os modulos verticais (eventos, governanca) e a inovacao.

---

## 11. Criterios de aceite sugeridos

- **Aniversariantes:** lista do mes corrente (e do proximo) com membros e
  casamentos, exportavel e filtravel por filial.
- **Usuarios:** admin cria usuario, define papel/filial e desativa; o login
  reflete as permissoes imediatamente; nenhum tenant enxerga usuario de outro.
- **Votacao:** assembleia cria votacao com quorum obrigatorio, membros votam em
  segredo, apuracao fecha sozinha, ata assinada internamente sai com o resultado
  - em um caso real de eleicao de diretoria.
- **Rol de Membros:** o rol fecha com totais por professos/nao professos
  (derivados) e por situacao, com frequencia e historico imutavel por membro.
- **Eventos:** registrar EBD e Culto com total e/ou chamada nominal e ver medias,
  maior/menor publico e comparativos mes a mes e anual.
- **Financeiro do exemplo:** reproduzir o Demonstrativo Mensal e o comparativo
  EntradasxSaidas nos moldes das planilhas do cliente, com o plano de contas
  legado.

---

*Documento vivo - ao concluir um item, marca-lo aqui e refletir o avanco em
`Docs/00_Checkpoint.md` e `Docs/02_Backlog.md`.*
