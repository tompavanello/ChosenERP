# Chosen ERP
## Documentação Completa — PRD, Especificação de Funcionalidades e Roadmap de Desenvolvimento

**Versão:** 1.0
**Data:** Setembro/2026
**Domínio:** chosenerp.com.br

---

## Sumário

1. Sumário Executivo
2. Visão de Produto (PRD)
3. Personas e Perfis de Acesso
4. Arquitetura Técnica
5. Modelo de Dados (Multi-tenant)
6. Especificação Completa de Funcionalidades (por módulo)
7. Requisitos Não Funcionais
8. Métricas de Sucesso (KPIs)
9. Riscos e Mitigação
10. Roadmap e Checkpoints de Desenvolvimento (Fases 0 a 4)
11. Critérios de Aceite por Fase
12. Glossário

---

## 1. Sumário Executivo

O **Chosen ERP** é uma plataforma SaaS multi-tenant de gestão eclesiástica que unifica administração de secretaria, controle financeiro de alta performance, gestão de ministérios/voluntariado, discipulado, governança institucional e comunicação digital em um único ecossistema.

O produto se diferencia da concorrência nacional (BeChurch, Eklesia, Cathedral, GIgreja) e internacional (ChMeetings, Planning Center) por três pilares:

- **Profundidade financeira de nível ERP corporativo** (repasses automáticos entre filiais, Open Finance, folha de pagamento, auditoria imutável).
- **Inteligência de dados** (predição de evasão de membros, mapas de calor de células, trilhas de discipulado gamificadas).
- **Governança institucional completa** (atas digitais, votação eletrônica, gestão de mandatos, portal do contador) — área hoje inexplorada pelos concorrentes.

Stack: **Go** (serviços críticos de auth/financeiro), **Python/FastAPI + Celery** (serviços de dados, relatórios, integrações), **PostgreSQL** (com Row-Level Security para isolamento multi-tenant), **Redis + RabbitMQ** (cache e mensageria).

---

## 2. Visão de Produto (PRD)

### 2.1 Problema

Igrejas de médio e grande porte hoje utilizam sistemas fragmentados: uma planilha para dízimos, um app de terceiros para escalas, WhatsApp manual para comunicação, e nenhuma ferramenta cobre governança (atas, mandatos, compliance fiscal) ou dá visibilidade preditiva sobre saúde da congregação (risco de evasão, engajamento).

### 2.2 Proposta de Valor

> "O único sistema que cuida da alma da igreja (pastoral, discipulado, família) e do corpo administrativo (finanças, governança, compliance) em uma única plataforma — com inteligência que antecipa problemas antes que eles aconteçam."

### 2.3 Público-Alvo

| Segmento | Perfil | Necessidade Principal |
|---|---|---|
| Igrejas locais (100–500 membros) | Independentes ou pequenas redes | Substituir planilhas, organizar secretaria |
| Redes/Denominações (500–5.000 membros, múltiplas filiais) | Matriz + congregações | Repasses financeiros, visão consolidada, padronização |
| Grandes Convenções (5.000+ membros, dezenas de filiais) | Estrutura corporativa | Compliance, auditoria, folha de pagamento, governança estatutária |
| Ministérios especializados | Infantil, louvor, missões | Módulos verticais (check-in, escalas, sustento missionário) |

### 2.4 Objetivos do Produto

1. Centralizar dados demográficos, financeiros, espirituais e de engajamento da igreja.
2. Reduzir em ≥70% o trabalho manual da secretaria (emissão de documentos, certificados, carteirinhas).
3. Garantir transparência e compliance financeiro/jurídico (livro-caixa, repasses, atas, auditoria).
4. Antecipar problemas pastorais via dados (evasão, queda de engajamento).
5. Entregar a melhor experiência de membro do mercado via SuperApp.

### 2.5 Fora de Escopo (explicitamente não cobre no MVP)

- Contabilidade fiscal completa (o sistema gera dados para o contador, não substitui um sistema contábil homologado).
- Emissão de nota fiscal de produtos (não é ERP de comércio).
- Gestão de escolas/colégios confessionais (pode ser módulo futuro separado).

---

## 3. Personas e Perfis de Acesso (RBAC)

| Perfil | Acesso |
|---|---|
| **Super Admin (Chosen)** | Configuração da plataforma, suporte a tenants |
| **Admin da Sede** | Visão consolidada de todas as filiais, financeiro global, configurações institucionais |
| **Admin/Pastor da Filial** | Dados da sua congregação: membros, financeiro local, escalas |
| **Tesoureiro** | Módulo financeiro completo da sua unidade, sem acesso a prontuário pastoral |
| **Secretário(a)** | Cadastros, emissão de documentos, certificados |
| **Líder de Ministério/Célula** | Escalas do seu grupo, frequência, membros do seu grupo |
| **Pastor/Conselheiro** | Prontuário pastoral criptografado (E2EE), aconselhamento |
| **Contador Externo** | Portal restrito: exportação fiscal, sem dados pastorais |
| **Membro (App)** | Autoatendimento: doações, carteirinha, eventos, devocional |
| **Visitante** | Cadastro simplificado, trilha de acolhimento |

---

## 4. Arquitetura Técnica

### 4.1 Stack

| Camada | Tecnologia | Função |
|---|---|---|
| Banco de dados | PostgreSQL 16+ | ACID, JSONB para formulários dinâmicos, RLS para isolamento multi-tenant |
| Backend crítico | Go (Golang) | Auth/RBAC, motor financeiro, API Gateway, WebSockets |
| Backend de dados | Python (FastAPI + Celery) | Relatórios, PDFs, integrações externas, IA/ML |
| Cache | Redis | Sessões, consultas frequentes |
| Mensageria | RabbitMQ | Processamento assíncrono (disparo de mensagens, cálculo de repasses) |
| Storage | S3 (ou equivalente) | Fotos, comprovantes, documentos |
| Observabilidade | Prometheus + Grafana + OpenTelemetry | Monitoramento, tracing distribuído |

### 4.2 Microsserviços

1. **Auth & Tenant Service (Go)** — login, JWT, RBAC, resolução de tenant/filial.
2. **Ledger & Finance Service (Go)** — transações financeiras, repasses, conciliação, folha de pagamento.
3. **Member & Roster Service (Python)** — CRUD de membros, genealogia, células, discipulado.
4. **Communication & Worker Service (Python/Celery)** — filas de notificação, geração de relatórios pesados.
5. **Governance Service (Go)** — atas, votações, mandatos, trilha de auditoria.
6. **Integration Hub (Python)** — Open Finance, WhatsApp Business API, gateways de pagamento, IoT.

### 4.3 Padrões de Integração

- API REST + gRPC interno entre serviços.
- Webhooks para integrações externas (gateways de pagamento, WhatsApp).
- API pública documentada (OpenAPI) para o futuro Marketplace de Integrações.

---

## 5. Modelo de Dados (Multi-tenant)

Estratégia: **banco compartilhado com isolamento lógico via Row-Level Security (RLS)**.

### 5.1 Tabelas-núcleo (principais)

- `tenants` — sede/convenção
- `branches` — filiais/congregações (FK: tenant_id)
- `users` — usuários administrativos (FK: branch_id, role)
- `members` — membros (FK: branch_id)
- `families` — núcleos familiares
- `member_relationships` — vínculos (cônjuge, filho, pai, discipulador)
- `visitors` — visitantes e funil de acolhimento
- `benefactors` — benfeitores externos
- `ministries` — ministérios/departamentos
- `ministry_members` — vínculo membro-ministério (papel: líder/voluntário)
- `small_groups` — células/pequenos grupos (com geolocalização)
- `group_attendance` — frequência em células
- `financial_transactions` — lançamentos (append-only, imutável)
- `financial_categories` — plano de contas
- `transfers` — repasses entre filiais
- `assets` — patrimônio (com QR code)
- `payroll` — folha de pagamento (CLT + prebenda)
- `events` — eventos e cultos
- `service_liturgy` — ordem de culto
- `rosters` — escalas de voluntários
- `checkins` — check-in infantil
- `minutes` (atas) — governança
- `votes` — votações eletrônicas
- `audit_log` — trilha de auditoria imutável (hash-chained)
- `pastoral_records` — prontuário pastoral (criptografado E2EE)
- `documents` — certificados, cartas, carteirinhas gerados

### 5.2 Segurança

- RLS garante que toda query filtra automaticamente por `branch_id`, exceto para perfis com escopo "Sede".
- `pastoral_records` usa criptografia de ponta a ponta — chave de decriptação vinculada ao usuário pastor, não recuperável por admin de sistema.
- `financial_transactions` e `audit_log` são **append-only** (sem UPDATE/DELETE), com hash encadeado para garantir integridade (estilo blockchain simplificado).

---

## 6. Especificação Completa de Funcionalidades

### MÓDULO 1 — Gestão de Pessoas e Secretaria Digital

**1.1 Cadastro de Membros**
- Dados pessoais completos (nome, CPF/RG, nascimento, estado civil, profissão, contato, foto).
- Dados eclesiásticos: status (ativo/inativo/visitante/transferido/falecido), datas de batismo/adesão, cargo/ofício.
- Perfil 360º: linha do tempo completa (visita → conversão → batismo → cursos → doações → voluntariado).
- Histórico de participação e anotações pastorais com controle de acesso granular.

**1.2 Gestão de Famílias e Genealogia**
- Vinculação em núcleos familiares (chefe, cônjuge, filhos, dependentes).
- Árvore genealógica visual e interativa (parentesco).
- **Árvore de discipulado espiritual** (quem evangelizou/batizou quem — diferencial exclusivo).

**1.3 Benfeitores e Visitantes**
- Cadastro rápido de benfeitores sem vínculo de membresia.
- Cadastro de visitantes com **Trilha de Maturidade / Onboarding Gamificado**: boas-vindas automática → convite para "Café com o Pastor" → curso de princípios → integração em célula.

**1.4 Secretaria Digital**
- Emissão automática de carteirinha (física e digital via QR Code).
- Cartas de transferência e recomendação.
- Certificados (batismo, casamento, apresentação de bebês).
- Gestão multi-igrejas: hierarquia Sede > Congregações > Sub-congregações.
- Transferência digital de cadastro entre filiais.

**1.5 Compliance de Dados (LGPD)**
- Termo de consentimento digital para dados sensíveis (religiosos, imagem).
- Exportação total de dados do titular (portabilidade).
- Anonimização/exclusão sob solicitação.

---

### MÓDULO 2 — Financeiro e Arrecadação

**2.1 Entradas**
- Dízimos: registro individualizado, opção de anonimato.
- Ofertas/doações: por culto, evento, campanha ou doador.
- Formas de pagamento: PIX, boleto, cartão, espécie.
- **Doações recorrentes** (débito automático mensal via cartão/PIX).
- **Campanha de crowdfunding** para obras/construção com página pública e termômetro de meta.
- Recibos digitais automáticos (e-mail/WhatsApp), incluindo **recibo com validade jurídica para dedução de IR** quando aplicável.

**2.2 Saídas**
- Contas a pagar/pagas com plano de contas (manutenção, ação social, eventos, salários, mídia, utilidades).
- Anexo de comprovantes digitais.
- Centro de custo por ministério/célula.
- **Orçamento inteligente (budgeting)**: teto de gastos por ministério, com workflow de aprovação do tesoureiro.

**2.3 Repasses e Multi-filial**
- **Motor de repasses dinâmico (split de pagamento)**: regras configuráveis (ex.: 10% filial → sede, 5% missões, 2% convenção estadual), executado automaticamente.

**2.4 Conciliação e Compliance**
- Conciliação bancária via importação OFX.
- **Conciliação via Open Finance** (integração direta com Banco Central — categorização em tempo real, sem upload manual).
- Fluxo de caixa (realizado vs. previsto).
- **Folha de pagamento**: CLT (funcionários) + prebenda/pró-labore (pastores), com regimes tributários distintos.
- **Trilha de auditoria imutável**: log com hash de integridade de toda alteração financeira.
- **Portal do Contador**: acesso externo restrito para exportação fiscal sem acesso a dados pastorais.

**2.5 Patrimônio**
- Cadastro de bens (instrumentos, equipamentos, veículos, imóveis).
- Etiquetas QR Code para inventário físico.
- Cálculo automático de depreciação.
- Histórico de manutenção.

**2.6 Hardware Integrado**
- Totens e máquinas POS Chosen: doador digita CPF, passa cartão, recibo cai automaticamente no app.

---

### MÓDULO 3 — Ministérios, Voluntariado e Pequenos Grupos

**3.1 Ministérios**
- Cadastro de ministérios/departamentos (louvor, infantil, ação social, recepção, ensino, mídia).
- Atribuição de líderes, coordenadores, voluntários.
- **Banco de dons e talentos**: teste de dons espirituais + habilidades profissionais para escalar por competência.

**3.2 Escalas**
- Criação de escalas com notificação e confirmação de presença.
- **Escalas inteligentes**: cruzamento de disponibilidade, férias e regras internas (ex.: bloquear escala de quem não atingiu frequência mínima na célula).
- Detecção automática de conflitos de agenda.

**3.3 Células / Pequenos Grupos / EBD**
- Mapeamento por região/bairro, controle de frequência, registro de estudos.
- **Mapa de calor geográfico**: identifica áreas com concentração de membros sem célula e sugere abertura de novos grupos.
- Geolocalização para o membro encontrar a célula mais próxima.

**3.4 Verificação de Voluntários**
- **Background check** integrado para voluntários de ministério infantil (obrigatório em várias denominações).

---

### MÓDULO 4 — Cultos, Eventos e Mídias

**4.1 Planejamento de Culto**
- Ordem de culto colaborativa (liturgia, músicas, tempo de pregação, cronômetro).
- Integração com softwares de projeção (Hollyrics/ProPresenter via API).
- Repositório de mídia: cifras, partituras, vídeos.

**4.2 Check-in Infantil**
- Check-in via tablet/totem com impressão de etiquetas de segurança.
- **Reconhecimento facial** para reduzir filas: reconhece o núcleo familiar e imprime etiqueta com restrições alimentares automaticamente.

**4.3 Eventos**
- Venda/reserva de vagas para retiros, conferências, jantares.
- Mapa de assentos/camas em alojamentos de retiro.

**4.4 Espaços e Reservas**
- Controle de salas, veículos, equipamentos — evita conflitos de reserva entre ministérios.

**4.5 Streaming**
- Plataforma de transmissão ao vivo integrada ao app (sem anúncios de terceiros), com botões de "Pedido de Oração" e "Ofertar" em tempo real durante a live.

---

### MÓDULO 5 — Discipulado e Educação

- Escola Bíblica/Seminário interno (EAD + presencial), com trilhas certificadas e pré-requisitos entre módulos.
- Biblioteca física/digital com controle de empréstimos e liberação de e-books por nível de discipulado.
- Devocional diário no app com plano de leitura bíblica gamificado (badges de constância).

---

### MÓDULO 6 — Governança e Compliance Institucional *(diferencial exclusivo)*

- **Livro de atas digital** com assinatura eletrônica válida juridicamente.
- **Votação eletrônica** para assembleias (eleição de diretoria, aprovação de orçamento, moções), com apuração e ata automáticas.
- **Gestão de mandatos**: alertas de vencimento de mandato de cargos estatutários.
- **Gestão de convênios e documentação legal**: escrituras, alvarás, contratos, seguros, com alertas de vencimento.

---

### MÓDULO 7 — Missões e Expansão

- Cadastro de missionários enviados (nacionais/internacionais).
- Controle de sustento financeiro mensal e prestação de contas do campo.
- Painel de visão de expansão: metas de plantação de igrejas vinculadas a indicadores (discípulos, células maduras).

---

### MÓDULO 8 — Ciclo de Vida e Cerimônias

- **Casamentos**: checklist de sessões de aconselhamento pré-nupcial, agenda de cerimônia, certidão.
- **Funerais e cuidado ao luto**: registro, follow-up pastoral automático em 30/60/90 dias.
- **Capelania**: agenda de visitas hospitalares/prisionais, relatórios de atendimento.

---

### MÓDULO 9 — Comunicação e SuperApp

- **SuperApp White-Label**: Bíblia nativa, hinário, carteirinha digital dinâmica, devocional gamificado, inscrição/pagamento de eventos.
- **Central de comunicação omnichannel**: push, e-mail, SMS, WhatsApp oficial — segmentado por filtro (ex.: só homens casados da filial Centro).
- **Integração nativa WhatsApp**: aniversariantes, lembretes de escala, boas-vindas a visitantes.
- Mural de avisos e notificações push segmentadas.

---

### MÓDULO 10 — Inteligência de Dados e Inovação

- **Motor de IA para predição de evasão**: cruza check-in, leitura, frequência em célula e contribuição; alerta o pastor com sugestão de ação.
- **Smart Facilities (IoT)**: liga ar-condicionado e destrava porta automaticamente antes de eventos agendados.
- **Chosen Network**: marketplace interno de serviços entre membros (economia da comunidade).
- **Marketplace de API pública**: permite integrações de terceiros (contabilidade, som, projeção).

---

### MÓDULO 11 — Segurança Física e Operacional

- **Botão de pânico** integrado ao app para emergências durante cultos.
- Gestão de estacionamento com sinalização de lotação em tempo real.

---

### MÓDULO 12 — Relatórios e Dashboards de Gestão

**Financeiros**
- DRE eclesiástico, balancete mensal/anual.
- Comparativo por período (dízimos vs. ofertas vs. despesas).
- Relatório de inadimplência/queda de contribuição (com sigilo preservado).

**Demográficos e Administrativos**
- Dashboard de crescimento (retenção, novos membros, batismos).
- Pirâmide etária, mapa de distribuição geográfica.
- Aniversariantes do mês (membros e casamentos).
- Engajamento e frequência por ministério/célula.

**Governança**
- Painel de mandatos vigentes/vencendo.
- Histórico de votações e atas.

---

## 7. Requisitos Não Funcionais

| Categoria | Requisito |
|---|---|
| Arquitetura | Multi-tenant com isolamento via RLS no PostgreSQL |
| Segurança | RBAC granular, criptografia E2EE para prontuário pastoral, MFA para admins |
| Compliance | LGPD nativo (consentimento, portabilidade, exclusão) |
| Disponibilidade | SLA 99,9%, backups diários, auto-scaling para picos de domingo |
| Performance | API p95 < 300ms para operações de leitura; motor financeiro com garantia ACID |
| Auditoria | Logs imutáveis com hash-chain para transações financeiras e governança |
| Portabilidade | Exportação total de dados do tenant a qualquer momento |
| Internacionalização | Suporte a multi-idioma e multi-moeda (fase 3+) |

---

## 8. Métricas de Sucesso (KPIs)

- **Adoção (MAU)**: % de membros ativos usando o app mensalmente.
- **Engajamento de voluntários**: redução de ausências não avisadas nas escalas.
- **Eficiência financeira**: % de doações digitais (PIX/cartão/recorrente) vs. espécie.
- **Retenção de visitantes**: taxa de conversão visitante → membro após trilha de acolhimento.
- **Precisão preditiva**: acurácia do modelo de predição de evasão (validado contra evasão real).
- **Tempo de secretaria**: redução média de horas/semana gastas em tarefas manuais.

---

## 9. Riscos e Mitigação

| Risco | Impacto | Mitigação |
|---|---|---|
| Vazamento de dados pastorais sensíveis | Alto | Criptografia E2EE, RLS, auditoria de acesso |
| Erro em cálculo de repasses financeiros | Alto | Testes automatizados extensivos, transações ACID, dupla checagem em produção |
| Baixa adoção pelo membro comum (app) | Médio | UX simplificado, gamificação, treinamento de onboarding |
| Dependência de APIs externas (Open Finance, WhatsApp) | Médio | Circuit breakers, fallback manual (OFX/mensagem manual) |
| Complexidade de customização por denominação | Médio | Motor de campos dinâmicos (JSONB) desde o MVP |
| Resistência de lideranças tradicionais à mudança | Médio | Fase de piloto com igrejas parceiras, suporte de migração de dados |

---

## 10. Roadmap e Checkpoints de Desenvolvimento

### FASE 0 — Fundação Técnica (Semanas 1–6)
**Objetivo:** Infraestrutura pronta para receber os primeiros módulos.

- [ ] Setup de repositórios, CI/CD, ambientes (dev/staging/prod)
- [ ] Provisionamento PostgreSQL com estratégia RLS definida e testada
- [ ] Auth & Tenant Service (Go): login, JWT, RBAC básico
- [ ] Estrutura multi-tenant validada com tenant + branch fake de teste
- [ ] Observabilidade (logs, métricas, tracing) configurada
- [ ] Object storage (S3) configurado

**Checkpoint de saída:** Login funcional, criação de tenant/filial, isolamento de dados comprovado por teste automatizado de RLS.

---

### FASE 1 — MVP: Secretaria + Financeiro Básico (Semanas 7–16)
**Objetivo:** Substituir planilhas — produto vendável para igrejas locais.

- [ ] Módulo 1: cadastro de membros, famílias, árvore genealógica básica
- [ ] Módulo 1: cadastro de visitantes e benfeitores
- [ ] Módulo 2: lançamento de dízimos, ofertas, despesas (manual)
- [ ] Módulo 2: plano de contas e centro de custo
- [ ] Módulo 2: recibos digitais automáticos
- [ ] Módulo 12: relatórios financeiros básicos (balancete, DRE simplificado)
- [ ] Emissão de carteirinha digital (QR Code)
- [ ] App do membro — versão básica (ver carteirinha, ver avisos)

**Checkpoint de saída:** Uma igreja piloto opera 30 dias consecutivos só com o Chosen ERP para secretaria e financeiro, sem planilha paralela.

---

### FASE 2 — Multi-filial + Ministérios + Comunicação (Semanas 17–28)
**Objetivo:** Atender redes de igrejas e cobrir voluntariado/comunicação.

- [ ] Hierarquia Sede > Filiais, com visão consolidada
- [ ] Motor de repasses financeiros (split de pagamento)
- [ ] Módulo 3: ministérios, escalas, células com frequência
- [ ] Módulo 4: check-in infantil (com etiqueta, sem reconhecimento facial ainda)
- [ ] Módulo 9: integração WhatsApp (aniversariantes, lembretes de escala, boas-vindas)
- [ ] Trilha de acolhimento automatizada (onboarding de visitantes)
- [ ] Conciliação bancária via OFX
- [ ] Doações recorrentes (débito automático)

**Checkpoint de saída:** Uma rede com ao menos 3 filiais opera com repasse automático validado por auditoria financeira, e uma igreja usa check-in infantil em culto real.

---

### FASE 3 — Governança + Patrimônio + Discipulado (Semanas 29–40)
**Objetivo:** Diferenciação competitiva — módulos que nenhum concorrente tem.

- [ ] Módulo 6: atas digitais com assinatura eletrônica
- [ ] Módulo 6: votação eletrônica para assembleias
- [ ] Módulo 6: gestão de mandatos e documentação legal
- [ ] Módulo 2: gestão de patrimônio com QR Code e depreciação
- [ ] Módulo 2: folha de pagamento (CLT + prebenda)
- [ ] Módulo 2: portal do contador (acesso externo restrito)
- [ ] Módulo 5: escola bíblica/discipulado com trilhas certificadas
- [ ] Prontuário pastoral criptografado (E2EE)
- [ ] Trilha de auditoria imutável (hash-chain)

**Checkpoint de saída:** Uma assembleia real de igreja parceira realiza votação de diretoria 100% pelo sistema, com ata gerada automaticamente e validada juridicamente.

---

### FASE 4 — Inteligência, IoT e Ecossistema (Semanas 41–56)
**Objetivo:** Inovações de mercado ("oceano azul") e escalabilidade do ecossistema.

- [ ] Motor de IA para predição de evasão de membros
- [ ] Conciliação via Open Finance (substituindo OFX manual)
- [ ] Reconhecimento facial no check-in infantil
- [ ] Smart Facilities (integração IoT com fechaduras/climatização)
- [ ] Chosen Network (marketplace interno de membros)
- [ ] Marketplace de API pública para integrações de terceiros
- [ ] Módulo de streaming integrado ao app
- [ ] Módulo 7: missões e expansão
- [ ] Módulo 8: casamentos, funerais, capelania
- [ ] Módulo 11: botão de pânico, gestão de estacionamento
- [ ] Multi-idioma e multi-moeda

**Checkpoint de saída:** Modelo de predição de evasão validado com acurácia mínima aceitável (definir baseline com dados reais); pelo menos 1 integração de terceiro publicada no marketplace.

---

### Visão Consolidada do Roadmap

| Fase | Duração estimada | Foco | Resultado |
|---|---|---|---|
| Fase 0 | 6 semanas | Infraestrutura | Base técnica multi-tenant validada |
| Fase 1 | 10 semanas | MVP | Produto vendável para igreja local |
| Fase 2 | 12 semanas | Escala | Produto vendável para redes/filiais |
| Fase 3 | 12 semanas | Diferenciação | Nenhum concorrente cobre esse escopo |
| Fase 4 | 16 semanas | Inovação | Liderança de mercado via IA/IoT/ecossistema |

**Tempo total estimado até o produto "líder de mercado completo": ~56 semanas (~13 meses)**, com possibilidade de comercialização já a partir da Fase 1 (mês 4).

---

## 11. Critérios de Aceite por Fase (resumo)

- **Fase 0:** Testes automatizados comprovam que um usuário da Filial A nunca consegue ler dados da Filial B via API, mesmo manipulando requisições.
- **Fase 1:** Zero dependência de planilha externa para operação diária de secretaria/financeiro na igreja piloto.
- **Fase 2:** Cálculo de repasse auditado bate 100% com conferência manual em 3 ciclos consecutivos.
- **Fase 3:** Ata gerada pelo sistema aceita como documento oficial em cartório/registro, quando exigido.
- **Fase 4:** Modelo preditivo de evasão supera baseline ingênuo (ex.: "todo mundo com queda de frequência > 30 dias") em precisão.

---

## 12. Glossário

- **ChMS**: Church Management Software.
- **RLS**: Row-Level Security — mecanismo do PostgreSQL para isolamento de dados por linha.
- **E2EE**: End-to-End Encryption — criptografia de ponta a ponta.
- **Split de pagamento**: divisão automática de um valor recebido entre múltiplos destinatários.
- **Open Finance**: sistema regulado pelo Banco Central que permite compartilhamento de dados financeiros via API com consentimento do usuário.
- **Multi-tenant**: arquitetura em que múltiplos clientes (igrejas) compartilham a mesma infraestrutura com dados isolados.
- **RBAC**: Role-Based Access Control — controle de acesso baseado em papéis/funções.

---

*Documento vivo — recomenda-se revisão a cada fechamento de fase do roadmap.*
