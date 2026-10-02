// Cliente da API do App do Membro.
//
// Mesmo principio do webadmin: a API e servida na MESMA origem por caminho
// (/api/*), entao todo fetch e RELATIVO. O tenant (igreja) vem do subdominio.

export const STORAGE_KEYS = {
  token: "chosen_member_token",
  refresh: "chosen_member_refresh",
  user: "chosen_member_user",
};

const RESERVED_SLUGS = new Set(["app", "www", "api", "admin", "localhost"]);

/** Extrai o slug da igreja do host (subdominio). */
export function tenantSlugFromHost(): string {
  if (typeof window === "undefined") return "";
  const host = window.location.hostname.toLowerCase();
  const base = process.env.NEXT_PUBLIC_BASE_DOMAIN?.toLowerCase();
  if (base) {
    if (!host.endsWith(`.${base}`)) return "";
    const sub = host.slice(0, -(base.length + 1));
    return sub.includes(".") || RESERVED_SLUGS.has(sub) ? "" : sub;
  }
  const parts = host.split(".");
  if (parts.length < 3) return "";
  const slug = parts[0];
  return RESERVED_SLUGS.has(slug) ? "" : slug;
}

// ---- Sessao ----
export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(STORAGE_KEYS.token);
}
export function getRefresh(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(STORAGE_KEYS.refresh);
}
export function getCachedUser(): User | null {
  if (typeof window === "undefined") return null;
  const raw = localStorage.getItem(STORAGE_KEYS.user);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as User;
  } catch {
    return null;
  }
}
function storeSession(access: string, refresh: string, user: User) {
  if (typeof window === "undefined") return;
  localStorage.setItem(STORAGE_KEYS.token, access);
  localStorage.setItem(STORAGE_KEYS.refresh, refresh);
  localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(user));
}
export function clearSession() {
  if (typeof window === "undefined") return;
  localStorage.removeItem(STORAGE_KEYS.token);
  localStorage.removeItem(STORAGE_KEYS.refresh);
  localStorage.removeItem(STORAGE_KEYS.user);
}

let onSessionLost: (() => void) | null = null;
export function registerSessionLost(fn: () => void) {
  onSessionLost = fn;
}

// ---- Tipos ----
export interface Membership {
  tenant_id: string;
  tenant_name: string;
  tenant_slug: string;
  role: string;
  branch_id: string;
  is_active: boolean;
  member_id?: string;
}
export interface User {
  id: string;
  email: string;
  full_name: string;
  tenant_id: string;
  branch_id: string;
  role: string;
  member_id?: string;
  phone?: string;
  must_change_password?: boolean;
  permissions?: string[];
  memberships?: Membership[];
}
export interface PublicTenant {
  name: string;
  slug: string;
  logo_url?: string | null;
  brand_color?: string | null;
  favicon_url?: string | null;
  pix_key?: string | null;
  pix_name?: string | null;
}
export interface Address {
  zip_code?: string;
  street?: string;
  number?: string;
  complement?: string;
  district?: string;
  city?: string;
  state?: string;
}
export interface Member {
  id: string;
  full_name: string;
  nickname?: string | null;
  email?: string | null;
  phone?: string | null;
  whatsapp?: string | null;
  birth_date?: string | null;
  baptism_date?: string | null;
  joined_at?: string | null;
  membership_status: string;
  roll_class: string;
  address?: Address | null;
  photo_url?: string | null;
  card_ref?: string | null;
}
export interface Family {
  id: string;
  name: string;
  code?: string;
}
export interface ChurchEvent {
  id: string;
  title: string;
  kind_name?: string;
  starts_at: string;
  ends_at?: string | null;
  location?: string | null;
  notes?: string | null;
}
export interface Announcement {
  id: string;
  title: string;
  body: string;
  published_at: string;
}
export interface PrayerRequest {
  id: string;
  author_name?: string | null;
  body: string;
  visibility: string;
  is_anonymous: boolean;
  status: string;
  answered_note?: string | null;
  prayer_count: number;
  reacted_by_me: boolean;
  created_at: string;
}

export interface Birthday {
  id: string;
  full_name: string;
  birth_date: string;
  day: number;
  age: number;
  whatsapp?: string | null;
  phone?: string | null;
}
export interface MarriageAnniversary {
  id: string;
  full_name: string;
  spouse_name?: string | null;
  marriage_date: string;
  day: number;
  years: number;
}
export interface MemberMinistry {
  id: string;
  name: string;
  role: string;
  leader_name?: string | null;
  started_at?: string | null;
}
export interface Contribution {
  id: string;
  amount: number;
  category_name?: string | null;
  payment_method?: string | null;
  description?: string | null;
  occurred_at: string;
}

export interface Tokens {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface LoginResult {
  requires_tenant_selection?: boolean;
  selection_token?: string;
  tenants?: { id: string; name: string; slug: string; role: string }[];
  tokens?: Tokens;
  user?: User;
}

// ---- Núcleo HTTP ----
async function refreshTokens(): Promise<boolean> {
  const refresh = getRefresh();
  if (!refresh) return false;
  const res = await fetch("/api/v1/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refresh }),
  });
  if (!res.ok) return false;
  const data = (await res.json()) as { tokens: Tokens };
  if (typeof window !== "undefined") {
    localStorage.setItem(STORAGE_KEYS.token, data.tokens.access_token);
    localStorage.setItem(STORAGE_KEYS.refresh, data.tokens.refresh_token);
  }
  return true;
}

async function api<T>(path: string, init?: RequestInit, retry = true): Promise<T> {
  const token = getToken();
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const slug = tenantSlugFromHost();
  if (slug) headers.set("X-Tenant-Slug", slug);

  const res = await fetch(path, { ...init, headers });
  if (res.status === 401 && retry && getRefresh()) {
    if (await refreshTokens()) return api<T>(path, init, false);
    clearSession();
    onSessionLost?.();
  }
  if (!res.ok) {
    let msg = `erro ${res.status}`;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      /* corpo nao-JSON */
    }
    throw new Error(msg);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

// ---- Auth ----
export async function login(identifier: string, password: string): Promise<LoginResult> {
  const slug = tenantSlugFromHost();
  const data = await api<LoginResult & { error?: string }>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ identifier, password, tenant_slug: slug || undefined }),
  });
  if (data.requires_tenant_selection) return data;
  if (data.tokens && data.user) {
    storeSession(data.tokens.access_token, data.tokens.refresh_token, data.user);
  }
  return data;
}

/** Troca a propria senha (usa a senha atual, inclusive a provisoria). */
export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  await api<{ ok: boolean }>("/api/v1/me/password", {
    method: "POST",
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
}

export async function fetchMe(): Promise<User> {
  const user = await api<User>("/api/v1/me");
  if (typeof window !== "undefined") {
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(user));
  }
  return user;
}

export async function getPublicTenant(slug: string): Promise<PublicTenant> {
  const tenant = await api<PublicTenant>(`/api/v1/public/tenant/${encodeURIComponent(slug)}`);
  applyThemeColor(tenant.brand_color);
  return tenant;
}

/** Atualiza a cor da barra do navegador com o branding da igreja. */
export function applyThemeColor(color?: string | null) {
  if (typeof document === "undefined" || !color) return;
  let meta = document.querySelector('meta[name="theme-color"]');
  if (!meta) {
    meta = document.createElement("meta");
    meta.setAttribute("name", "theme-color");
    document.head.appendChild(meta);
  }
  meta.setAttribute("content", color);
}

// ---- App do membro ----
export async function getMeMember(): Promise<Member> {
  const data = await api<{ member: Member }>("/api/v1/me/member");
  return data.member;
}
export async function updateMeMember(input: Partial<Member> & { address?: Address }): Promise<Member> {
  const data = await api<{ member: Member }>("/api/v1/me/member", {
    method: "PATCH",
    body: JSON.stringify(input),
  });
  return data.member;
}
export async function getMeFamily(): Promise<Family[]> {
  const data = await api<{ families: Family[] }>("/api/v1/me/family");
  return data.families ?? [];
}
export async function getMeEvents(from?: string, to?: string): Promise<ChurchEvent[]> {
  const qs = new URLSearchParams();
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  const q = qs.toString();
  const data = await api<{ events: ChurchEvent[] }>(`/api/v1/me/events${q ? `?${q}` : ""}`);
  return data.events ?? [];
}
export async function getMeAnnouncements(): Promise<Announcement[]> {
  const data = await api<{ announcements: Announcement[] }>("/api/v1/me/announcements");
  return data.announcements ?? [];
}
export async function getMyPrayers(): Promise<PrayerRequest[]> {
  const data = await api<{ prayer_requests: PrayerRequest[] }>("/api/v1/me/prayer-requests");
  return data.prayer_requests ?? [];
}
export async function getPrayerWall(): Promise<PrayerRequest[]> {
  const data = await api<{ prayer_requests: PrayerRequest[] }>("/api/v1/me/prayer-wall");
  return data.prayer_requests ?? [];
}
export async function createPrayer(input: {
  body: string;
  visibility: string;
  is_anonymous: boolean;
}): Promise<PrayerRequest> {
  return api<PrayerRequest>("/api/v1/me/prayer-requests", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
export async function reactPrayer(id: string): Promise<PrayerRequest> {
  return api<PrayerRequest>(`/api/v1/me/prayer-requests/${id}/react`, { method: "POST" });
}

export async function getMeBirthdays(month?: number): Promise<{
  month: number;
  birthdays: Birthday[];
  marriages: MarriageAnniversary[];
}> {
  const qs = month ? `?month=${month}` : "";
  return api(`/api/v1/me/birthdays${qs}`);
}

export async function getMeMinistries(): Promise<MemberMinistry[]> {
  const data = await api<{ ministries: MemberMinistry[] }>("/api/v1/me/ministries");
  return data.ministries ?? [];
}

export async function getMeContributions(year?: number): Promise<{
  year: number;
  contributions: Contribution[];
  total: number;
}> {
  const qs = year ? `?year=${year}` : "";
  return api(`/api/v1/me/contributions${qs}`);
}

// ---- Formatação ----
export function datePt(iso?: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric" });
}
export function dateTimePt(iso?: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export const VISIBILITY_LABELS: Record<string, string> = {
  pastor: "Somente pastor",
  pastor_conselho: "Pastor + conselho",
  grupo: "Meu grupo",
  igreja: "Igreja",
};
