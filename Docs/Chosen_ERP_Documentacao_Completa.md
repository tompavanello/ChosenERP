# Chosen ERP
## Documentacao Completa - PRD, Especificacao de Funcionalidades e Roadmap de Desenvolvimento

**Versao:** 1.0
**Data:** Setembro/2026
**Dominio:** chosenerp.com.br

---

## Sumario

1. Sumario Executivo
2. Visao de Produto (PRD)
3. Personas e Perfis de Acesso
4. Arquitetura Tecnica
5. Modelo de Dados (Multi-tenant)
6. Especificacao Completa de Funcionalidades (por modulo)
7. Requisitos Nao Funcionais
8. Metricas de Sucesso (KPIs)
9. Riscos e Mitigacao
10. Roadmap e Checkpoints de Desenvolvimento (Fases 0 a 4)
11. Criterios de Aceite por Fase
12. Glossario

---

## 1. Sumario Executivo

O **Chosen ERP** e uma plataforma SaaS multi-tenant de gestao eclesiastica que unifica administracao de secretaria, controle financeiro de alta performance, gestao de ministerios/voluntariado, discipulado, governanca institucional e comunicacao digital em um unico ecossistema.

O produto se diferencia da concorrencia nacional (BeChurch, Eklesia, Cathedral, GIgreja) e internacional (ChMeetings, Planning Center) por tres pilares:

- **Profundidade financeira de nivel ERP corporativo** (repasses automaticos entre filiais, Open Finance, folha de pagamento, auditoria imutavel).
- **Inteligencia de dados** (predicao de evasao de membros, mapas de calor de celulas, trilhas de discipulado gamificadas).
- **Governanca institucional completa** (atas digitais, votacao eletronica, gestao de mandatos, portal do contador) - area hoje inexplorada pelos concorrentes.

Stack: **Go** (servicos criticos de auth/financeiro), **Python/FastAPI + Celery** (servicos de dados, relatorios, integracoes), **PostgreSQL** (com Row-Level Security para isolamento multi-tenant), **Redis + RabbitMQ** (cache e mensageria).

---

## 2. Visao de Produto (PRD)

### 2.1 Problema

Igrejas de medio e grande porte hoje utilizam sistemas fragmentados: uma planilha para dizimos, um app de terceiros para escalas, WhatsApp manual para comunicacao, e nenhuma ferramenta cobre governanca (atas, mandatos, compliance fiscal) ou da visibilidade preditiva sobre saude da congregacao (risco de evasao, engajamento).

### 2.2 Proposta de Valor

> "O unico sistema que cuida da alma da igreja (pastoral, discipulado, familia) e do corpo administrativo (financas, governanca, compliance) em uma unica plataforma - com inteligencia que antecipa problemas antes que eles acontecam."

### 2.3 Publico-Alvo

| Segmento | Perfil | Necessidade Principal |
|---|---|---|
| Igrejas locais (100-500 membros) | Independentes ou pequenas redes | Substituir planilhas, organizar secretaria |
| Redes/Denominacoes (500-5.000 membros, multiplas filiais) | Matriz + congregacoes | Repasses financeiros, visao consolidada, padronizacao |
| Grandes Convencoes (5.000+ membros, dezenas de filiais) | Estrutura corporativa | Compliance, auditoria, folha de pagamento, governanca estatutaria |
| Ministerios especializados | Infantil, louvor, missoes | Modulos verticais (check-in, escalas, sustento missionario) |

### 2.4 Objetivos do Produto

1. Centralizar dados demograficos, financeiros, espirituais e de engajamento da igreja.
2. Reduzir em >=70% o trabalho manual da secretaria (emissao de documentos, certificados, carteirinhas).
3. Garantir transparencia e compliance financeiro/juridico (livro-caixa, repasses, atas, auditoria).
4. Antecipar problemas pastorais via dados (evasao, queda de engajamento).
5. Entregar a melhor experiencia de membro do mercado via SuperApp.

### 2.5 Fora de Escopo (explicitamente nao cobre no MVP)

- Contabilidade fiscal completa (o sistema gera dados para o contador, nao substitui um sistema contabil homologado).
- Emissao de nota fiscal de produtos (nao e ERP de comercio).
- Gestao de escolas/colegios confessionais (pode ser modulo futuro separado).

---

## 3. Personas e Perfis de Acesso (RBAC)

| Perfil | Acesso |
|---|---|
| **Super Admin (Chosen)** | Configuracao da plataforma, suporte a tenants |
| **Admin da Sede** | Visao consolidada de todas as filiais, financeiro global, configuracoes institucionais |
| **Admin/Pastor da Filial** | Dados da sua congregacao: membros, financeiro local, escalas |
| **Tesoureiro** | Modulo financeiro completo da sua unidade, sem acesso a prontuario pastoral |
| **Secretario(a)** | Cadastros, emissao de documentos, certificados |
| **Lider de Ministerio/Celula** | Escalas do seu grupo, frequencia, membros do seu grupo |
| **Pastor/Conselheiro** | Prontuario pastoral criptografado (E2EE), aconselhamento |
| **Contador Externo** | Portal restrito: exportacao fiscal, sem dados pastorais |
| **Membro (App)** | Autoatendimento: doacoes, carteirinha, eventos, devocional |
| **Visitante** | Cadastro simplificado, trilha de acolhimento |

---

## 4. Arquitetura Tecnica

### 4.1 Stack

| Camada | Tecnologia | Funcao |
|---|---|---|
| Banco de dados | PostgreSQL 16+ | ACID, JSONB para formularios dinamicos, RLS para isolamento multi-tenant |
| Backend critico | Go (Golang) | Auth/RBAC, motor financeiro, API Gateway, WebSockets |
| Backend de dados | Python (FastAPI + Celery) | Relatorios, PDFs, integracoes externas, IA/ML |
| Cache | Redis | Sessoes, consultas frequentes |
| Mensageria | RabbitMQ | Processamento assincrono (disparo de mensagens, calculo de repasses) |
| Storage | S3 (ou equivalente) | Fotos, comprovantes, documentos |
| Observabilidade | Prometheus + Grafana + OpenTelemetry | Monitoramento, tracing distribuido |

### 4.2 Microsservicos

1. **Auth & Tenant Service (Go)** - login, JWT, RBAC, resolucao de tenant/filial.
2. **Ledger & Finance Service (Go)** - transacoes financeiras, repasses, conciliacao, folha de pagamento.
3. **Member & Roster Service (Python)** - CRUD de membros, genealogia, celulas, discipulado.
4. **Communication & Worker Service (Python/Celery)** - filas de notificacao, geracao de relatorios pesados.
5. **Governance Service (Go)** - atas, votacoes, mandatos, trilha de auditoria.
6. **Integration Hub (Python)** - Open Finance, WhatsApp Business API, gateways de pagamento, IoT.

### 4.3 Padroes de Integracao

- API REST + gRPC interno entre servicos.
- Webhooks para integracoes externas (gateways de pagamento, WhatsApp).
- API publica documentada (OpenAPI) para o futuro Marketplace de Integracoes.

---

## 5. Modelo de Dados (Multi-tenant)

Estrategia: **banco compartilhado com isolamento logico via Row-Level Security (RLS)**.

### 5.1 Tabelas-nucleo (principais)

- `tenants` - sede/convencao
- `branches` - filiais/congregacoes (FK: tenant_id)
- `users` - usuarios administrativos (FK: branch_id, role)
- `members` - membros (FK: branch_id)
- `families` - nucleos familiares
- `member_relationships` - vinculos (conjuge, filho, pai, discipulador)
- `visitors` - visitantes e funil de acolhimento
- `benefactors` - benfeitores externos
- `ministries` - ministerios/departamentos
- `ministry_members` - vinculo membro-ministerio (papel: lider/voluntario)
- `small_groups` - celulas/pequenos grupos (com geolocalizacao)
- `group_attendance` - frequencia em celulas
- `financial_transactions` - lancamentos (append-only, imutavel)
- `financial_categories` - plano de contas
- `transfers` - repasses entre filiais
- `assets` - patrimonio (com QR code)
- `payroll` - folha de pagamento (CLT + prebenda)
- `events` - eventos e cultos
- `service_liturgy` - ordem de culto
- `rosters` - escalas de voluntarios
- `checkins` - check-in infantil
- `minutes` (atas) - governanca
- `votes` - votacoes eletronicas
- `audit_log` - trilha de auditoria imutavel (hash-chained)
- `pastoral_records` - prontuario pastoral (criptografado E2EE)
- `documents` - certificados, cartas, carteirinhas gerados

### 5.2 Seguranca

- RLS garante que toda query filtra automaticamente por `branch_id`, exceto para perfis com escopo "Sede".
- `pastoral_records` usa criptografia de ponta a ponta - chave de decriptacao vinculada ao usuario pastor, nao recuperavel por admin de sistema.
- `financial_transactions` e `audit_log` sao **append-only** (sem UPDATE/DELETE), com hash encadeado para garantir integridade (estilo blockchain simplificado).

---

## 6. Especificacao Completa de Funcionalidades

### MODULO 1 - Gestao de Pessoas e Secretaria Digital

**1.1 Cadastro de Membros**
- Dados pessoais completos (nome, CPF/RG, nascimento, estado civil, profissao, contato, foto).
- Dados eclesiasticos: status (ativo/inativo/visitante/transferido/falecido), datas de batismo/adesao, cargo/oficio.
- Perfil 360o: linha do tempo completa (visita -> conversao -> batismo -> cursos -> doacoes -> voluntariado).
- Historico de participacao e anotacoes pastorais com controle de acesso granular.

**1.2 Gestao de Familias e Genealogia**
- Vinculacao em nucleos familiares (chefe, conjuge, filhos, dependentes).
- Arvore genealogica visual e interativa (parentesco).
- **Arvore de discipulado espiritual** (quem evangelizou/batizou quem - diferencial exclusivo).

**1.3 Benfeitores e Visitantes**
- Cadastro rapido de benfeitores sem vinculo de membresia.
- Cadastro de visitantes com **Trilha de Maturidade / Onboarding Gamificado**: boas-vindas automatica -> convite para "Cafe com o Pastor" -> curso de principios -> integracao em celula.

**1.4 Secretaria Digital**
- Emissao automatica de carteirinha (fisica e digital via QR Code).
- Cartas de transferencia e recomendacao.
- Certificados (batismo, casamento, apresentacao de bebes).
- Gestao multi-igrejas: hierarquia Sede > Congregacoes > Sub-congregacoes.
- Transferencia digital de cadastro entre filiais.

**1.5 Compliance de Dados (LGPD)**
- Termo de consentimento digital para dados sensiveis (religiosos, imagem).
- Exportacao total de dados do titular (portabilidade).
- Anonimizacao/exclusao sob solicitacao.

---

### MODULO 2 - Financeiro e Arrecadacao

**2.1 Entradas**
- Dizimos: registro individualizado, opcao de anonimato.
- Ofertas/doacoes: por culto, evento, campanha ou doador.
- Formas de pagamento: PIX, boleto, cartao, especie.
- **Doacoes recorrentes** (debito automatico mensal via cartao/PIX).
- **Campanha de crowdfunding** para obras/construcao com pagina publica e termometro de meta.
- Recibos digitais automaticos (e-mail/WhatsApp), incluindo **recibo com validade juridica para deducao de IR** quando aplicavel.

**2.2 Saidas**
- Contas a pagar/pagas com plano de contas (manutencao, acao social, eventos, salarios, midia, utilidades).
- Anexo de comprovantes digitais.
- Centro de custo por ministerio/celula.
- **Orcamento inteligente (budgeting)**: teto de gastos por ministerio, com workflow de aprovacao do tesoureiro.

**2.3 Repasses e Multi-filial**
- **Motor de repasses dinamico (split de pagamento)**: regras configuraveis (ex.: 10% filial -> sede, 5% missoes, 2% convencao estadual), executado automaticamente.

**2.4 Conciliacao e Compliance**
- Conciliacao bancaria via importacao OFX.
- **Conciliacao via Open Finance** (integracao direta com Banco Central - categorizacao em tempo real, sem upload manual).
- Fluxo de caixa (realizado vs. previsto).
- **Folha de pagamento**: CLT (funcionarios) + prebenda/pro-labore (pastores), com regimes tributarios distintos.
- **Trilha de auditoria imutavel**: log com hash de integridade de toda alteracao financeira.
- **Portal do Contador**: acesso externo restrito para exportacao fiscal sem acesso a dados pastorais.

**2.5 Patrimonio**
- Cadastro de bens (instrumentos, equipamentos, veiculos, imoveis).
- Etiquetas QR Code para inventario fisico.
- Calculo automatico de depreciacao.
- Historico de manutencao.

**2.6 Hardware Integrado**
- Totens e maquinas POS Chosen: doador digita CPF, passa cartao, recibo cai automaticamente no app.

---

### MODULO 3 - Ministerios, Voluntariado e Pequenos Grupos

**3.1 Ministerios**
- Cadastro de ministerios/departamentos (louvor, infantil, acao social, recepcao, ensino, midia).
- Atribuicao de lideres, coordenadores, voluntarios.
- **Banco de dons e talentos**: teste de dons espirituais + habilidades profissionais para escalar por competencia.

**3.2 Escalas**
- Criacao de escalas com notificacao e confirmacao de presenca.
- **Escalas inteligentes**: cruzamento de disponibilidade, ferias e regras internas (ex.: bloquear escala de quem nao atingiu frequencia minima na celula).
- Deteccao automatica de conflitos de agenda.

**3.3 Celulas / Pequenos Grupos / EBD**
- Mapeamento por regiao/bairro, controle de frequencia, registro de estudos.
- **Mapa de calor geografico**: identifica areas com concentracao de membros sem celula e sugere abertura de novos grupos.
- Geolocalizacao para o membro encontrar a celula mais proxima.

**3.4 Verificacao de Voluntarios**
- **Background check** integrado para voluntarios de ministerio infantil (obrigatorio em varias denominacoes).

---

### MODULO 4 - Cultos, Eventos e Midias

**4.1 Planejamento de Culto**
- Ordem de culto colaborativa (liturgia, musicas, tempo de pregacao, cronometro).
- Integracao com softwares de projecao (Hollyrics/ProPresenter via API).
- Repositorio de midia: cifras, partituras, videos.

**4.2 Check-in Infantil**
- Check-in via tablet/totem com impressao de etiquetas de seguranca.
- **Reconhecimento facial** para reduzir filas: reconhece o nucleo familiar e imprime etiqueta com restricoes alimentares automaticamente.

**4.3 Eventos**
- Venda/reserva de vagas para retiros, conferencias, jantares.
- Mapa de assentos/camas em alojamentos de retiro.

**4.4 Espacos e Reservas**
- Controle de salas, veiculos, equipamentos - evita conflitos de reserva entre ministerios.

**4.5 Streaming**
- Plataforma de transmissao ao vivo integrada ao app (sem anuncios de terceiros), com botoes de "Pedido de Oracao" e "Ofertar" em tempo real durante a live.

---

### MODULO 5 - Discipulado e Educacao

- Escola Biblica/Seminario interno (EAD + presencial), com trilhas certificadas e pre-requisitos entre modulos.
- Biblioteca fisica/digital com controle de emprestimos e liberacao de e-books por nivel de discipulado.
- Devocional diario no app com plano de leitura biblica gamificado (badges de constancia).

---

### MODULO 6 - Governanca e Compliance Institucional *(diferencial exclusivo)*

- **Livro de atas digital** com assinatura eletronica valida juridicamente.
- **Votacao eletronica** para assembleias (eleicao de diretoria, aprovacao de orcamento, mocoes), com apuracao e ata automaticas.
- **Gestao de mandatos**: alertas de vencimento de mandato de cargos estatutarios.
- **Gestao de convenios e documentacao legal**: escrituras, alvaras, contratos, seguros, com alertas de vencimento.

---

### MODULO 7 - Missoes e Expansao

- Cadastro de missionarios enviados (nacionais/internacionais).
- Controle de sustento financeiro mensal e prestacao de contas do campo.
- Painel de visao de expansao: metas de plantacao de igrejas vinculadas a indicadores (discipulos, celulas maduras).

---

### MODULO 8 - Ciclo de Vida e Cerimonias

- **Casamentos**: checklist de sessoes de aconselhamento pre-nupcial, agenda de cerimonia, certidao.
- **Funerais e cuidado ao luto**: registro, follow-up pastoral automatico em 30/60/90 dias.
- **Capelania**: agenda de visitas hospitalares/prisionais, relatorios de atendimento.

---

### MODULO 9 - Comunicacao e SuperApp

- **SuperApp White-Label**: Biblia nativa, hinario, carteirinha digital dinamica, devocional gamificado, inscricao/pagamento de eventos.
- **Central de comunicacao omnichannel**: push, e-mail, SMS, WhatsApp oficial - segmentado por filtro (ex.: so homens casados da filial Centro).
- **Integracao nativa WhatsApp**: aniversariantes, lembretes de escala, boas-vindas a visitantes.
- Mural de avisos e notificacoes push segmentadas.

---

### MODULO 10 - Inteligencia de Dados e Inovacao

- **Motor de IA para predicao de evasao**: cruza check-in, leitura, frequencia em celula e contribuicao; alerta o pastor com sugestao de acao.
- **Smart Facilities (IoT)**: liga ar-condicionado e destrava porta automaticamente antes de eventos agendados.
- **Chosen Network**: marketplace interno de servicos entre membros (economia da comunidade).
- **Marketplace de API publica**: permite integracoes de terceiros (contabilidade, som, projecao).

---

### MODULO 11 - Seguranca Fisica e Operacional

- **Botao de panico** integrado ao app para emergencias durante cultos.
- Gestao de estacionamento com sinalizacao de lotacao em tempo real.

---

### MODULO 12 - Relatorios e Dashboards de Gestao

**Financeiros**
- DRE eclesiastico, balancete mensal/anual.
- Comparativo por periodo (dizimos vs. ofertas vs. despesas).
- Relatorio de inadimplencia/queda de contribuicao (com sigilo preservado).

**Demograficos e Administrativos**
- Dashboard de crescimento (retencao, novos membros, batismos).
- Piramide etaria, mapa de distribuicao geografica.
- Aniversariantes do mes (membros e casamentos).
- Engajamento e frequencia por ministerio/celula.

**Governanca**
- Painel de mandatos vigentes/vencendo.
- Historico de votacoes e atas.

---

## 7. Requisitos Nao Funcionais

| Categoria | Requisito |
|---|---|
| Arquitetura | Multi-tenant com isolamento via RLS no PostgreSQL |
| Seguranca | RBAC granular, criptografia E2EE para prontuario pastoral, MFA para admins |
| Compliance | LGPD nativo (consentimento, portabilidade, exclusao) |
| Disponibilidade | SLA 99,9%, backups diarios, auto-scaling para picos de domingo |
| Performance | API p95 < 300ms para operacoes de leitura; motor financeiro com garantia ACID |
| Auditoria | Logs imutaveis com hash-chain para transacoes financeiras e governanca |
| Portabilidade | Exportacao total de dados do tenant a qualquer momento |
| Internacionalizacao | Suporte a multi-idioma e multi-moeda (fase 3+) |

---

## 8. Metricas de Sucesso (KPIs)

- **Adocao (MAU)**: % de membros ativos usando o app mensalmente.
- **Engajamento de voluntarios**: reducao de ausencias nao avisadas nas escalas.
- **Eficiencia financeira**: % de doacoes digitais (PIX/cartao/recorrente) vs. especie.
- **Retencao de visitantes**: taxa de conversao visitante -> membro apos trilha de acolhimento.
- **Precisao preditiva**: acuracia do modelo de predicao de evasao (validado contra evasao real).
- **Tempo de secretaria**: reducao media de horas/semana gastas em tarefas manuais.

---

## 9. Riscos e Mitigacao

| Risco | Impacto | Mitigacao |
|---|---|---|
| Vazamento de dados pastorais sensiveis | Alto | Criptografia E2EE, RLS, auditoria de acesso |
| Erro em calculo de repasses financeiros | Alto | Testes automatizados extensivos, transacoes ACID, dupla checagem em producao |
| Baixa adocao pelo membro comum (app) | Medio | UX simplificado, gamificacao, treinamento de onboarding |
| Dependencia de APIs externas (Open Finance, WhatsApp) | Medio | Circuit breakers, fallback manual (OFX/mensagem manual) |
| Complexidade de customizacao por denominacao | Medio | Motor de campos dinamicos (JSONB) desde o MVP |
| Resistencia de liderancas tradicionais a mudanca | Medio | Fase de piloto com igrejas parceiras, suporte de migracao de dados |

---

## 10. Roadmap e Checkpoints de Desenvolvimento

### FASE 0 - Fundacao Tecnica (Semanas 1-6)
**Objetivo:** Infraestrutura pronta para receber os primeiros modulos.

- [ ] Setup de repositorios, CI/CD, ambientes (dev/staging/prod)
- [ ] Provisionamento PostgreSQL com estrategia RLS definida e testada
- [ ] Auth & Tenant Service (Go): login, JWT, RBAC basico
- [ ] Estrutura multi-tenant validada com tenant + branch fake de teste
- [ ] Observabilidade (logs, metricas, tracing) configurada
- [ ] Object storage (S3) configurado

**Checkpoint de saida:** Login funcional, criacao de tenant/filial, isolamento de dados comprovado por teste automatizado de RLS.

---

### FASE 1 - MVP: Secretaria + Financeiro Basico (Semanas 7-16)
**Objetivo:** Substituir planilhas - produto vendavel para igrejas locais.

- [ ] Modulo 1: cadastro de membros, familias, arvore genealogica basica
- [ ] Modulo 1: cadastro de visitantes e benfeitores
- [ ] Modulo 2: lancamento de dizimos, ofertas, despesas (manual)
- [ ] Modulo 2: plano de contas e centro de custo
- [ ] Modulo 2: recibos digitais automaticos
- [ ] Modulo 12: relatorios financeiros basicos (balancete, DRE simplificado)
- [ ] Emissao de carteirinha digital (QR Code)
- [ ] App do membro - versao basica (ver carteirinha, ver avisos)

**Checkpoint de saida:** Uma igreja piloto opera 30 dias consecutivos so com o Chosen ERP para secretaria e financeiro, sem planilha paralela.

---

### FASE 2 - Multi-filial + Ministerios + Comunicacao (Semanas 17-28)
**Objetivo:** Atender redes de igrejas e cobrir voluntariado/comunicacao.

- [ ] Hierarquia Sede > Filiais, com visao consolidada
- [ ] Motor de repasses financeiros (split de pagamento)
- [ ] Modulo 3: ministerios, escalas, celulas com frequencia
- [ ] Modulo 4: check-in infantil (com etiqueta, sem reconhecimento facial ainda)
- [ ] Modulo 9: integracao WhatsApp (aniversariantes, lembretes de escala, boas-vindas)
- [ ] Trilha de acolhimento automatizada (onboarding de visitantes)
- [ ] Conciliacao bancaria via OFX
- [ ] Doacoes recorrentes (debito automatico)

**Checkpoint de saida:** Uma rede com ao menos 3 filiais opera com repasse automatico validado por auditoria financeira, e uma igreja usa check-in infantil em culto real.

---

### FASE 3 - Governanca + Patrimonio + Discipulado (Semanas 29-40)
**Objetivo:** Diferenciacao competitiva - modulos que nenhum concorrente tem.

- [ ] Modulo 6: atas digitais com assinatura eletronica
- [ ] Modulo 6: votacao eletronica para assembleias
- [ ] Modulo 6: gestao de mandatos e documentacao legal
- [ ] Modulo 2: gestao de patrimonio com QR Code e depreciacao
- [ ] Modulo 2: folha de pagamento (CLT + prebenda)
- [ ] Modulo 2: portal do contador (acesso externo restrito)
- [ ] Modulo 5: escola biblica/discipulado com trilhas certificadas
- [ ] Prontuario pastoral criptografado (E2EE)
- [ ] Trilha de auditoria imutavel (hash-chain)

**Checkpoint de saida:** Uma assembleia real de igreja parceira realiza votacao de diretoria 100% pelo sistema, com ata gerada automaticamente e validada juridicamente.

---

### FASE 4 - Inteligencia, IoT e Ecossistema (Semanas 41-56)
**Objetivo:** Inovacoes de mercado ("oceano azul") e escalabilidade do ecossistema.

- [ ] Motor de IA para predicao de evasao de membros
- [ ] Conciliacao via Open Finance (substituindo OFX manual)
- [ ] Reconhecimento facial no check-in infantil
- [ ] Smart Facilities (integracao IoT com fechaduras/climatizacao)
- [ ] Chosen Network (marketplace interno de membros)
- [ ] Marketplace de API publica para integracoes de terceiros
- [ ] Modulo de streaming integrado ao app
- [ ] Modulo 7: missoes e expansao
- [ ] Modulo 8: casamentos, funerais, capelania
- [ ] Modulo 11: botao de panico, gestao de estacionamento
- [ ] Multi-idioma e multi-moeda

**Checkpoint de saida:** Modelo de predicao de evasao validado com acuracia minima aceitavel (definir baseline com dados reais); pelo menos 1 integracao de terceiro publicada no marketplace.

---

### Visao Consolidada do Roadmap

| Fase | Duracao estimada | Foco | Resultado |
|---|---|---|---|
| Fase 0 | 6 semanas | Infraestrutura | Base tecnica multi-tenant validada |
| Fase 1 | 10 semanas | MVP | Produto vendavel para igreja local |
| Fase 2 | 12 semanas | Escala | Produto vendavel para redes/filiais |
| Fase 3 | 12 semanas | Diferenciacao | Nenhum concorrente cobre esse escopo |
| Fase 4 | 16 semanas | Inovacao | Lideranca de mercado via IA/IoT/ecossistema |

**Tempo total estimado ate o produto "lider de mercado completo": ~56 semanas (~13 meses)**, com possibilidade de comercializacao ja a partir da Fase 1 (mes 4).

---

## 11. Criterios de Aceite por Fase (resumo)

- **Fase 0:** Testes automatizados comprovam que um usuario da Filial A nunca consegue ler dados da Filial B via API, mesmo manipulando requisicoes.
- **Fase 1:** Zero dependencia de planilha externa para operacao diaria de secretaria/financeiro na igreja piloto.
- **Fase 2:** Calculo de repasse auditado bate 100% com conferencia manual em 3 ciclos consecutivos.
- **Fase 3:** Ata gerada pelo sistema aceita como documento oficial em cartorio/registro, quando exigido.
- **Fase 4:** Modelo preditivo de evasao supera baseline ingenuo (ex.: "todo mundo com queda de frequencia > 30 dias") em precisao.

---

## 12. Glossario

- **ChMS**: Church Management Software.
- **RLS**: Row-Level Security - mecanismo do PostgreSQL para isolamento de dados por linha.
- **E2EE**: End-to-End Encryption - criptografia de ponta a ponta.
- **Split de pagamento**: divisao automatica de um valor recebido entre multiplos destinatarios.
- **Open Finance**: sistema regulado pelo Banco Central que permite compartilhamento de dados financeiros via API com consentimento do usuario.
- **Multi-tenant**: arquitetura em que multiplos clientes (igrejas) compartilham a mesma infraestrutura com dados isolados.
- **RBAC**: Role-Based Access Control - controle de acesso baseado em papeis/funcoes.

---

*Documento vivo - recomenda-se revisao a cada fechamento de fase do roadmap.*
