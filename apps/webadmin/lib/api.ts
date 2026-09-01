const API_URL = process.env.API_URL ?? "http://localhost:38080";

export interface LoginResult {
  tokens: { access_token: string; refresh_token: string; expires_in: number };
  user: {
    id: string;
    email: string;
    full_name: string;
    tenant_id: string;
    branch_id: string;
    role: string;
    permissions: string[];
  };
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

export interface Balance {
  income: number;
  expense: number;
  net: number;
  by_category: { category_id: string; category: string; type: string; total: number }[];
}

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...opts,
    headers: { "Content-Type": "application/json", ...(opts.headers ?? {}) },
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}

function authHeaders(token: string): HeadersInit {
  return { Authorization: `Bearer ${token}` };
}

export function login(email: string, password: string) {
  return request<LoginResult>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export function listMembers(token: string) {
  return request<{ members: Member[] }>("/api/v1/members", { headers: authHeaders(token) });
}

export function createMember(token: string, data: Partial<Member>) {
  return request<Member>("/api/v1/members", {
    method: "POST",
    headers: authHeaders(token),
    body: JSON.stringify(data),
  });
}

export function issueCard(token: string, memberId: string) {
  return request<{ card_ref: string; member: string }>(`/api/v1/members/${memberId}/card`, {
    method: "POST",
    headers: authHeaders(token),
  });
}

export function listCategories(token: string) {
  return request<{ categories: Category[] }>("/api/v1/finance/categories", { headers: authHeaders(token) });
}

export function listTransactions(token: string, type = "") {
  const q = type ? `?type=${encodeURIComponent(type)}` : "";
  return request<{ transactions: Transaction[] }>(`/api/v1/finance/transactions${q}`, {
    headers: authHeaders(token),
  });
}

export function createTransaction(token: string, data: Record<string, unknown>) {
  return request<{ transaction: Transaction; receipt_ref: string; receipt_token: string }>(
    "/api/v1/finance/transactions",
    { method: "POST", headers: authHeaders(token), body: JSON.stringify(data) },
  );
}

export function getBalance(token: string) {
  return request<Balance>("/api/v1/finance/balance", { headers: authHeaders(token) });
}

// ---- Pessoas: famílias, vínculos, visitantes, benfeitores ----

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

export function getMemberTree(token: string, id: string) {
  return request<{ member: Member; relationships: Relationship[] }>(`/api/v1/members/${id}/tree`, {
    headers: authHeaders(token),
  });
}

export function updateMember(token: string, id: string, data: Record<string, unknown>) {
  return request<Member>(`/api/v1/members/${id}`, { method: "PATCH", headers: authHeaders(token), body: JSON.stringify(data) });
}

export function addRelationship(token: string, id: string, relateMemberId: string, kind: string) {
  return request<{ ok: boolean }>(`/api/v1/members/${id}/relationships`, {
    method: "POST", headers: authHeaders(token), body: JSON.stringify({ relate_member_id: relateMemberId, kind }),
  });
}

export function listFamilies(token: string) {
  return request<{ families: Family[] }>("/api/v1/families", { headers: authHeaders(token) });
}
export function createFamily(token: string, name: string) {
  return request<Family>("/api/v1/families", { method: "POST", headers: authHeaders(token), body: JSON.stringify({ name }) });
}
export function addFamilyMember(token: string, familyId: string, memberId: string, relateId: string, relation: string) {
  return request<{ ok: boolean }>(`/api/v1/families/${familyId}/members`, {
    method: "POST", headers: authHeaders(token), body: JSON.stringify({ member_id: memberId, relate_id: relateId, relation }),
  });
}

export function listVisitors(token: string) {
  return request<{ visitors: Visitor[] }>("/api/v1/visitors", { headers: authHeaders(token) });
}
export function createVisitor(token: string, data: Record<string, unknown>) {
  return request<Visitor>("/api/v1/visitors", { method: "POST", headers: authHeaders(token), body: JSON.stringify(data) });
}
export function updateVisitorStage(token: string, id: string, stage: string) {
  return request<Visitor>(`/api/v1/visitors/${id}/stage`, { method: "PATCH", headers: authHeaders(token), body: JSON.stringify({ stage }) });
}

export function listBenefactors(token: string) {
  return request<{ benefactors: Benefactor[] }>("/api/v1/benefactors", { headers: authHeaders(token) });
}
export function createBenefactor(token: string, data: Record<string, unknown>) {
  return request<Benefactor>("/api/v1/benefactors", { method: "POST", headers: authHeaders(token), body: JSON.stringify(data) });
}

// ---- Relatórios ----

export function getMonthlyBalance(token: string, from = "", to = "") {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  return request<{ series: MonthlyPoint[] }>(`/api/v1/reports/balance?${q}`, { headers: authHeaders(token) });
}

export function getDRE(token: string, from = "", to = "") {
  const q = new URLSearchParams();
  if (from) q.set("from", from);
  if (to) q.set("to", to);
  return request<DRE>(`/api/v1/reports/dre?${q}`, { headers: authHeaders(token) });
}

// ---- Recibos: render e envio ----

export async function getReceiptHTML(token: string, documentId: string) {
  const res = await fetch(`${API_URL}/api/v1/receipts/${documentId}`, { headers: { ...authHeaders(token) } as HeadersInit });
  if (!res.ok) throw new Error(await res.text());
  return res.text();
}

export function sendReceipt(token: string, documentId: string, channel: string, recipient: string) {
  return request<Delivery>(`/api/v1/receipts/${documentId}/send`, {
    method: "POST", headers: authHeaders(token), body: JSON.stringify({ channel, recipient }),
  });
}

export function listDeliveries(token: string, documentId: string) {
  return request<{ deliveries: Delivery[] }>(`/api/v1/receipts/${documentId}/deliveries`, { headers: authHeaders(token) });
}
