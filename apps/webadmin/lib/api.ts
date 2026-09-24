// A API e servida na MESMA origem do webadmin, por caminho: o tunel Cloudflare
// manda /api/* para o Go e todo o resto para o Next; localmente o rewrite do
// next.config.ts (API_ORIGIN) faz o mesmo papel. Por isso todo fetch neste
// modulo e RELATIVO - o browser resolve contra a origem que ele abriu.
//
// Isto ja foi uma URL absoluta gravada no bundle em build time (API_URL). O
// custo era concreto: o mesmo build nao servia local e remoto, entao abrir o
// localhost mandava o browser para o dominio publico, e publicar exigia
// rebuildar o frontend sempre que o endereco mudasse. Com caminho relativo um
// build so serve os dois casos.

export const STORAGE_KEYS = {
  token: "chosen_token",
  refresh: "chosen_refresh",
  user: "chosen_user",
  branch: "chosen_branch",
  tenant: "chosen_tenant",
};

// ---- Multi-igreja (identidade global) ----

/** Vinculo da identidade com uma igreja. */
export interface Membership {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  role: string;
  branch_id: string;
  is_active: boolean;
}

/** Igreja oferecida no seletor apos o login. */
export interface TenantOption {
  id: string;
  name: string;
  slug: string;
  role: string;
}

/** Branding publico da igreja (tela de login). */
export interface PublicTenant {
  name: string;
  slug: string;
  logo_url?: string | null;
  brand_color?: string | null;
  favicon_url?: string | null;
}

/** Subdominios que NAO sao igreja. */
const RESERVED_SLUGS = new Set(["app", "www", "api", "admin", "localhost"]);

/**
 * Extrai o slug da igreja do host atual (subdominio). O dominio central
 * (`app.dominio`) e o localhost nao tem slug. Depende de pelo menos 3 rotulos
 * (ex.: `igreja.dominio.com`); o dominio base pode ser fixado em
 * NEXT_PUBLIC_BASE_DOMAIN.
 */
export function tenantSlugFromHost(): string {
  if (typeof window === "undefined") return "";
  const host = window.location.hostname.toLowerCase();
  const base = process.env.NEXT_PUBLIC_BASE_DOMAIN?.toLowerCase();
  if (base && host !== base && host.endsWith(`.${base}`)) {
    const sub = host.slice(0, -(base.length + 1));
    return sub.includes(".") ? "" : sub;
  }
  const parts = host.split(".");
  if (parts.length < 3) return "";
  const slug = parts[0];
  return RESERVED_SLUGS.has(slug) ? "" : slug;
}

/** Tenant ativo (referencia local; a autoridade e o JWT). */
export function getTenantContext(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(STORAGE_KEYS.tenant) ?? "";
}
export function setTenantContext(tenantID: string) {
  if (typeof window === "undefined") return;
  if (tenantID) localStorage.setItem(STORAGE_KEYS.tenant, tenantID);
  else localStorage.removeItem(STORAGE_KEYS.tenant);
}

/** Contexto de filial ativo: "" (padrao do token), "all" (Sede) ou um uuid. */
export function getBranchContext(): string {
  if (typeof window === "undefined") return "";
  return localStorage.getItem(STORAGE_KEYS.branch) ?? "";
}
export function setBranchContext(branchID: string) {
  if (typeof window === "undefined") return;
  if (branchID) localStorage.setItem(STORAGE_KEYS.branch, branchID);
  else localStorage.removeItem(STORAGE_KEYS.branch);
}
// Enviado em toda requisicao; o backend so honra para super_admin/admin_sede.
function branchHeader(): Record<string, string> {
  const v = getBranchContext();
  return v ? { "X-Branch-Id": v } : {};
}

// ---- Tipos ----
export interface User {
  id: string;
  email: string;
  full_name: string;
  tenant_id: string;
  branch_id: string;
  role: string;
  // Opcional em runtime: o cache do localStorage pode ter sido gravado por uma
  // versao anterior do app. O backend sempre devolve o array no login e no /me.
  permissions?: string[];
  mfa_enabled?: boolean;
  /** Igrejas da identidade (para o seletor/troca de igreja). */
  memberships?: Membership[];
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface Member {
  id: string;
  branch_id: string;
  first_name: string;
  last_name: string;
  full_name: string;
  nickname?: string;
  email?: string;
  phone?: string;
  whatsapp?: string;
  birth_date?: string;
  gender?: string;
  marital_status?: string;
  membership_status: string;
  /** Derivado no backend da situacao: "professo" quando ativo, senao "nao_professo". */
  roll_class?: string;
  profession?: string;
  /** Legado: virou member_cargos (a API devolve os dois). */
  office?: string;
  cpf?: string;
  rg?: string;
  baptism_date?: string;
  baptism_location?: string;
  joined_at?: string;
  /** Endereco do membro (requisito 1.1). */
  address?: MemberAddress;
  /** Motivo da baixa, quando a situacao e baixa/transferencia/falecimento. */
  exit_reason?: string;
  exited_at?: string;
  /** Data de casamento (aniversarios de casamento). */
  marriage_date?: string;
  photo_url?: string;
  created_at: string;
  /** Cargos ativos do membro, para os badges do grid. */
  cargos?: MemberCargoRef[];
  /** Numero da carteirinha ja emitida; ausente quando ainda nao houve emissao. */
  card_ref?: string;
}

/** Endereco do membro (jsonb no backend). */
export interface MemberAddress {
  zip_code?: string;
  street?: string;
  number?: string;
  complement?: string;
  district?: string;
  city?: string;
  state?: string;
}

/** Evento do historico eclesiastico do membro (requisito 1.8). */
export interface MemberHistory {
  id: string;
  member_id: string;
  occurred_at: string;
  kind: string;
  notes?: string;
  created_by?: string;
  created_at: string;
}


/** Cargo ativo resumido, como vem dentro do membro. */
export interface MemberCargoRef {
  id: string;
  name: string;
  kind: string;
  ends_at?: string;
}

/** Item do catalogo de cargos (customizavel pela igreja). */
export interface Cargo {
  id: string;
  branch_id?: string;
  name: string;
  slug: string;
  kind: string;
  requires_term: boolean;
  is_active: boolean;
  sort_order: number;
  created_at: string;
}

/** Mandato de um membro em um cargo (data de inicio, vencimento e situacao). */
export interface MemberCargo {
  id: string;
  member_id: string;
  cargo_id: string;
  cargo_name: string;
  cargo_kind: string;
  started_at?: string;
  ends_at?: string;
  status: "ativo" | "encerrado";
  notes?: string;
}

export interface Category {
  id: string;
  branch_id?: string;
  type: "income" | "expense";
  code: string;
  name: string;
  is_active: boolean;
}

export interface BankAccount {
  id: string;
  branch_id?: string;
  name: string;
  bank?: string;
  bank_code?: string;
  agency?: string;
  account_number?: string;
  account_type: "checking" | "savings" | "cash";
  initial_balance: number;
  is_active: boolean;
  created_at: string;
}

export interface Transaction {
  id: string;
  branch_id: string;
  category_id?: string;
  category_name?: string;
  account_id?: string;
  account_name?: string;
  type: "income" | "expense";
  amount: number;
  currency: string;
  payment_method?: string;
  is_anonymous: boolean;
  description?: string;
  receipt_issued: boolean;
  receipt_id?: string;
  receipt_ref?: string;
  receipt_token?: string;
  donor_member_id?: string;
  benefactor_id?: string;
  supplier_id?: string;
  supplier_name?: string;
  hash: string;
  occurred_at: string;
  /** Preenchido quando o lancamento foi estornado (continua no historico). */
  voided_at?: string;
  void_reason?: string;
  attachment_count?: number;
  attachment_url?: string;
}

export interface FinancialAttachment {
  id: string;
  transaction_id: string;
  file_name: string;
  file_url: string;
  content_type?: string;
  file_size: number;
  created_at: string;
}

export interface MonthlyPoint {
  month: string;
  income: number;
  expense: number;
  net: number;
}
export interface DRELine {
  category_id: string;
  category: string;
  type: string;
  total: number;
}
export interface DRE {
  from: string;
  to: string;
  income: number;
  expense: number;
  net: number;
  lines: DRELine[];
  comparison?: { prev_income: number; prev_expense: number; prev_net: number; delta_pct: number };
}
export interface Balance {
  income: number;
  expense: number;
  net: number;
  by_category: { category_id: string; category: string; type: string; total: number }[];
}
export interface Relationship {
  id: string;
  related_id: string;
  related_name: string;
  kind: string;
  relation: string;
}
export interface Family {
  id: string;
  name: string;
  /** Codigo de exibicao no padrao da planilha do cliente (#001, #002...). */
  code?: string;
  head_id?: string;
  head_name?: string;
  branch_id: string;
  address?: FamilyAddress;
  member_count: number;
}

export interface FamilyAddress {
  street?: string;
  number?: string;
  complement?: string;
  district?: string;
  city?: string;
  state?: string;
  zip?: string;
}

/** Membro dentro de uma familia; "relation" e o parentesco com o chefe. */
export interface FamilyMemberRow {
  id: string;
  full_name: string;
  relation: string;
  kind?: string;
  is_head: boolean;
  membership_status: string;
  phone?: string;
  whatsapp?: string;
  email?: string;
  birth_date?: string;
  photo_url?: string;
}
export interface Visitor {
  id: string;
  branch_id: string;
  first_name: string;
  last_name: string;
  full_name: string;
  email?: string;
  phone?: string;
  whatsapp?: string;
  source?: string;
  journey_stage: string;
  created_at: string;
}
export interface Benefactor {
  id: string;
  branch_id?: string;
  name: string;
  email?: string;
  phone?: string;
  notes?: string;
  created_at: string;
}

export interface Supplier {
  id: string;
  branch_id?: string;
  name: string;
  trade_name?: string;
  cpf?: string;
  cnpj?: string;
  email?: string;
  phone?: string;
  notes?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

// ---- Pessoas Unificadas ----
export interface Person {
  id: string;
  type: "member" | "visitor" | "benefactor";
  full_name: string;
  first_name: string;
  last_name: string;
  email?: string;
  phone?: string;
  whatsapp?: string;
  cpf?: string;
  rg?: string;
  birth_date?: string;
  age?: number;
  gender?: string;
  marital_status?: string;
  membership_status?: string;
  roll_class?: string;
  office?: string;
  profession?: string;
  nationality?: string;
  education?: string;
  baptism_date?: string;
  baptism_location?: string;
  joined_at?: string;
  address?: MemberAddress;
  exit_reason?: string;
  exited_at?: string;
  marriage_date?: string;
  source?: string;
  journey_stage?: string;
  notes?: string;
  branch_name?: string;
  family_name?: string;
  created_at: string;
  updated_at: string;
  tags: string[];
}

export interface PeopleQuery {
  type?: "member" | "visitor" | "benefactor" | "all";
  status?: string;
  q?: string;
  age_min?: number;
  age_max?: number;
  gender?: string;
  marital_status?: string;
  branch_id?: string;
  page?: number;
  page_size?: number;
  sort?: string;
  order?: "asc" | "desc";
}

export interface PeopleResponse {
  people: Person[];
  total: number;
  page: number;
  page_size: number;
}

export const listPeople = (params: PeopleQuery = {}) => {
  const q = new URLSearchParams();
  if (params.type && params.type !== "all") q.set("type", params.type);
  if (params.status) q.set("status", params.status);
  if (params.q) q.set("q", params.q);
  if (params.age_min) q.set("age_min", String(params.age_min));
  if (params.age_max) q.set("age_max", String(params.age_max));
  if (params.gender) q.set("gender", params.gender);
  if (params.marital_status) q.set("marital_status", params.marital_status);
  if (params.branch_id) q.set("branch_id", params.branch_id);
  if (params.page) q.set("page", String(params.page));
  if (params.page_size) q.set("page_size", String(params.page_size));
  if (params.sort) q.set("sort", params.sort);
  if (params.order) q.set("order", params.order);
  const queryString = q.toString();
  return api<PeopleResponse>(`/api/v1/people${queryString ? `?${queryString}` : ""}`);
};

export const createPerson = (type: Person["type"], data: Record<string, unknown>) =>
  api<Person>(`/api/v1/people/${type}`, { method: "POST", body: JSON.stringify(data) });

export const updatePerson = (type: Person["type"], id: string, data: Record<string, unknown>) =>
  api<Person>(`/api/v1/people/${type}/${id}`, { method: "PATCH", body: JSON.stringify(data) });

export interface Delivery {
  id: string;
  document_id: string;
  channel: string;
  recipient: string;
  status: string;
  error?: string;
  attempts: number;
  sent_at?: string;
  created_at: string;
}

// ---- Sessao (module-level) ----
let accessToken: string | null = null;
let refreshToken: string | null = null;
let onSessionLost: (() => void) | null = null;

// Restaura a sessao ja no carregamento do modulo (browser), antes de qualquer
// efeito. Sem isto, a primeira chamada disparada por uma pagina (ex.: /branches
// no layout) pode sair sem Bearer - efeitos de filhos rodam antes do
// AuthProvider - e devolver 401.
if (typeof window !== "undefined") {
  accessToken = localStorage.getItem(STORAGE_KEYS.token) ?? null;
  refreshToken = localStorage.getItem(STORAGE_KEYS.refresh) ?? null;
}

export function registerSessionLost(fn: () => void) {
  onSessionLost = fn;
}
export function hasSession() {
  return !!accessToken;
}
export function setTokens(tokens: Tokens) {
  accessToken = tokens.access_token;
  refreshToken = tokens.refresh_token;
  localStorage.setItem(STORAGE_KEYS.token, tokens.access_token);
  localStorage.setItem(STORAGE_KEYS.refresh, tokens.refresh_token);
}
export function restoreTokens() {
  accessToken = localStorage.getItem(STORAGE_KEYS.token) ?? null;
  refreshToken = localStorage.getItem(STORAGE_KEYS.refresh) ?? null;
}
export function clearTokens() {
  accessToken = null;
  refreshToken = null;
  localStorage.removeItem(STORAGE_KEYS.token);
  localStorage.removeItem(STORAGE_KEYS.refresh);
}

// ---- Core fetch com auto-refresh ----
async function tryRefresh(): Promise<boolean> {
  if (!refreshToken) return false;
  const res = await fetch("/api/v1/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!res.ok) return false;
  const data = (await res.json()) as { tokens: Tokens };
  setTokens(data.tokens);
  return true;
}

async function api<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const doFetch = (): Promise<Response> =>
    fetch(path, {
      ...opts,
      headers: {
        "Content-Type": "application/json",
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        ...branchHeader(),
        ...(opts.headers ?? {}),
      },
    });

  let res = await doFetch();
  if (res.status === 401 && refreshToken) {
    if (await tryRefresh()) {
      res = await doFetch();
    } else {
      clearTokens();
      onSessionLost?.();
    }
  }
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `HTTP ${res.status}`);
  }
  const text = await res.text();
  return text ? (JSON.parse(text) as T) : ({} as T);
}

// apiRaw e como api() mas devolve o Response RAW (sem fazer JSON.parse).
// Util para respostas text/html, multipart upload (FormData) e blobs (CSV/PDF).
// Nao impoe Content-Type: application/json por padrao - o chamador ou o browser
// definem o content-type apropriado (ex.: boundary para FormData).
async function apiRaw(path: string, opts: RequestInit = {}): Promise<Response> {
  const doFetch = (): Promise<Response> =>
    fetch(path, {
      ...opts,
      headers: {
        ...(opts.headers ?? {}),
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        ...branchHeader(),
      },
    });

  let res = await doFetch();
  if (res.status === 401 && refreshToken) {
    if (await tryRefresh()) {
      res = await doFetch();
    } else {
      clearTokens();
      onSessionLost?.();
    }
  }
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `HTTP ${res.status}`);
  }
  return res;
}

// ---- Auth / sessao ----

/** Login direto (tokens + user) ou pedido de selecao de igreja. */
export type LoginResponse =
  | { tokens: Tokens; user: User }
  | { requires_tenant_selection: true; selection_token: string; tenants: TenantOption[] };

export function login(email: string, password: string, code?: string, tenantSlug?: string) {
  return api<LoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({
      email,
      password,
      ...(code ? { code } : {}),
      ...(tenantSlug ? { tenant_slug: tenantSlug } : {}),
    }),
  });
}

export function selectTenant(selectionToken: string, tenantId: string) {
  return api<{ tokens: Tokens; user: User }>("/api/v1/auth/select-tenant", {
    method: "POST",
    body: JSON.stringify({ selection_token: selectionToken, tenant_id: tenantId }),
  });
}

export function switchTenant(tenantId: string) {
  return api<{ tokens: Tokens; user: User }>("/api/v1/auth/switch-tenant", {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId }),
  });
}

export function fetchMe() {
  return api<User>("/api/v1/me");
}

export const listMyTenants = () => api<{ tenants: Membership[] }>("/api/v1/me/tenants");

export const getPublicTenant = (slug: string) =>
  api<PublicTenant>(`/api/v1/public/tenant/${encodeURIComponent(slug)}`);

// ---- Pessoas ----
export const listMembers = () => api<{ members: Member[] }>("/api/v1/members");
export const getMember = (id: string) => api<Member>(`/api/v1/members/${id}`);
export const createMember = (data: Record<string, unknown>) =>
  api<Member>("/api/v1/members", { method: "POST", body: JSON.stringify(data) });
export const updateMember = (id: string, data: Record<string, unknown>) =>
  api<Member>(`/api/v1/members/${id}`, { method: "PATCH", body: JSON.stringify(data) });

// ---- Historico eclesiastico do membro (requisito 1.8) ----
export const listMemberHistory = (memberId: string) =>
  api<{ history: MemberHistory[] }>(`/api/v1/members/${memberId}/history`);
export const addMemberHistory = (memberId: string, data: Record<string, unknown>) =>
  api<MemberHistory>(`/api/v1/members/${memberId}/history`, {
    method: "POST",
    body: JSON.stringify(data),
  });
export const getMemberTree = (id: string) =>
  api<{ member: Member; relationships: Relationship[] }>(`/api/v1/members/${id}/tree`);
export const addRelationship = (id: string, relateMemberId: string, kind: string) =>
  api<{ ok: boolean }>(`/api/v1/members/${id}/relationships`, {
    method: "POST",
    body: JSON.stringify({ relate_member_id: relateMemberId, kind }),
  });
export interface CardResult {
  card_ref: string;
  /** Token do QR: com ele o front monta /member/{token}. */
  token: string;
  member: string;
}

/** Emite a carteirinha. E idempotente: repetir devolve a MESMA carteirinha. */
export const issueCard = (memberId: string) =>
  api<CardResult>(`/api/v1/members/${memberId}/card`, { method: "POST" });

/** Le a carteirinha ja emitida, sem emitir (404 quando nao existe). */
export const getCard = (memberId: string) => api<CardResult>(`/api/v1/members/${memberId}/card`);

// ---- Foto do membro ----
// Multipart: apiRaw nao forca Content-Type, deixando o browser definir o
// boundary, e preserva o auto-refresh do token em caso de 401.
export async function uploadMemberPhoto(memberId: string, file: File) {
  const form = new FormData();
  form.append("file", file);
  const res = await apiRaw(`/api/v1/members/${memberId}/photo`, { method: "POST", body: form });
  const text = await res.text();
  return (text ? JSON.parse(text) : {}) as { id: string; photo_url?: string };
}

export const deleteMemberPhoto = (memberId: string) =>
  api<{ ok: boolean }>(`/api/v1/members/${memberId}/photo`, { method: "DELETE" });

// ---- Cargos (funcoes/ministerios) ----
export const listCargos = () => api<{ cargos: Cargo[] }>("/api/v1/cargos");
export const createCargo = (data: Partial<Cargo>) =>
  api<Cargo>("/api/v1/cargos", { method: "POST", body: JSON.stringify(data) });
export const updateCargo = (id: string, data: Partial<Cargo>) =>
  api<Cargo>(`/api/v1/cargos/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteCargo = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/cargos/${id}`, { method: "DELETE" });

// ---- Mandatos do membro ----
export const listMemberCargos = (memberId: string) =>
  api<{ cargos: MemberCargo[] }>(`/api/v1/members/${memberId}/cargos`);
export const assignCargo = (memberId: string, data: Record<string, unknown>) =>
  api<MemberCargo>(`/api/v1/members/${memberId}/cargos`, {
    method: "POST",
    body: JSON.stringify(data),
  });
export const updateMemberCargo = (memberId: string, linkId: string, data: Record<string, unknown>) =>
  api<MemberCargo>(`/api/v1/members/${memberId}/cargos/${linkId}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
export const unassignCargo = (memberId: string, linkId: string) =>
  api<{ ok: boolean }>(`/api/v1/members/${memberId}/cargos/${linkId}`, { method: "DELETE" });

/**
 * Sincroniza os cargos do membro com a selecao do formulario.
 *
 * Duas decisoes que valem explicacao:
 *  - Desmarcar NAO apaga o mandato: encerra (status 'encerrado', vencimento hoje).
 *    O historico de quem exerceu cada cargo e justamente o que o requisito 1.4
 *    protege, e o backend recusa (409) apagar um cargo com mandatos por isso.
 *  - Remarcar REATIVA o mandato existente em vez de inserir outro: o indice
 *    unico (member_id, cargo_id, started_at) rejeitaria uma segunda linha com a
 *    mesma data de inicio. As datas do mandato podem ser ajustadas depois na
 *    aba Cargos, que e onde o historico completo aparece.
 */
export async function syncMemberCargos(memberId: string, cargoIds: string[]): Promise<void> {
  const { cargos: atuais } = await listMemberCargos(memberId);
  const desejados = new Set(cargoIds);
  const hoje = new Date().toISOString().slice(0, 10);

  // A lista ja vem com os ativos primeiro e o mandato mais recente antes dos
  // antigos: a primeira ocorrencia de cada cargo e o vinculo que vale hoje.
  const vigente = new Map<string, MemberCargo>();
  for (const mc of atuais) {
    if (!vigente.has(mc.cargo_id)) vigente.set(mc.cargo_id, mc);
  }

  for (const [cargoId, mc] of vigente) {
    if (desejados.has(cargoId)) {
      if (mc.status !== "ativo") {
        await updateMemberCargo(memberId, mc.id, { status: "ativo", ends_at: "" });
      }
    } else if (mc.status === "ativo") {
      await updateMemberCargo(memberId, mc.id, { status: "encerrado", ends_at: hoje });
    }
  }

  for (const cargoId of desejados) {
    if (vigente.has(cargoId)) continue;
    await assignCargo(memberId, { cargo_id: cargoId, started_at: hoje, status: "ativo" });
  }
}

// ---- Familias ----
export const listFamilies = () => api<{ families: Family[] }>("/api/v1/families");
export const getFamily = (id: string) => api<Family>(`/api/v1/families/${id}`);
export const createFamily = (name: string) =>
  api<Family>("/api/v1/families", { method: "POST", body: JSON.stringify({ name }) });
export const updateFamily = (id: string, data: Record<string, unknown>) =>
  api<Family>(`/api/v1/families/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteFamily = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/families/${id}`, { method: "DELETE" });
export const listFamilyMembers = (familyId: string) =>
  api<{ members: FamilyMemberRow[] }>(`/api/v1/families/${familyId}/members`);
export const addFamilyMember = (familyId: string, memberId: string, relateId: string, relation: string) =>
  api<{ ok: boolean }>(`/api/v1/families/${familyId}/members`, {
    method: "POST",
    body: JSON.stringify({ member_id: memberId, relate_id: relateId, relation }),
  });
export const removeFamilyMember = (familyId: string, memberId: string) =>
  api<{ ok: boolean }>(`/api/v1/families/${familyId}/members/${memberId}`, { method: "DELETE" });
export const listMemberFamilies = (memberId: string) =>
  api<{ families: Family[] }>(`/api/v1/members/${memberId}/families`);

// ---- Visitantes / Benfeitores ----
export const listVisitors = () => api<{ visitors: Visitor[] }>("/api/v1/visitors");
export const createVisitor = (data: Record<string, unknown>) =>
  api<Visitor>("/api/v1/visitors", { method: "POST", body: JSON.stringify(data) });
export const updateVisitorStage = (id: string, stage: string) =>
  api<Visitor>(`/api/v1/visitors/${id}/stage`, { method: "PATCH", body: JSON.stringify({ stage }) });
export const convertVisitorToMember = (visitorId: string, data: Record<string, unknown>) =>
  api<Member>(`/api/v1/visitors/${visitorId}/convert`, { method: "POST", body: JSON.stringify(data) });
export const revertVisitorConversion = (visitorId: string) =>
  api<{ ok: boolean }>(`/api/v1/visitors/${visitorId}/revert`, { method: "POST" });

// ---- Busca de pessoas para ministry multi-select ----
export const searchMembers = (q: string) =>
  api<{ members: Member[] }>(`/api/v1/members?q=${encodeURIComponent(q)}`);

export const listBenefactors = () => api<{ benefactors: Benefactor[] }>("/api/v1/benefactors");
export const createBenefactor = (data: Record<string, unknown>) =>
  api<Benefactor>("/api/v1/benefactors", { method: "POST", body: JSON.stringify(data) });

// ---- Fornecedores ----
export const listSuppliers = (q = "") =>
  api<{ suppliers: Supplier[] }>(`/api/v1/suppliers${q ? `?q=${encodeURIComponent(q)}` : ""}`);
export const createSupplier = (data: Record<string, unknown>) =>
  api<Supplier>("/api/v1/suppliers", { method: "POST", body: JSON.stringify(data) });
export const updateSupplier = (id: string, data: Record<string, unknown>) =>
  api<Supplier>(`/api/v1/suppliers/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteSupplier = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/suppliers/${id}`, { method: "DELETE" });

// ---- Financeiro ----
export const listCategories = () => api<{ categories: Category[] }>("/api/v1/finance/categories");
export const createCategory = (data: Record<string, unknown>) =>
  api<Category>("/api/v1/finance/categories", { method: "POST", body: JSON.stringify(data) });
export const updateCategory = (id: string, data: Record<string, unknown>) =>
  api<Category>(`/api/v1/finance/categories/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteCategory = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/finance/categories/${id}`, { method: "DELETE" });
export const listAccounts = () => api<{ accounts: BankAccount[] }>("/api/v1/finance/accounts");
export const createAccount = (data: Record<string, unknown>) =>
  api<BankAccount>("/api/v1/finance/accounts", { method: "POST", body: JSON.stringify(data) });
export const updateAccount = (id: string, data: Record<string, unknown>) =>
  api<BankAccount>(`/api/v1/finance/accounts/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteAccount = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/finance/accounts/${id}`, { method: "DELETE" });

// ---- Auditoria financeira ----
export interface FinancialAudit {
  id: string;
  branch_id: string;
  title: string;
  period_start: string;
  period_end: string;
  status: "aberta" | "fechada";
  notes?: string;
  closed_at?: string;
  signer_name?: string;
  signer_role?: string;
  signature_hash?: string;
  created_at: string;
  total_items: number;
  audited_items: number;
  total_amount: number;
  audited_amount: number;
}
export interface AuditAllocation { event_id: string; event_name: string; amount: number; }
export interface AuditItem {
  transaction_id: string;
  occurred_at: string;
  type: "income" | "expense";
  amount: number;
  currency: string;
  description?: string;
  category_name?: string;
  account_name?: string;
  supplier_name?: string;
  payment_method?: string;
  is_anonymous: boolean;
  donor_name?: string;
  receipt_ref?: string;
  attachment_count: number;
  allocations: AuditAllocation[];
  audited: boolean;
  audited_at?: string;
  notes?: string;
}
export const listAudits = () => api<{ audits: FinancialAudit[] }>("/api/v1/finance/audits");
export const createAudit = (data: Record<string, unknown>) =>
  api<FinancialAudit>("/api/v1/finance/audits", { method: "POST", body: JSON.stringify(data) });
export const getAudit = (id: string) =>
  api<{ audit: FinancialAudit; items: AuditItem[] }>(`/api/v1/finance/audits/${id}`);
export const markAudit = (id: string, data: Record<string, unknown>) =>
  api<{ ok: boolean }>(`/api/v1/finance/audits/${id}/mark`, { method: "POST", body: JSON.stringify(data) });
export const closeAudit = (id: string, data: Record<string, unknown>) =>
  api<FinancialAudit>(`/api/v1/finance/audits/${id}/close`, { method: "POST", body: JSON.stringify(data) });
export const deleteAudit = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/finance/audits/${id}`, { method: "DELETE" });

// ---- Demonstrativo para assembleia ----
export interface AssemblyCategory { category_id: string; category: string; type: string; total: number; }
export interface AssemblyBalance { income: number; expense: number; net: number; by_category: AssemblyCategory[]; }
export interface AssemblyReport { from: string; to: string; balance: AssemblyBalance; months: MonthlyPoint[]; }
export const getAssemblyReport = (from: string, to: string) => {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  return api<AssemblyReport>(`/api/v1/reports/assembly?${q.toString()}`);
};
export const listTransactions = (type = "") => {
  const q = type ? `?type=${encodeURIComponent(type)}` : "";
  return api<{ transactions: Transaction[] }>(`/api/v1/finance/transactions${q}`);
};
export const createTransaction = (data: Record<string, unknown>) =>
  api<{ transaction: Transaction; receipt_ref: string; receipt_token: string }>("/api/v1/finance/transactions", {
    method: "POST",
    body: JSON.stringify(data),
  });

// ---- Importacao em lote de lancamentos (CSV/planilha) ----
export interface ImportResult {
  imported: number;
  skipped: number;
  errors: { line: number; error: string }[];
}
export const importTransactions = (csv: string) =>
  api<ImportResult>("/api/v1/finance/transactions/import", {
    method: "POST",
    body: JSON.stringify({ csv }),
  });

/** Estorna um lancamento (append-only: nao apaga, marca como anulado). */
export const voidTransaction = (id: string, reason = "") =>
  api<{ ok: boolean }>(`/api/v1/finance/transactions/${id}/void`, {
    method: "POST",
    body: JSON.stringify({ reason }),
  });

// ---- Rateio do lancamento por evento ----
export interface EventAllocation {
  id: string;
  event_id: string;
  event_name: string;
  amount: number;
}
export const listTransactionEvents = (id: string) =>
  api<{ allocations: EventAllocation[] }>(`/api/v1/finance/transactions/${id}/events`);

// Importacao com mapeamento de colunas (CSV/XLSX).
export const previewTransactions = (data: string, filename: string) =>
  api<{ rows: string[][] }>("/api/v1/finance/transactions/import/preview", {
    method: "POST",
    body: JSON.stringify({ data, filename }),
  });
export const importTransactionsFile = (
  data: string,
  filename: string,
  startRow: number,
  mapping: Record<string, number>,
) =>
  api<ImportResult>("/api/v1/finance/transactions/import", {
    method: "POST",
    body: JSON.stringify({ data, filename, start_row: startRow, mapping }),
  });

// ---- Anexos de lancamentos ----
// Upload usa multipart/form-data (nao JSON) - apiRaw nao forca Content-Type,
// deixando o browser definir o boundary automaticamente, e mantem o
// auto-refresh de token em caso de 401.
export async function uploadAttachment(transactionId: string, file: File): Promise<FinancialAttachment> {
  const form = new FormData();
  form.append("file", file);
  const res = await apiRaw(`/api/v1/finance/transactions/${transactionId}/attachments`, {
    method: "POST",
    body: form,
  });
  const text = await res.text();
  return text ? (JSON.parse(text) as FinancialAttachment) : ({} as FinancialAttachment);
}
export const listAttachments = (transactionId: string) =>
  api<{ attachments: FinancialAttachment[] }>(`/api/v1/finance/transactions/${transactionId}/attachments`);
export const getBalance = () => api<Balance>("/api/v1/finance/balance");

// ---- Relatorios ----
export const getMonthlyBalance = (from = "", to = "") => {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  return api<{ series: MonthlyPoint[] }>(`/api/v1/reports/balance?${q}`);
};
export const getDRE = (from = "", to = "") => {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  return api<DRE>(`/api/v1/reports/dre?${q}`);
};

// ---- Relatorios de pessoas ----
export interface Birthday {
  id: string;
  full_name: string;
  birth_date: string;
  day: number;
  age: number;
  whatsapp?: string;
  phone?: string;
  branch_id: string;
}
export interface MarriageAnniversary {
  id: string;
  full_name: string;
  spouse_name?: string;
  marriage_date: string;
  day: number;
  years: number;
}
export interface BirthdaysResult {
  month: number;
  birthdays: Birthday[];
  marriages: MarriageAnniversary[];
}
export interface PyramidRow {
  bucket: string;
  male: number;
  female: number;
  total: number;
}
export interface CountRow {
  key: string;
  count: number;
}
export interface Demographics {
  total: number;
  age_pyramid: PyramidRow[];
  by_gender: CountRow[];
  by_marital_status: CountRow[];
  by_status: CountRow[];
  by_state: CountRow[];
  by_city: CountRow[];
}
export const getBirthdays = (month?: number) =>
  api<BirthdaysResult>(`/api/v1/reports/birthdays${month ? `?month=${month}` : ""}`);
export const getDemographics = () => api<Demographics>("/api/v1/reports/demographics");

// ---- Demonstrativo Mensal (regime de caixa) ----
export interface WeekRange {
  label: string;
  from: string;
  to: string;
}
export interface StatementLine {
  category_id: string;
  code: string;
  name: string;
  weeks: number[];
  total: number;
}
export interface MonthlyStatement {
  month: string;
  from: string;
  to: string;
  weeks: WeekRange[];
  income: StatementLine[];
  expense: StatementLine[];
  total_income: number;
  total_expense: number;
  opening_balance: number;
  closing_balance: number;
}
export const getMonthlyStatement = (month?: string) =>
  api<MonthlyStatement>(`/api/v1/reports/monthly-statement${month ? `?month=${month}` : ""}`);

// ---- Painel consolidado Sede > Filiais ----
export interface ConsolidatedBranch {
  id: string;
  name: string;
  kind: string;
  parent_id?: string;
  member_count: number;
  visitor_count: number;
  income: number;
  expense: number;
  net: number;
}
export interface ConsolidatedTotals {
  member_count: number;
  visitor_count: number;
  income: number;
  expense: number;
  net: number;
}
export interface ConsolidatedResult {
  branches: ConsolidatedBranch[];
  totals: ConsolidatedTotals;
}
export const getConsolidated = (from = "", to = "") => {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  const qs = q.toString();
  return api<ConsolidatedResult>(`/api/v1/reports/consolidated${qs ? `?${qs}` : ""}`);
};

// ---- Recibos ----
// Recebemos text/html (nao JSON), entao usamos apiRaw que preserva o
// auto-refresh de token sem tentar fazer JSON.parse.
export async function getReceiptHTML(documentId: string): Promise<string> {
  const res = await apiRaw(`/api/v1/receipts/${documentId}`, {
    headers: { "Content-Type": "application/json" },
  });
  return res.text();
}
export const sendReceipt = (documentId: string, channel: string, recipient: string) =>
  api<Delivery>(`/api/v1/receipts/${documentId}/send`, {
    method: "POST",
    body: JSON.stringify({ channel, recipient }),
  });
export const listDeliveries = (documentId: string) =>
  api<{ deliveries: Delivery[] }>(`/api/v1/receipts/${documentId}/deliveries`);

// ---- Branches ----
export interface Branch {
  id: string;
  name: string;
  slug: string;
  kind: string;
  cnpj?: string;
  parent_id?: string;
  address?: Record<string, unknown>;
  geo?: { lat?: number; lng?: number };
  is_active?: boolean;
  member_count?: number;
  created_at?: string;
  /** Canais: indica no grid se a filial tem WhatsApp conectado. */
  whatsapp_status?: "disconnected" | "connecting" | "connected";
  whatsapp_phone?: string;
  /** Numero efetivamente conectado (preenchido quando a conexao abre). */
  whatsapp_number?: string;
}
export const listBranches = () => api<{ branches: Branch[] }>("/api/v1/branches");
export const createBranch = (data: Record<string, unknown>) =>
  api<Branch>("/api/v1/branches", { method: "POST", body: JSON.stringify(data) });
export const updateBranch = (id: string, data: Record<string, unknown>) =>
  api<Branch>(`/api/v1/branches/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteBranch = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/branches/${id}`, { method: "DELETE" });

// ---- Canais por filial (WhatsApp Evolution + SMTP) ----
export interface BranchChannels {
  branch_id: string;
  whatsapp_phone: string;
  whatsapp_instance: string;
  whatsapp_status: "disconnected" | "connecting" | "connected";
  whatsapp_number: string;
  smtp_host: string;
  smtp_port: number;
  smtp_user: string;
  smtp_password_set: boolean;
  smtp_from: string;
  smtp_from_name: string;
  smtp_secure: boolean;
}
export const getBranchChannels = (id: string) =>
  api<BranchChannels>(`/api/v1/branches/${id}/channels`);
export const updateBranchChannels = (id: string, data: Record<string, unknown>) =>
  api<BranchChannels>(`/api/v1/branches/${id}/channels`, { method: "PATCH", body: JSON.stringify(data) });
export interface WhatsAppConnectResult {
  instance: string;
  status: string;
  number?: string;
  qrcode_base64: string;
  code?: string;
}
export const connectBranchWhatsApp = (id: string) =>
  api<WhatsAppConnectResult>(`/api/v1/branches/${id}/whatsapp/connect`, { method: "POST" });
export const getBranchWhatsAppState = (id: string) =>
  api<{ instance: string; status: string; connected: boolean; number?: string }>(`/api/v1/branches/${id}/whatsapp/state`);
export const disconnectBranchWhatsApp = (id: string) =>
  api<{ ok: boolean; status: string }>(`/api/v1/branches/${id}/whatsapp/logout`, { method: "POST" });

// ---- Dados da igreja (tenant) ----
export interface Tenant {
  id: string;
  name: string;
  slug: string;
  legal_name?: string;
  cnpj?: string;
  plan: string;
  locale: string;
  timezone: string;
  /** White-label: branding aplicado no login do subdominio. */
  logo_url?: string | null;
  brand_color?: string | null;
  favicon_url?: string | null;
  custom_domain?: string | null;
  is_active: boolean;
  updated_at: string;
}
export const getTenant = () => api<Tenant>("/api/v1/tenant");
export const updateTenant = (data: Record<string, unknown>) =>
  api<Tenant>("/api/v1/tenant", { method: "PATCH", body: JSON.stringify(data) });

// ---- Perfil do proprio usuario ----
export const updateProfile = (data: { full_name?: string; email?: string }) =>
  api<User>("/api/v1/me", { method: "PATCH", body: JSON.stringify(data) });
export const changePassword = (currentPassword: string, newPassword: string) =>
  api<{ ok: boolean }>("/api/v1/me/password", {
    method: "POST",
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });

// ---- Usuarios e acessos ----
export interface AdminUser {
  id: string;
  branch_id?: string;
  role: string;
  email: string;
  full_name: string;
  is_active: boolean;
  mfa_enabled: boolean;
  last_login_at?: string;
  created_at: string;
}
export interface RoleInfo {
  id: string;
  key: string;
  name: string;
  is_system: boolean;
  permissions: string[];
  user_count: number;
}
export interface PermissionInfo {
  key: string;
  module: string;
  name: string;
}

export const listUsers = () => api<{ users: AdminUser[] }>("/api/v1/users");
export const createUser = (data: Record<string, unknown>) =>
  api<AdminUser>("/api/v1/users", { method: "POST", body: JSON.stringify(data) });
export const updateUser = (id: string, data: Record<string, unknown>) =>
  api<AdminUser>(`/api/v1/users/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const resetUserPassword = (id: string, password: string) =>
  api<{ ok: boolean }>(`/api/v1/users/${id}/password`, {
    method: "POST",
    body: JSON.stringify({ password }),
  });
export const listRoles = () => api<{ roles: RoleInfo[] }>("/api/v1/roles");
export const listPermissions = () => api<{ permissions: PermissionInfo[] }>("/api/v1/permissions");

// ---- MFA (TOTP) do proprio usuario ----
export const mfaStatus = () => api<{ enabled: boolean }>("/api/v1/auth/mfa");
export const mfaSetup = () =>
  api<{ secret: string; otpauth_url: string; account: string }>("/api/v1/auth/mfa/setup", { method: "POST" });
export const mfaEnable = (code: string) =>
  api<{ ok: boolean; enabled: boolean }>("/api/v1/auth/mfa/enable", {
    method: "POST",
    body: JSON.stringify({ code }),
  });
export const mfaDisable = () =>
  api<{ ok: boolean; enabled: boolean }>("/api/v1/auth/mfa/disable", { method: "POST" });

// ---- Repasses ----
export interface Transfer {
  id: string;
  branch_id?: string;
  from_branch_id: string;
  from_branch: string;
  to_branch_id: string;
  to_branch: string;
  transaction_id?: string;
  amount: number;
  rule_name?: string;
  executed_at: string;
}
export const listTransfers = () => api<{ transfers: Transfer[] }>("/api/v1/finance/transfers");
export const createTransfer = (data: Record<string, unknown>) =>
  api<Transfer>("/api/v1/finance/transfers", { method: "POST", body: JSON.stringify(data) });

// ---- Doacoes recorrentes ----
export interface RecurringDonation {
  id: string;
  branch_id: string;
  member_id?: string;
  member_name?: string;
  benefactor_id?: string;
  category_id?: string;
  category_name?: string;
  account_id?: string;
  account_name?: string;
  subtype: string;
  amount: number;
  frequency: string;
  next_run_at: string;
  last_run_at?: string;
  times_run: number;
  is_active: boolean;
  payment_method?: string;
  description?: string;
  created_at: string;
}
export const listRecurring = () => api<{ recurring: RecurringDonation[] }>("/api/v1/finance/recurring");
export const createRecurring = (data: Record<string, unknown>) =>
  api<RecurringDonation>("/api/v1/finance/recurring", { method: "POST", body: JSON.stringify(data) });
export const updateRecurring = (id: string, data: Record<string, unknown>) =>
  api<RecurringDonation>(`/api/v1/finance/recurring/${id}`, { method: "PATCH", body: JSON.stringify(data) });

// ---- Ministerios / voluntarios ----
export interface Ministry {
  id: string;
  branch_id: string;
  name: string;
  slug: string;
  description?: string;
  leader_id?: string;
  leader_name?: string;
  is_active: boolean;
  created_at: string;
}
export interface MinistryMember {
  ministry_id: string;
  member_id: string;
  member_name: string;
  role: string;
  started_at?: string;
}
export const listMinistries = () => api<{ ministries: Ministry[] }>("/api/v1/ministries");
export const createMinistry = (data: Record<string, unknown>) =>
  api<Ministry>("/api/v1/ministries", { method: "POST", body: JSON.stringify(data) });
export const updateMinistry = (id: string, data: Record<string, unknown>) =>
  api<Ministry>(`/api/v1/ministries/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteMinistry = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/ministries/${id}`, { method: "DELETE" });
export const listMinistryMembers = (id: string) =>
  api<{ members: MinistryMember[] }>(`/api/v1/ministries/${id}/members`);
export const addMinistryMember = (id: string, memberId: string, role: string) =>
  api<{ ok: boolean }>(`/api/v1/ministries/${id}/members`, {
    method: "POST",
    body: JSON.stringify({ member_id: memberId, role }),
  });
export const removeMinistryMember = (ministryId: string, memberId: string) =>
  api<{ ok: boolean }>(`/api/v1/ministries/${ministryId}/members/${memberId}`, {
    method: "DELETE",
  });
export const addMultipleMinistryMembers = (id: string, members: { member_id: string; role: string }[]) =>
  api<{ ok: boolean }>(`/api/v1/ministries/${id}/members/batch`, {
    method: "POST",
    body: JSON.stringify({ members }),
  });

// ---- Grupos / celulas / check-in ----
export interface SmallGroup {
  id: string;
  branch_id: string;
  ministry_id?: string;
  name: string;
  kind: string;
  leader_id?: string;
  leader_name?: string;
  address?: string;
  max_members?: number;
  weekday?: number;
  meeting_time?: string;
  is_active: boolean;
  created_at: string;
}
export interface AttendanceCheckin {
  group_id: string;
  group_name: string;
  member_id?: string;
  member_name: string;
  attended_at: string;
  present: boolean;
}
export const listGroups = () => api<{ groups: SmallGroup[] }>("/api/v1/groups");
export const createGroup = (data: Record<string, unknown>) =>
  api<SmallGroup>("/api/v1/groups", { method: "POST", body: JSON.stringify(data) });
export const updateGroup = (id: string, data: Record<string, unknown>) =>
  api<SmallGroup>(`/api/v1/groups/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteGroup = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/groups/${id}`, { method: "DELETE" });
export const checkIn = (groupId: string, data: Record<string, unknown>) =>
  api<AttendanceCheckin>(`/api/v1/groups/${groupId}/attendance`, { method: "POST", body: JSON.stringify(data) });
export const listAttendance = (groupId: string) =>
  api<{ attendance: AttendanceCheckin[] }>(`/api/v1/groups/${groupId}/attendance`);

// ---- Avisos ----
export interface Announcement {
  id: string;
  branch_id: string;
  title: string;
  body: string;
  audience: string;
  audience_filter?: AudienceFilter;
  channel: string;
  schedule_type: "manual" | "once" | "daily" | "event";
  schedule_at?: string;
  schedule_time?: string;
  schedule_event_id?: string;
  schedule_offset_minutes: number;
  last_run_at?: string;
  run_count: number;
  is_active: boolean;
  published_at: string;
  created_at: string;
}

// Corpo de criacao/edicao. So as chaves abaixo sao aceitas pelo backend.
export interface AnnouncementInput {
  title?: string;
  body?: string;
  audience?: string;
  audience_filter?: AudienceFilter;
  channel?: string;
  schedule_type?: "manual" | "once" | "daily" | "event";
  schedule_at?: string;
  schedule_time?: string;
  schedule_event_id?: string;
  schedule_offset_minutes?: number;
  is_active?: boolean;
}

export interface AnnouncementDelivery {
  id: string;
  announcement_id: string;
  channel: string;
  recipient: string;
  recipient_name?: string;
  source?: string;
  provider: string;
  provider_message_id?: string;
  status: string;
  attempts: number;
  error?: string;
  sent_at?: string;
  created_at: string;
}

// Segmentacao do disparo em massa (#32). Sexo, estado civil, faixa etaria e
// situacao sao atributos so de membros - ao usa-los, visitantes saem do publico.
export interface AudienceFilter {
  group_ids?: string[];
  ministry_ids?: string[];
  branch_ids?: string[];
  genders?: string[];
  marital_statuses?: string[];
  membership_statuses?: string[];
  age_min?: number;
  age_max?: number;
}

export interface DeliveryStats {
  pending: number;
  sent: number;
  failed: number;
  total: number;
}

export interface SendAnnouncementResult {
  announcement_id: string;
  channel: string;
  provider: string;
  recipient_count: number;
  status: string;
}

export const listAnnouncements = () => api<{ announcements: Announcement[] }>("/api/v1/announcements");
export const createAnnouncement = (data: AnnouncementInput) =>
  api<Announcement>("/api/v1/announcements", { method: "POST", body: JSON.stringify(data) });
export const updateAnnouncement = (id: string, data: AnnouncementInput) =>
  api<Announcement>(`/api/v1/announcements/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteAnnouncement = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/announcements/${id}`, { method: "DELETE" });

// Historico de execucoes dos comunicados agendados.
export interface AnnouncementRun {
  id: string;
  announcement_id: string;
  announcement_title: string;
  schedule_type: string;
  period_key?: string;
  recipient_count: number;
  fired_at: string;
}
export const listNotificationRuns = (limit = 100) =>
  api<{ runs: AnnouncementRun[] }>(`/api/v1/notification-runs?limit=${limit}`);

export interface SendInput {
  channel?: string;
  audience: string;
  audience_filter?: AudienceFilter;
}

export const sendAnnouncement = (id: string, data: SendInput) =>
  api<SendAnnouncementResult>(`/api/v1/announcements/${id}/send`, {
    method: "POST",
    body: JSON.stringify(data),
  });

export const previewAudience = (data: SendInput) =>
  api<{ count: number }>("/api/v1/announcements/audience/preview", {
    method: "POST",
    body: JSON.stringify(data),
  });

export const listAnnouncementDeliveries = (id: string) =>
  api<{ deliveries: AnnouncementDelivery[]; stats: DeliveryStats }>(
    `/api/v1/announcements/${id}/deliveries`
  );

// ---- Automacoes de WhatsApp (#31) ----
export interface NotificationSettings {
  tenant_id: string;
  birthdays_enabled: boolean;
  birthday_template: string;
  roster_reminders_enabled: boolean;
  roster_reminder_hours: number;
  roster_template: string;
  visitor_welcome_enabled: boolean;
  visitor_welcome_delay_hours: number;
  visitor_welcome_template: string;
  updated_at?: string;
}

export const getNotificationSettings = () =>
  api<NotificationSettings>("/api/v1/notifications/settings");

export const updateNotificationSettings = (data: Partial<NotificationSettings>) =>
  api<NotificationSettings>("/api/v1/notifications/settings", {
    method: "PATCH",
    body: JSON.stringify(data),
  });

export const runNotifications = () =>
  api<{ queued: number }>("/api/v1/notifications/run", { method: "POST" });

export interface TestMessageInput {
  phone: string;
  message?: string;
}
export interface TestMessageResult {
  provider: string;
  status: string;
  to: string;
}
export const sendTestMessage = (data: TestMessageInput) =>
  api<TestMessageResult>("/api/v1/announcements/send-test", {
    method: "POST",
    body: JSON.stringify(data),
  });

// ---- Eventos, tipos de evento, chamada e frequencia ----
export interface EventKind {
  id: string;
  branch_id?: string;
  name: string;
  slug: string;
  sort_order: number;
  is_active: boolean;
  color?: string;
}
export interface ChurchEvent {
  id: string;
  branch_id: string;
  kind_id?: string;
  kind_name?: string;
  starts_at: string;
  ends_at?: string;
  participants_count: number;
  attendance_count: number;
  attendance_mode: "nominal" | "count";
  estimated_cost?: number;
  cost_actual: number;
  invited_count: number;
  notes?: string;
  created_at: string;
}
export interface EventInvitee {
  id: string;
  event_id: string;
  member_id?: string;
  member_name?: string;
  ministry_id?: string;
  ministry_name?: string;
}
export interface EventAttendance {
  id: string;
  event_id: string;
  member_id: string;
  member_name: string;
  present: boolean;
}
export interface FrequencyEntry {
  id: string;
  member_id: string;
  frequency: "frequente" | "pouco_frequente" | "nao_frequente";
  started_at: string;
  ended_at?: string;
  notes?: string;
}

export const listEventKinds = () => api<{ kinds: EventKind[] }>("/api/v1/event-kinds");
export const createEventKind = (data: Record<string, unknown>) =>
  api<EventKind>("/api/v1/event-kinds", { method: "POST", body: JSON.stringify(data) });
export const updateEventKind = (id: string, data: Record<string, unknown>) =>
  api<EventKind>(`/api/v1/event-kinds/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteEventKind = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/event-kinds/${id}`, { method: "DELETE" });

export const listEvents = (params: { from?: string; to?: string; kind?: string } = {}) => {
  const q = new URLSearchParams();
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  if (params.kind) q.set("kind", params.kind);
  const qs = q.toString();
  return api<{ events: ChurchEvent[] }>(`/api/v1/events${qs ? `?${qs}` : ""}`);
};
export const getEvent = (id: string) => api<ChurchEvent>(`/api/v1/events/${id}`);
export const createEvent = (data: Record<string, unknown>) =>
  api<ChurchEvent>("/api/v1/events", { method: "POST", body: JSON.stringify(data) });
export const updateEvent = (id: string, data: Record<string, unknown>) =>
  api<ChurchEvent>(`/api/v1/events/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteEvent = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/events/${id}`, { method: "DELETE" });
export const listEventAttendance = (id: string) =>
  api<{ attendance: EventAttendance[] }>(`/api/v1/events/${id}/attendance`);
export const saveEventAttendance = (id: string, participantsCount: number, presentMemberIds: string[]) =>
  api<{ ok: boolean }>(`/api/v1/events/${id}/attendance`, {
    method: "POST",
    body: JSON.stringify({ participants_count: participantsCount, present_member_ids: presentMemberIds }),
  });
export const listEventInvitees = (id: string) =>
  api<{ invitees: EventInvitee[] }>(`/api/v1/events/${id}/invitees`);
export const setEventInvitees = (id: string, memberIds: string[], ministryIds: string[]) =>
  api<{ ok: boolean }>(`/api/v1/events/${id}/invitees`, {
    method: "POST",
    body: JSON.stringify({ member_ids: memberIds, ministry_ids: ministryIds }),
  });

export const listMemberFrequency = (memberId: string) =>
  api<{ frequency: FrequencyEntry[] }>(`/api/v1/members/${memberId}/frequency`);
export const setMemberFrequency = (memberId: string, data: Record<string, unknown>) =>
  api<FrequencyEntry>(`/api/v1/members/${memberId}/frequency`, {
    method: "POST",
    body: JSON.stringify(data),
  });

// ---- App do membro (publico por token) ----
/**
 * Resposta do endpoint publico da carteirinha. E uma projecao MINIMA de
 * proposito: o token que endereca a rota esta impresso no QR da carteirinha
 * fisica, entao qualquer coisa devolvida aqui e alcancavel por quem fotografar
 * um cartao. Por isso o `member` traz so o nome - nada de CPF, RG, telefone ou
 * data de nascimento (ver internal/httpapi/memberapp_handlers.go).
 */
export interface PublicCardData {
  member: {
    full_name: string;
  };
  card: {
    token: string;
    card_ref: string;
    member: string;
    branch_name?: string;
  };
  announcements: Announcement[];
}
export const getPublicCard = (token: string) => api<PublicCardData>(`/api/v1/public/card/${token}`);

/** URL do endpoint HTML de impressao da carteirinha (mesma origem do webadmin). */
export const cardPrintURL = (token: string) => `/api/v1/public/card/${token}/print`;

/**
 * URL da foto do membro dono da carteirinha. Nao use `assetURL(photo_url)`:
 * aquele formato aponta para /api/v1/attachments/{arquivo}, que nao tem auth
 * nenhuma e serve junto os comprovantes do financeiro. Este endpoint e amarrado
 * ao token da carteirinha e e o unico que pode ficar publico.
 *
 * Devolve 404 quando o membro nao tem foto; o <Avatar> cai para as iniciais.
 */
export const cardPhotoURL = (token: string) => `/api/v1/public/card/${token}/photo`;

/**
 * Resolve a URL de um arquivo do backend a partir do caminho relativo que a API
 * devolve (ex.: photo_url, no formato `/api/v1/attachments/<arquivo>`).
 *
 * Hoje e quase uma passagem, porque webadmin e API estao na mesma origem: o
 * caminho que a API devolve ja funciona direto no <img>. Continua existindo como
 * o unico ponto que traduz caminho do backend em URL para o browser - se as duas
 * origens voltarem a divergir, e so aqui que muda. Preserva tambem o caso de um
 * valor ja absoluto (dado legado) e a normalizacao de vazio para undefined.
 */
export const assetURL = (path?: string | null) =>
  !path ? undefined : path.startsWith("http") ? path : path;

// ---- LGPD: consentimento, portabilidade e anonimizacao ----
export interface ConsentTerm {
  id: string;
  version: number;
  title: string;
  body: string;
  is_active: boolean;
  created_at: string;
}
export interface ConsentRecord {
  id: string;
  term_id: string;
  term_title: string;
  version: number;
  subject_type: string;
  subject_id: string;
  consented: boolean;
  consented_at?: string;
  created_at: string;
}
export const listConsentTerms = () => api<{ terms: ConsentTerm[] }>("/api/v1/consent-terms");
export const createConsentTerm = (data: Record<string, unknown>) =>
  api<ConsentTerm>("/api/v1/consent-terms", { method: "POST", body: JSON.stringify(data) });
export const listMemberConsents = (memberId: string) =>
  api<{ consents: ConsentRecord[] }>(`/api/v1/members/${memberId}/consents`);
export const recordConsent = (memberId: string, data: Record<string, unknown>) =>
  api<ConsentRecord>(`/api/v1/members/${memberId}/consents`, { method: "POST", body: JSON.stringify(data) });
export const anonymizeMember = (memberId: string) =>
  api<{ ok: boolean }>(`/api/v1/members/${memberId}/anonymize`, { method: "POST" });
export const exportMemberData = (memberId: string) =>
  downloadExport(`/api/v1/members/${memberId}/export`, `dados-membro-${memberId}.json`);

// ---- Governanca (Modulo 6): atas, votacao, mandatos e convenios ----
export type MinuteStatus = "rascunho" | "aprovada" | "assinada" | "cancelada";
export interface Minute {
  id: string;
  branch_id: string;
  title: string;
  meeting_at: string;
  kind: string;
  body?: string;
  status: MinuteStatus;
  quorum_required: number;
  attendance_count: number;
  signature_count: number;
  vote_count: number;
  created_at: string;
  updated_at: string;
}
export interface MinuteSignature {
  id: string;
  minute_id: string;
  user_id?: string;
  signer_name: string;
  signer_role?: string;
  document_hash: string;
  signed_at: string;
  hash: string;
}
export interface VoteOption {
  id: string;
  label: string;
  sort_order: number;
  votes: number;
}
export type VoteStatus = "rascunho" | "aberta" | "encerrada" | "cancelada";
export interface Vote {
  id: string;
  branch_id: string;
  minute_id?: string;
  minute_title?: string;
  title: string;
  description?: string;
  kind: string;
  secret: boolean;
  quorum_required: number;
  min_attendance: number;
  opens_at?: string;
  closes_at?: string;
  status: VoteStatus;
  ballot_count: number;
  participant_count: number;
  quorum_met: boolean;
  options?: VoteOption[];
  result?: VoteResult;
  created_at: string;
  updated_at: string;
}
export interface VoteResult {
  total: number;
  participants: number;
  quorum_required: number;
  quorum_met: boolean;
  winner?: string;
  options: VoteOption[];
  generated_at: string;
}
export interface LegalDocument {
  id: string;
  branch_id: string;
  kind: string;
  title: string;
  description?: string;
  reference?: string;
  issued_at?: string;
  expires_at?: string;
  file_url?: string;
  days_to_expiry?: number;
  expired: boolean;
  created_at: string;
  updated_at: string;
}
export interface Mandate {
  id: string;
  member_id: string;
  member_name: string;
  cargo_id: string;
  cargo_name: string;
  cargo_kind: string;
  started_at?: string;
  ends_at?: string;
  status: "ativo" | "encerrado";
  days_to_expiry?: number;
  expiring: boolean;
}

export const listMinutes = (params: { from?: string; to?: string } = {}) => {
  const q = new URLSearchParams();
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  const qs = q.toString();
  return api<{ minutes: Minute[] }>(`/api/v1/minutes${qs ? `?${qs}` : ""}`);
};
export const createMinute = (data: Record<string, unknown>) =>
  api<Minute>("/api/v1/minutes", { method: "POST", body: JSON.stringify(data) });
export const updateMinute = (id: string, data: Record<string, unknown>) =>
  api<Minute>(`/api/v1/minutes/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteMinute = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/minutes/${id}`, { method: "DELETE" });
export const signMinute = (id: string) =>
  api<MinuteSignature>(`/api/v1/minutes/${id}/sign`, { method: "POST" });
export const listMinuteSignatures = (id: string) =>
  api<{ signatures: MinuteSignature[] }>(`/api/v1/minutes/${id}/signatures`);

export const listVotes = (status = "") =>
  api<{ votes: Vote[] }>(`/api/v1/votes${status ? `?status=${status}` : ""}`);
export const createVote = (data: Record<string, unknown>) =>
  api<Vote>("/api/v1/votes", { method: "POST", body: JSON.stringify(data) });
export const updateVote = (id: string, data: Record<string, unknown>) =>
  api<Vote>(`/api/v1/votes/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteVote = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/votes/${id}`, { method: "DELETE" });
export const openVote = (id: string) =>
  api<Vote>(`/api/v1/votes/${id}/open`, { method: "POST" });
export const closeVote = (id: string) =>
  api<Vote>(`/api/v1/votes/${id}/close`, { method: "POST" });
export const castBallot = (id: string, optionId: string) =>
  api<VoteResult>(`/api/v1/votes/${id}/ballot`, {
    method: "POST",
    body: JSON.stringify({ option_id: optionId }),
  });
export const getVoteResult = (id: string) => api<VoteResult>(`/api/v1/votes/${id}/result`);

export const listLegalDocuments = (kind = "") =>
  api<{ documents: LegalDocument[] }>(`/api/v1/legal-documents${kind ? `?kind=${kind}` : ""}`);
export const createLegalDocument = (data: Record<string, unknown>) =>
  api<LegalDocument>("/api/v1/legal-documents", { method: "POST", body: JSON.stringify(data) });
export const updateLegalDocument = (id: string, data: Record<string, unknown>) =>
  api<LegalDocument>(`/api/v1/legal-documents/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteLegalDocument = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/legal-documents/${id}`, { method: "DELETE" });

export const listMandates = () => api<{ mandates: Mandate[] }>("/api/v1/governance/mandates");

// ---- Escalas de voluntarios ----
export type RosterStatus = "rascunho" | "publicada" | "concluida" | "cancelada";
export type AssignmentStatus = "convidado" | "confirmado" | "recusado";
export interface RosterAssignment {
  id: string;
  roster_id: string;
  member_id: string;
  member_name: string;
  role?: string;
  status: AssignmentStatus;
  responded_at?: string;
  notes?: string;
}
export interface Roster {
  id: string;
  branch_id: string;
  ministry_id?: string;
  ministry_name?: string;
  event_id?: string;
  event_name?: string;
  event_kind_id?: string;
  event_kind_name?: string;
  generated_event: boolean;
  title: string;
  starts_at: string;
  ends_at?: string;
  location?: string;
  notes?: string;
  status: RosterStatus;
  assignment_count: number;
  confirmed_count: number;
  assignments?: RosterAssignment[];
  created_at: string;
}
export interface RosterConflict {
  member_id: string;
  member_name: string;
  other_roster_id: string;
  other_roster_title: string;
  other_starts_at: string;
}
export interface RosterSuggestion {
  member_id: string;
  member_name: string;
  frequency?: string;
  busy: boolean;
}

export const listRosters = (params: { from?: string; to?: string; ministry?: string } = {}) => {
  const q = new URLSearchParams();
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  if (params.ministry) q.set("ministry", params.ministry);
  const qs = q.toString();
  return api<{ rosters: Roster[] }>(`/api/v1/rosters${qs ? `?${qs}` : ""}`);
};
export const getRoster = (id: string) => api<Roster>(`/api/v1/rosters/${id}`);
export const createRoster = (data: Record<string, unknown>) =>
  api<Roster>("/api/v1/rosters", { method: "POST", body: JSON.stringify(data) });
export const updateRoster = (id: string, data: Record<string, unknown>) =>
  api<Roster>(`/api/v1/rosters/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteRoster = (id: string, deleteEvent = false) =>
  api<{ ok: boolean }>(`/api/v1/rosters/${id}${deleteEvent ? "?delete_event=true" : ""}`, { method: "DELETE" });
export const setRosterAssignments = (id: string, assignments: { member_id: string; role?: string }[]) =>
  api<Roster>(`/api/v1/rosters/${id}/assignments`, {
    method: "POST",
    body: JSON.stringify({ assignments }),
  });
export const respondRosterAssignment = (rosterId: string, assignmentId: string, status: AssignmentStatus, notes?: string) =>
  api<RosterAssignment>(`/api/v1/rosters/${rosterId}/assignments/${assignmentId}`, {
    method: "PATCH",
    body: JSON.stringify({ status, notes }),
  });
export const getRosterConflicts = (id: string) =>
  api<{ conflicts: RosterConflict[] }>(`/api/v1/rosters/${id}/conflicts`);
export const getRosterSuggestions = (params: { ministry_id: string; starts_at: string; ends_at?: string; roster_id?: string }) => {
  const q = new URLSearchParams();
  q.set("ministry_id", params.ministry_id);
  q.set("starts_at", params.starts_at);
  if (params.ends_at) q.set("ends_at", params.ends_at);
  if (params.roster_id) q.set("roster_id", params.roster_id);
  return api<{ suggestions: RosterSuggestion[] }>(`/api/v1/rosters/suggestions?${q}`);
};

// ---- Exportacao de relatorios (CSV/PDF) ----
// Usa apiRaw para preservar o auto-refresh em caso de 401 e obter o blob
// diretamente (nao tenta JSON.parse no conteudo binario).
export async function downloadExport(path: string, filename: string): Promise<void> {
  const res = await apiRaw(path);
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * Abre o relatorio de impressao (HTML) numa nova aba e dispara a impressao.
 *
 * Nao da para usar `window.open(url)` direto: o endpoint de exportacao exige
 * Bearer e uma navegacao nova nao envia o header. Por isso busca com apiRaw
 * (que autentica e renova o token) e escreve o HTML na aba nova.
 */
export async function openPrint(path: string): Promise<void> {
  // Abre a aba ANTES do await: se abrir depois, os bloqueadores de pop-up
  // recusam a nova janela (nao e mais um gesto do usuario).
  const w = window.open("", "_blank");
  const res = await apiRaw(path);
  const html = await res.text();
  if (!w) return;
  w.document.open();
  w.document.write(html);
  w.document.close();
  w.focus();
  w.setTimeout(() => w.print(), 400);
}

// ---- Ministerio Infantil (Kids) ----
export interface KidTrack {
  id: string;
  branch_id: string;
  name: string;
  description: string;
  age_min?: number;
  age_max?: number;
  is_active: boolean;
  lesson_count: number;
  created_at: string;
}
export interface KidLesson {
  id: string;
  track_id: string;
  position: number;
  title: string;
  objective: string;
  verse: string;
  content: string;
  materials: string;
  created_at: string;
}
export interface KidClass {
  id: string;
  branch_id: string;
  name: string;
  age_min?: number;
  age_max?: number;
  track_id: string;
  track_name: string;
  room: string;
  leader_member_id: string;
  leader_name: string;
  is_active: boolean;
  enrollment_count: number;
  created_at: string;
}
export interface KidEnrollment {
  id: string;
  class_id: string;
  class_name: string;
  member_id: string;
  member_name: string;
  birth_date?: string;
  status: "active" | "paused" | "ended";
  start_date: string;
  end_date?: string;
  dietary_restrictions: string;
  notes: string;
  guardians_count: number;
  created_at: string;
}
export interface KidGuardian {
  id: string;
  enrollment_id: string;
  member_id: string;
  member_name: string;
  relationship: string;
  is_primary: boolean;
}
export interface KidSession {
  id: string;
  class_id: string;
  class_name: string;
  lesson_id: string;
  lesson_title: string;
  lesson_position: number;
  starts_at: string;
  ends_at?: string;
  status: "scheduled" | "open" | "closed";
  notes: string;
  checkin_count: number;
}
export interface KidRosterEntry {
  checkin_id: string;
  enrollment_id: string;
  member_id: string;
  member_name: string;
  birth_date?: string;
  status: "" | "present" | "absent";
  security_code: string;
  checkin_at?: string;
  checkout_at?: string;
  dropoff_guardian_id: string;
  pickup_guardian_id: string;
  dietary_restrictions: string;
  guardians: string;
}
export interface KidEvolutionRow {
  enrollment_id: string;
  member_id: string;
  member_name: string;
  birth_date?: string;
  attended: number;
  total_sessions: number;
  attendance_pct: number;
  lessons_attended: number;
  total_lessons: number;
  progress_pct: number;
  last_lesson: string;
}
export interface KidEvolution {
  class_id: string;
  class_name: string;
  track_name: string;
  total_sessions: number;
  total_lessons: number;
  rows: KidEvolutionRow[];
}

export const listKidTracks = () => api<{ tracks: KidTrack[] }>("/api/v1/kids/tracks");
export const createKidTrack = (data: Record<string, unknown>) =>
  api<KidTrack>("/api/v1/kids/tracks", { method: "POST", body: JSON.stringify(data) });
export const updateKidTrack = (id: string, data: Record<string, unknown>) =>
  api<KidTrack>(`/api/v1/kids/tracks/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteKidTrack = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/tracks/${id}`, { method: "DELETE" });

export const listKidLessons = (trackId: string) =>
  api<{ lessons: KidLesson[] }>(`/api/v1/kids/tracks/${trackId}/lessons`);
export const createKidLesson = (trackId: string, data: Record<string, unknown>) =>
  api<KidLesson>(`/api/v1/kids/tracks/${trackId}/lessons`, { method: "POST", body: JSON.stringify(data) });
export const updateKidLesson = (id: string, data: Record<string, unknown>) =>
  api<KidLesson>(`/api/v1/kids/lessons/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteKidLesson = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/lessons/${id}`, { method: "DELETE" });

export const listKidClasses = () => api<{ classes: KidClass[] }>("/api/v1/kids/classes");
export const createKidClass = (data: Record<string, unknown>) =>
  api<KidClass>("/api/v1/kids/classes", { method: "POST", body: JSON.stringify(data) });
export const updateKidClass = (id: string, data: Record<string, unknown>) =>
  api<KidClass>(`/api/v1/kids/classes/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteKidClass = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/classes/${id}`, { method: "DELETE" });

export const listKidEnrollments = (classId: string) =>
  api<{ enrollments: KidEnrollment[] }>(`/api/v1/kids/classes/${classId}/enrollments`);
export const createKidEnrollment = (classId: string, data: Record<string, unknown>) =>
  api<KidEnrollment>(`/api/v1/kids/classes/${classId}/enrollments`, { method: "POST", body: JSON.stringify(data) });
export const updateKidEnrollment = (id: string, data: Record<string, unknown>) =>
  api<KidEnrollment>(`/api/v1/kids/enrollments/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteKidEnrollment = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/enrollments/${id}`, { method: "DELETE" });

export const listKidGuardians = (enrollmentId: string) =>
  api<{ guardians: KidGuardian[] }>(`/api/v1/kids/enrollments/${enrollmentId}/guardians`);
export const addKidGuardian = (enrollmentId: string, data: Record<string, unknown>) =>
  api<KidGuardian>(`/api/v1/kids/enrollments/${enrollmentId}/guardians`, { method: "POST", body: JSON.stringify(data) });
export const deleteKidGuardian = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/guardians/${id}`, { method: "DELETE" });

export const listKidSessions = (params: { class_id?: string; from?: string; to?: string } = {}) => {
  const q = new URLSearchParams();
  if (params.class_id) q.set("class_id", params.class_id);
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  const qs = q.toString();
  return api<{ sessions: KidSession[] }>(`/api/v1/kids/sessions${qs ? `?${qs}` : ""}`);
};
export const createKidSession = (data: Record<string, unknown>) =>
  api<KidSession>("/api/v1/kids/sessions", { method: "POST", body: JSON.stringify(data) });
export const updateKidSession = (id: string, data: Record<string, unknown>) =>
  api<KidSession>(`/api/v1/kids/sessions/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const deleteKidSession = (id: string) =>
  api<{ ok: boolean }>(`/api/v1/kids/sessions/${id}`, { method: "DELETE" });

export const getKidRoster = (sessionId: string) =>
  api<{ roster: KidRosterEntry[] }>(`/api/v1/kids/sessions/${sessionId}/roster`);
export const kidCheckin = (sessionId: string, data: { enrollment_id: string; dropoff_guardian_id?: string }) =>
  api<KidRosterEntry>(`/api/v1/kids/sessions/${sessionId}/checkin`, { method: "POST", body: JSON.stringify(data) });
export const kidCheckout = (sessionId: string, data: { enrollment_id: string; security_code?: string; pickup_guardian_id?: string }) =>
  api<KidRosterEntry>(`/api/v1/kids/sessions/${sessionId}/checkout`, { method: "POST", body: JSON.stringify(data) });
export const kidAbsence = (sessionId: string, data: { enrollment_id: string; absent: boolean }) =>
  api<{ ok: boolean }>(`/api/v1/kids/sessions/${sessionId}/absence`, { method: "POST", body: JSON.stringify(data) });

export const getKidEvolution = (classId: string, params: { from?: string; to?: string } = {}) => {
  const q = new URLSearchParams();
  if (params.from) q.set("from", params.from);
  if (params.to) q.set("to", params.to);
  const qs = q.toString();
  return api<KidEvolution>(`/api/v1/kids/classes/${classId}/evolution${qs ? `?${qs}` : ""}`);
};
