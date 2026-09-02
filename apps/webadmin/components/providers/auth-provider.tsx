"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { login as apiLogin, restoreTokens, setTokens, clearTokens, fetchMe, STORAGE_KEYS, registerSessionLost, type User } from "@/lib/api";

interface AuthContextValue {
  user: User | null;
  ready: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  hasPerm: (perm: string) => boolean;
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

  const logout = useCallback(() => {
    clearTokens();
    localStorage.removeItem(STORAGE_KEYS.user);
    setUser(null);
    window.location.href = "/";
  }, []);

  useEffect(() => {
    registerSessionLost(logout);
    restoreTokens();
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
        localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(me));
      } catch {
        /* token inválido/expirado; limpeza já feita pelo cliente */
      } finally {
        setReady(true);
      }
    };
    doLoad();
  }, [logout]);

  const login = useCallback(async (email: string, password: string) => {
    const res = await apiLogin(email, password);
    setTokens(res.tokens);
    localStorage.setItem(STORAGE_KEYS.user, JSON.stringify(res.user));
    setUser(res.user);
  }, []);

  const hasPerm = useCallback((perm: string) => !!user && user.permissions.includes(perm), [user]);

  const value = useMemo(() => ({ user, ready, login, logout, hasPerm }), [user, ready, login, logout, hasPerm]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
