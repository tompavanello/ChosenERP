const API_URL = process.env.API_URL ?? "http://localhost:38080";

export const STORAGE_KEYS = { token: "chosen_token", refresh: "chosen_refresh", user: "chosen_user" };

// ---- Tipos ----
export interface User {
  id: string;
  email: string;
  full_name: string;
  tenant_id: string;
  branch_id: string;
  role: string;
  permissions: string[];
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
  profession?: string;
  office?: string;
  created_at: string;
}

export interface Category {
  id: string;
  branch_id?: string;
  type: "income" | "expense";
  code: string;
  name: string;
}

export interface Transaction {
  id: string;
  branch_id: string;
  category_id?: string;
  category_name?: string;
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
  hash: string;
  occurred_at: string;
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
  head_id?: string;
  branch_id: string;
  member_count: number;
}
export interface Visitor {
  id: string;
  branch_id: string;
  first_name: string;
  last_name: string;
  full_name: string;
  email?: string;
  phone?: string;
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

// ---- Sessão (module-level) ----
let accessToken: string | null = null;
let refreshToken: string | null = null;
let onSessionLost: (() => void) | null = null;

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
  const res = await fetch(`${API_URL}/api/v1/auth/refresh`, {
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
    fetch(`${API_URL}${path}`, {
      ...opts,
      headers: {
        "Content-Type": "application/json",
        ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
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

// ---- Auth / sessão ----
export function login(email: string, password: string) {
  return api<{ tokens: Tokens; user: User }>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}
export function fetchMe() {
  return api<User>("/api/v1/me");
}

// ---- Pessoas ----
export const listMembers = () => api<{ members: Member[] }>("/api/v1/members");
export const getMember = (id: string) => api<Member>(`/api/v1/members/${id}`);
export const createMember = (data: Partial<Member>) =>
  api<Member>("/api/v1/members", { method: "POST", body: JSON.stringify(data) });
export const updateMember = (id: string, data: Record<string, unknown>) =>
  api<Member>(`/api/v1/members/${id}`, { method: "PATCH", body: JSON.stringify(data) });
export const getMemberTree = (id: string) =>
  api<{ member: Member; relationships: Relationship[] }>(`/api/v1/members/${id}/tree`);
export const addRelationship = (id: string, relateMemberId: string, kind: string) =>
  api<{ ok: boolean }>(`/api/v1/members/${id}/relationships`, {
    method: "POST",
    body: JSON.stringify({ relate_member_id: relateMemberId, kind }),
  });
export const issueCard = (memberId: string) =>
  api<{ card_ref: string; member: string }>(`/api/v1/members/${memberId}/card`, { method: "POST" });

// ---- Famílias ----
export const listFamilies = () => api<{ families: Family[] }>("/api/v1/families");
export const createFamily = (name: string) =>
  api<Family>("/api/v1/families", { method: "POST", body: JSON.stringify({ name }) });
export const addFamilyMember = (familyId: string, memberId: string, relateId: string, relation: string) =>
  api<{ ok: boolean }>(`/api/v1/families/${familyId}/members`, {
    method: "POST",
    body: JSON.stringify({ member_id: memberId, relate_id: relateId, relation }),
  });

// ---- Visitantes / Benfeitores ----
export const listVisitors = () => api<{ visitors: Visitor[] }>("/api/v1/visitors");
export const createVisitor = (data: Record<string, unknown>) =>
  api<Visitor>("/api/v1/visitors", { method: "POST", body: JSON.stringify(data) });
export const updateVisitorStage = (id: string, stage: string) =>
  api<Visitor>(`/api/v1/visitors/${id}/stage`, { method: "PATCH", body: JSON.stringify({ stage }) });
export const listBenefactors = () => api<{ benefactors: Benefactor[] }>("/api/v1/benefactors");
export const createBenefactor = (data: Record<string, unknown>) =>
  api<Benefactor>("/api/v1/benefactors", { method: "POST", body: JSON.stringify(data) });

// ---- Financeiro ----
export const listCategories = () => api<{ categories: Category[] }>("/api/v1/finance/categories");
export const createCategory = (data: Record<string, unknown>) =>
  api<Category>("/api/v1/finance/categories", { method: "POST", body: JSON.stringify(data) });
export const listTransactions = (type = "") => {
  const q = type ? `?type=${encodeURIComponent(type)}` : "";
  return api<{ transactions: Transaction[] }>(`/api/v1/finance/transactions${q}`);
};
export const createTransaction = (data: Record<string, unknown>) =>
  api<{ transaction: Transaction; receipt_ref: string; receipt_token: string }>("/api/v1/finance/transactions", {
    method: "POST",
    body: JSON.stringify(data),
  });
export const getBalance = () => api<Balance>("/api/v1/finance/balance");

// ---- Relatórios ----
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

// ---- Recibos ----
export async function getReceiptHTML(documentId: string) {
  const res = await fetch(`${API_URL}/api/v1/receipts/${documentId}`, {
    headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error(await res.text());
  return res.text();
}
export const sendReceipt = (documentId: string, channel: string, recipient: string) =>
  api<Delivery>(`/api/v1/receipts/${documentId}/send`, {
    method: "POST",
    body: JSON.stringify({ channel, recipient }),
  });
export const listDeliveries = (documentId: string) =>
  api<{ deliveries: Delivery[] }>(`/api/v1/receipts/${documentId}/deliveries`);
