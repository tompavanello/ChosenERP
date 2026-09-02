"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Church, Lock, Mail, Eye, EyeOff, Loader2 } from "lucide-react";
import { useAuth } from "@/components/providers/auth-provider";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [email, setEmail] = useState("admin@demo.local");
  const [password, setPassword] = useState("admin123");
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await login(email, password);
      router.replace("/dashboard");
    } catch (err) {
      setError("Credenciais inválidas. Tente novamente.");
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white shadow-lg shadow-violet-500/30">
            <Church className="h-8 w-8" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Chosen ERP</h1>
            <p className="text-sm text-zinc-500">Gestão eclesiástica completa</p>
          </div>
        </div>

        <form onSubmit={onSubmit} className="card space-y-4">
          <div className="space-y-1">
            <label className="label" htmlFor="email">E-mail</label>
            <div className="relative">
              <Mail className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <input id="email" type="email" className="input pl-9" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
          </div>
          <div className="space-y-1">
            <label className="label" htmlFor="password">Senha</label>
            <div className="relative">
              <Lock className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <input id="password" type={showPw ? "text" : "password"} className="input pl-9 pr-9" value={password} onChange={(e) => setPassword(e.target.value)} />
              <button type="button" onClick={() => setShowPw((v) => !v)} className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400">
                {showPw ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <button type="submit" className="btn-primary btn-base w-full" disabled={loading}>
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Entrar"}
          </button>
        </form>

        <div className="mt-4 rounded-lg bg-zinc-100 p-3 text-center text-xs text-zinc-500 dark:bg-zinc-900">
          Dev: <b>admin@demo.local</b> / <b>admin123</b>
        </div>
      </div>
    </main>
  );
}
