export const MEMBERSHIP_STATUS: Record<string, { label: string; tone: string }> = {
  active: { label: "Ativo", tone: "bg-emerald-50 text-emerald-700" },
  member: { label: "Membro", tone: "bg-emerald-50 text-emerald-700" },
  inactive: { label: "Inativo", tone: "bg-zinc-100 text-zinc-600" },
  visitor: { label: "Visitante", tone: "bg-amber-50 text-amber-700" },
  transferred: { label: "Transferido", tone: "bg-sky-50 text-sky-700" },
  deceased: { label: "Falecido", tone: "bg-zinc-100 text-zinc-500" },
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

export const JOURNEY_STAGES: Record<string, { label: string; tone: string; order: number }> = {
  welcome: { label: "Boas-vindas", tone: "bg-zinc-100 text-zinc-600", order: 0 },
  coffee_pastor: { label: "Café com o Pastor", tone: "bg-sky-50 text-sky-700", order: 1 },
  course: { label: "Curso de princípios", tone: "bg-violet-50 text-violet-700", order: 2 },
  cell: { label: "Célula", tone: "bg-indigo-50 text-indigo-700", order: 3 },
  converted: { label: "Convertido", tone: "bg-emerald-50 text-emerald-700", order: 4 },
};

export const RELATION_LABELS: Record<string, string> = {
  spouse: "Cônjuge",
  parent: "Pai/Mãe",
  child: "Filho(a)",
  discipler: "Discipulador(a)",
  disciple: "Discípulo(a)",
  dependent: "Dependente",
  relative: "Parente",
};

export const OFFICES: Record<string, string> = {
  diacono: "Diácono",
  presbitero: "Presbítero",
  evangelista: "Evangelista",
  pastor: "Pastor",
  missionario: "Missionário",
};

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

// Menus do sidebar: exibidos se o papel tiver pelo menos uma das permissões.
export const NAV_BY_PERMISSION: { key: string; minPerm: string[] }[] = [
  { key: "overview", minPerm: [] },
  { key: "members", minPerm: ["members.read"] },
  { key: "families", minPerm: ["families.read"] },
  { key: "visitors", minPerm: ["members.read"] },
  { key: "benefactors", minPerm: ["members.read"] },
  { key: "finance", minPerm: ["finance.read"] },
  { key: "reports", minPerm: ["reports.read", "finance.read"] },
];
