// Situação no Rol (requisito 1.6). "professo" é DERIVADO: só `active` é
// professo (não existe "professo inativo"); `member` é o não professo que
// segue ativo na igreja. Espelha o CHECK da migração 000022.
export const MEMBERSHIP_STATUS: Record<string, { label: string; tone: string }> = {
  active: { label: "Ativo (professo)", tone: "green" },
  member: { label: "Não professo", tone: "sky" },
  inactive: { label: "Inativo", tone: "zinc" },
  dismissed: { label: "Baixado do Rol", tone: "red" },
  transferred: { label: "Transferido", tone: "amber" },
  deceased: { label: "Falecido", tone: "zinc" },
  other: { label: "Outros", tone: "zinc" },
};

/** Motivos de baixa (requisito 1.7). Espelha o CHECK da migração 000022. */
export const EXIT_REASONS: Record<string, string> = {
  falecimento: "Falecimento",
  desligamento: "Desligamento a pedido do membro",
  transferencia: "Transferência para outra igreja",
  abandono: "Abandono das atividades eclesiásticas",
  ausencia: "Ausência superior a 1 ano",
  outro: "Outro",
};

/** Situações que exigem motivo/data de saída (baixa do Rol). */
export const EXIT_STATUSES = ["dismissed", "transferred", "deceased"];

/** Frequência do membro (requisito 1.5), com histórico. */
export const FREQUENCY: Record<string, { label: string; tone: string }> = {
  frequente: { label: "Frequente", tone: "green" },
  pouco_frequente: { label: "Pouco frequente", tone: "amber" },
  nao_frequente: { label: "Não frequente", tone: "zinc" },
};

/** Tipos de evento do histórico eclesiástico (requisito 1.8). */
export const MEMBER_HISTORY_KINDS: Record<string, string> = {
  cadastro: "Cadastro",
  status: "Alteração de situação",
  reativacao: "Reativação",
  batismo_infantil: "Batismo infantil",
  profissao_fe: "Profissão de fé",
  recebido_jurisdicao: "Recebido por jurisdição",
  recebido_transferencia: "Recebido por transferência",
  transferencia: "Transferência para outra igreja",
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
  widowed: "Viúvo(a)",
};

export const PAYMENT_METHODS: Record<string, string> = {
  pix: "PIX",
  card: "Cartão",
  boleto: "Boleto",
  cash: "Espécie",
  transfer: "Transferência",
};

export const ACCOUNT_TYPES: Record<string, string> = {
  checking: "Conta corrente",
  savings: "Poupança",
  cash: "Caixa",
};

export const JOURNEY_STAGES: Record<string, { label: string; tone: string; order: number }> = {
  welcome: { label: "Boas-vindas", tone: "zinc", order: 0 },
  coffee_pastor: { label: "Café com o Pastor", tone: "sky", order: 1 },
  course: { label: "Curso de princípios", tone: "sky", order: 2 },
  cell: { label: "Célula", tone: "indigo", order: 3 },
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
  spouse: "Cônjuge",
  parent: "Pai/Mãe",
  child: "Filho(a)",
  discipler: "Discipulador(a)",
  disciple: "Discípulo(a)",
  dependent: "Dependente",
  relative: "Parente",
};

// OFFICES (diacono/presbitero/...) foi removido: o catálogo de cargos agora vem
// do banco (GET /api/v1/cargos) e a igreja pode criar os seus. A coluna legada
// members.office continua sendo lida para exibição, mas não é mais a fonte de
// verdade — ver CARGOS_KIND e o seletor de cargos do formulário do membro.

/** Agrupamentos do catálogo de cargos (espelham o CHECK da migração 000020). */
export const CARGO_KINDS: Record<string, { label: string; tone: string }> = {
  eclesiastico: { label: "Eclesiástico", tone: "brand" },
  lideranca: { label: "Liderança", tone: "sky" },
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
  "families.read": "Ver famílias",
  "finance.read": "Ver financeiro",
  "finance.write": "Lançar financeiro",
  "finance.authorize": "Aprovar orçamento",
  "ministries.write": "Gerir ministérios",
  "governance.write": "Gerir governança",
  "reports.read": "Ver relatórios",
};

// NAV_BY_PERMISSION foi removido: estava morto e desatualizado (não incluía
// transfers/ministries/announcements). O menu real é o NAV_SECTIONS inline em
// app/dashboard/layout.tsx — fonte de verdade única para não voltar a divergir.
// "families" saiu junto: o cadastro de famílias virou aba dentro do membro.
