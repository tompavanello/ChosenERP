"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  login as apiLogin, selectTenant as apiSelectTenant, switchTenant as apiSwitchTenant,
  restoreTokens, setTokens, clearTokens, fetchMe, getBranchContext, setBranchContext,
  setTenantContext, hasSession, STORAGE_KEYS, registerSessionLost,
  type Tokens, type User, type TenantOption,
} from "@/lib/api";

const isHQRole = (role: string) => role === "super_admin" || role === "admin_sede";

/** A Sede enxerga todas as filiais por padrão (contexto "all"). */
function ensureDefaultBranchContext(u: User) {
  if (isHQRole(u.role) && !getBranchContext()) setBranchContext("all");
}

/** Seleção de igreja pendente (identidade com mais de um vínculo). */
interface PendingSelection {
  token: string;
  tenants: TenantOption[];
}

interface AuthContextValue {
  user: User | null;
  ready: boolean;
  pendingSelection: PendingSelection | null;
  /** Retorna true quando entrou direto; false quando precisa escolher a igreja. */
  login: (email: string, password: string, code?: string, tenantSlug?: string) => Promise<boolean>;
  selectTenant: (tenantId: string) => Promise<void>;
  switchTenant: (tenantId: string) => Promise<void>;
  logout: () => void;
  hasPerm: (perm: string) => boolean;
  /** Recarrega o perfil (/me) e atualiza sessão + cache local. */
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth deve estar dentro de <AuthProvider>");
  return ctx;
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const [pendingSelection, setPendingSelection] = useState<PendingSelection | null>(null);

  const logout = useCallback(() => {
    clearTokens();
    localStorage.removeItem(STORAGE_KEYS.user);
    setTenantContext("");
    setUser(null);
    setPendingSelection(null);
    window.location.href = "/";
  }, []);

  const persistSession = useCallback((tokens: Tokens, u: User) => {
    setTokens(tokens);
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(u));
    setTenantContext(u.tenant_id);
    ensureDefaultBranchContext(u);
    setUser(u);
  }, []);

  useEffect(() => {
    registerSessionLost(logout);
    restoreTokens();
    // Sem sessão (ex.: tela de login), não chama /me — evita um 401 no console.
    if (!hasSession()) {
      setReady(true);
      return;
    }
    const cached = localStorage.getItem(STORAGE_KEYS.user);
    const doLoad = async () => {
      if (cached) {
        try {
          setUser(JSON.parse(cached));
        } catch {
          /* ignore */
        }
      }
      try {
        const me = await fetchMe();
        setUser(me);
        setTenantContext(me.tenant_id);
        ensureDefaultBranchContext(me);
        localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(me));
      } catch {
        /* token inválido/expirado; limpeza já feita pelo cliente */
      } finally {
        setReady(true);
      }
    };
    doLoad();
  }, [logout]);

  const login = useCallback(async (email: string, password: string, code?: string, tenantSlug?: string) => {
    const res = await apiLogin(email, password, code, tenantSlug);
    if ("requires_tenant_selection" in res) {
      setPendingSelection({ token: res.selection_token, tenants: res.tenants });
      return false;
    }
    setPendingSelection(null);
    persistSession(res.tokens, res.user);
    return true;
  }, [persistSession]);

  const selectTenant = useCallback(async (tenantId: string) => {
    if (!pendingSelection) throw new Error("nenhuma seleção de igreja pendente");
    const res = await apiSelectTenant(pendingSelection.token, tenantId);
    setPendingSelection(null);
    persistSession(res.tokens, res.user);
  }, [pendingSelection, persistSession]);

  // Troca de igreja de uma sessão ativa: novos tokens e recarrega os dados no
  // novo escopo (mesmo padrão do switcher de filial).
  const switchTenant = useCallback(async (tenantId: string) => {
    const res = await apiSwitchTenant(tenantId);
    persistSession(res.tokens, res.user);
    window.location.reload();
  }, [persistSession]);

  // `user` pode vir do cache do localStorage (gravado por uma versão anterior
  // do app), então `permissions` é tratado como opcional em tempo de execução.
  const hasPerm = useCallback((perm: string) => !!user && (user.permissions ?? []).includes(perm), [user]);

  const refresh = useCallback(async () => {
    const me = await fetchMe();
    setUser(me);
    setTenantContext(me.tenant_id);
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(me));
  }, []);

  const value = useMemo(
    () => ({ user, ready, pendingSelection, login, selectTenant, switchTenant, logout, hasPerm, refresh }),
    [user, ready, pendingSelection, login, selectTenant, switchTenant, logout, hasPerm, refresh],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
