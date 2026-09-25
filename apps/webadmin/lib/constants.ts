// Situacao no Rol (requisito 1.6). "professo" e DERIVADO: so `active` e
// professo (nao existe "professo inativo"); `member` e o nao professo que
// segue ativo na igreja. Espelha o CHECK da migracao 000022.
export const MEMBERSHIP_STATUS: Record<string, { label: string; tone: string }> = {
  active: { label: "Ativo (professo)", tone: "green" },
  member: { label: "Nao professo", tone: "sky" },
  inactive: { label: "Inativo", tone: "zinc" },
  dismissed: { label: "Baixado do Rol", tone: "red" },
  transferred: { label: "Transferido", tone: "amber" },
  deceased: { label: "Falecido", tone: "zinc" },
  other: { label: "Outros", tone: "zinc" },
};

/** Motivos de baixa (requisito 1.7). Espelha o CHECK da migracao 000022. */
export const EXIT_REASONS: Record<string, string> = {
  falecimento: "Falecimento",
  desligamento: "Desligamento a pedido do membro",
  transferencia: "Transferencia para outra igreja",
  abandono: "Abandono das atividades eclesiasticas",
  ausencia: "Ausencia superior a 1 ano",
  outro: "Outro",
};

/** Situacoes que exigem motivo/data de saida (baixa do Rol). */
export const EXIT_STATUSES = ["dismissed", "transferred", "deceased"];

/** Frequencia do membro (requisito 1.5), com historico. */
export const FREQUENCY: Record<string, { label: string; tone: string }> = {
  frequente: { label: "Frequente", tone: "green" },
  pouco_frequente: { label: "Pouco frequente", tone: "amber" },
  nao_frequente: { label: "Nao frequente", tone: "zinc" },
};

/** Tipos de evento do historico eclesiastico (requisito 1.8). */
export const MEMBER_HISTORY_KINDS: Record<string, string> = {
  cadastro: "Cadastro",
  status: "Alteracao de situacao",
  reativacao: "Reativacao",
  batismo_infantil: "Batismo infantil",
  profissao_fe: "Profissao de fe",
  recebido_jurisdicao: "Recebido por jurisdicao",
  recebido_transferencia: "Recebido por transferencia",
  transferencia: "Transferencia para outra igreja",
  desligamento: "Desligamento",
  abandono: "Abandono das atividades",
  baixa_rol: "Baixa do Rol",
  falecimento: "Falecimento",
  outro: "Outros",
};

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
