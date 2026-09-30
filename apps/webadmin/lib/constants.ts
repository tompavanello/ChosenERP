// Situacao no Rol (requisito 1.6, revisada pela migracao 000066): 3 situacoes.
// "professo" e DERIVADO: so `active` e professo (nao existe "professo inativo").
// `inactive` e subdividido pelo MOTIVO da inatividade (EXIT_REASONS).
export const MEMBERSHIP_STATUS: Record<string, { label: string; tone: string }> = {
  active: { label: "Ativo Professo", tone: "green" },
  member: { label: "Ativo Nao Professo", tone: "sky" },
  inactive: { label: "Inativo", tone: "zinc" },
};

/** Motivos da inatividade (requisito 1.7). Espelha o CHECK da migracao 000022. */
export const EXIT_REASONS: Record<string, string> = {
  falecimento: "Falecimento",
  desligamento: "Desligamento",
  transferencia: "Transferencia para outra igreja",
  abandono: "Abandono das atividades eclesiasticas",
  ausencia: "Ausencia superior a 1 ano",
  outro: "Outro",
};

/** Situacoes que exigem motivo/data de inatividade. */
export const EXIT_STATUSES = ["inactive"];

/** Frequencia do membro (requisito 1.5), com historico. */
export const FREQUENCY: Record<string, { label: string; tone: string }> = {
  frequente: { label: "Frequente", tone: "green" },
  pouco_frequente: { label: "Pouco frequente", tone: "amber" },
  nao_frequente: { label: "Nao frequente", tone: "zinc" },
};

/** Tipos de evento do historico eclesiastico (requisito 1.8).
 *  Fallback de exibicao: a fonte de verdade e o catalogo configuravel
 *  `member-event-kinds` (migracao 000065). */
export const MEMBER_HISTORY_KINDS: Record<string, string> = {
  cadastro: "Cadastro",
  status: "Alteracao de situacao",
  reativacao: "Reativacao",
  batismo_infantil: "Batismo de crianca",
  batismo_crianca: "Batismo de crianca",
  batismo_profissao_fe: "Batismo e profissao de fe",
  apresentacao_crianca: "Apresentacao de crianca",
  profissao_fe: "Profissao de fe",
  recebido_jurisdicao: "Recebimento por jurisdicao",
  recebido_transferencia: "Recebimento por transferencia",
  saida_transferencia: "Saida por transferencia",
  transferencia: "Saida por transferencia",
  desligamento_pedido: "Desligamento a pedido do membro",
  desligamento_disciplinar: "Desligamento por ato disciplinar",
  desligamento: "Desligamento",
  abandono_atividades: "Abandono das atividades",
  abandono: "Abandono das atividades",
  baixa_rol: "Baixa do Rol",
  saida_falecimento: "Saida por falecimento",
  falecimento: "Saida por falecimento",
  ordenacao_diacono: "Ordenacao de diacono(a)",
  ordenacao_presbitero: "Ordenacao de presbitero(a)",
  dissolucao_pastoral: "Dissolucao das relacoes pastorais",
  decisao_presbiterio: "Decisao do presbiterio",
  outro: "Outros",
};

/** Agrupamento dos eventos da vida eclesiastica por categoria (000065). */
export const EVENT_CATEGORY: Record<string, { label: string; order: number }> = {
  batismo: { label: "Batismo", order: 1 },
  profissao: { label: "Profissao de fe", order: 2 },
  recepcao: { label: "Recepcao", order: 3 },
  retorno: { label: "Retorno", order: 4 },
  saida: { label: "Saida / Baixa", order: 5 },
  ordenacao: { label: "Ordenacao", order: 6 },
  disciplina: { label: "Disciplina", order: 7 },
  sistema: { label: "Sistema (automatico)", order: 8 },
  outro: { label: "Outros", order: 9 },
};

/** Datas que um evento pode registrar (migracao 000067). */
export const EVENT_DATE_FIELDS: Record<string, string> = {
  none: "Nao atualiza data",
  baptism: "Data do batismo",
  joined_at: "Membro desde",
  marriage_date: "Data de casamento",
};

/** Tons aceitos pelo Badge. */
export const EVENT_TONES = ["zinc", "sky", "green", "amber", "red", "indigo", "brand"];

/** Dias da semana (0=domingo .. 6=sabado), na convencao do EXTRACT(DOW) do Postgres. */
export const WEEKDAYS: Record<number, string> = {
  0: "Domingo", 1: "Segunda", 2: "Terca", 3: "Quarta", 4: "Quinta", 5: "Sexta", 6: "Sabado",
};

/** Ordem de exibicao dos dias da semana (comeca no domingo). */
export const WEEKDAY_ORDER = [0, 1, 2, 3, 4, 5, 6];

export const GENDER: Record<string, string> = { male: "Masculino", female: "Feminino", other: "Outro" };

export const MARITAL_STATUS: Record<string, string> = {
  single: "Solteiro(a)",
  married: "Casado(a)",
  divorced: "Divorciado(a)",
  widowed: "Viuvo(a)",
};

export const PAYMENT_METHODS: Record<string, string> = {
  pix: "PIX",
  card: "Cartao",
  boleto: "Boleto",
  cash: "Especie",
  transfer: "Transferencia",
};

export const ACCOUNT_TYPES: Record<string, string> = {
  checking: "Conta corrente",
  savings: "Poupanca",
  cash: "Caixa",
};

export const JOURNEY_STAGES: Record<string, { label: string; tone: string; order: number }> = {
  welcome: { label: "Boas-vindas", tone: "zinc", order: 0 },
  coffee_pastor: { label: "Cafe com o Pastor", tone: "sky", order: 1 },
  course: { label: "Curso de principios", tone: "sky", order: 2 },
  cell: { label: "Celula", tone: "indigo", order: 3 },
  converted: { label: "Convertido", tone: "green", order: 4 },
};

export const JOURNEY_ORDER = ["welcome", "coffee_pastor", "course", "cell", "converted"];

export const VISITOR_SOURCES: { value: string; label: string }[] = [
  { value: "indicado", label: "Indicado por membro" },
  { value: "evento", label: "Evento" },
  { value: "google", label: "Google" },
  { value: "porta", label: "Visita na porta" },
  { value: "redes sociais", label: "Redes sociais" },
];

export const RELATION_LABELS: Record<string, string> = {
  spouse: "Conjuge",
  parent: "Pai/Mae",
  child: "Filho(a)",
  discipler: "Discipulador(a)",
  disciple: "Discipulo(a)",
  dependent: "Dependente",
  relative: "Parente",
};

// OFFICES (diacono/presbitero/...) foi removido: o catalogo de cargos agora vem
// do banco (GET /api/v1/cargos) e a igreja pode criar os seus. A coluna legada
// members.office continua sendo lida para exibicao, mas nao e mais a fonte de
// verdade - ver CARGOS_KIND e o seletor de cargos do formulario do membro.

/** Agrupamentos do catalogo de cargos (espelham o CHECK da migracao 000020). */
export const CARGO_KINDS: Record<string, { label: string; tone: string }> = {
  eclesiastico: { label: "Eclesiastico", tone: "brand" },
  lideranca: { label: "Lideranca", tone: "sky" },
  ensino: { label: "Ensino", tone: "green" },
  apoio: { label: "Apoio", tone: "amber" },
  outro: { label: "Outro", tone: "zinc" },
};

export const CARGO_STATUS: Record<string, { label: string; tone: string }> = {
  ativo: { label: "Ativo", tone: "green" },
  encerrado: { label: "Encerrado", tone: "zinc" },
};

/** Janela (em dias) para sinalizar um mandato como "vencendo". */
export const CARGO_EXPIRY_WINDOW_DAYS = 60;

export const PERMISSION_LABELS: Record<string, string> = {
  "members.read": "Ver membros",
  "members.write": "Cadastrar/editar membros",
  "members.delete": "Excluir membro",
  "families.read": "Ver familias",
  "finance.read": "Ver financeiro",
  "finance.write": "Lancar financeiro",
  "finance.reconcile": "Conciliar financeiro (trava o periodo)",
  "finance.audit": "Auditar financeiro",
  "finance.authorize": "Aprovar orcamento",
  "ministries.write": "Gerir ministerios",
  "governance.write": "Gerir governanca",
  "reports.read": "Ver relatorios",
};

// NAV_BY_PERMISSION foi removido: estava morto e desatualizado (nao incluia
// transfers/ministries/announcements). O menu real e o NAV_SECTIONS inline em
// app/dashboard/layout.tsx - fonte de verdade unica para nao voltar a divergir.
// "families" saiu junto: o cadastro de familias virou aba dentro do membro.
