"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Church } from "lucide-react";
import { login } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("admin@demo.local");
  const [password, setPassword] = useState("admin123");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await login(email, password);
      localStorage.setItem("chosen_token", res.tokens.access_token);
      localStorage.setItem("chosen_user", JSON.stringify(res.user));
      router.push("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Falha ao autenticar");
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-2 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-violet-700 text-white">
            <Church className="h-8 w-8" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Chosen ERP</h1>
            <p className="text-sm text-zinc-500">Gestão eclesiástica completa</p>
          </div>
        </div>

        <form onSubmit={onSubmit} className="card space-y-4">
          <div>
            <label className="label" htmlFor="email">E-mail</label>
            <input id="email" type="email" className="input" value={email}
              onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div>
            <label className="label" htmlFor="password">Senha</label>
            <input id="password" type="password" className="input" value={password}
              onChange={(e) => setPassword(e.target.value)} />
          </div>
          {error && <p className="text-sm text-red-600">{error}</p>}
          <button type="submit" className="btn-primary w-full" disabled={loading}>
            {loading ? "Entrando..." : "Entrar"}
          </button>
        </form>
        <p className="mt-4 text-center text-xs text-zinc-400">
          Dev: admin@demo.local · admin123
        </p>
      </div>
    </main>
  );
}
