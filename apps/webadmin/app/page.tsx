"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Church, Lock, Mail, Eye, EyeOff, Loader2, ShieldCheck, Building2 } from "lucide-react";
import { useAuth } from "@/components/providers/auth-provider";
import { getPublicTenant, tenantSlugFromHost, type PublicTenant } from "@/lib/api";

// A conveniencia de pre-preencher o login da demo vale so no `npm run dev`.
// No build de producao (o que o tunel publica) os campos saem vazios e a dica
// de credenciais nao e renderizada: a senha demo e rotacionada e vive no .env
// (DEMO_ADMIN_PASSWORD), nao no bundle. `process.env.NODE_ENV` e substituido em
// build time, entao isto e uma constante nos dois lados - nao gera divergencia
// de hidratacao.
//
// So o E-MAIL e pre-preenchido. A senha saiu daqui: ela era `admin123` fixo no
// codigo e, depois da rotacao das contas de demo, o formulario passou a abrir
// com uma senha que ja nao vale mais - clicar em Entrar dava 401 e parecia
// senha errada do usuario. A senha de verdade esta no .env; o campo comeca
// vazio nos dois modos.
const MODO_DEV = process.env.NODE_ENV !== "production";

export default function LoginPage() {
  const router = useRouter();
  const { login, pendingSelection, selectTenant } = useAuth();
  const [email, setEmail] = useState(MODO_DEV ? "admin@demo.local" : "");
  const [password, setPassword] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [code, setCode] = useState("");
  const [mfa, setMfa] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  // Branding do subdominio (igreja.dominio) + slug detectado do host.
  const [slug, setSlug] = useState("");
  const [brand, setBrand] = useState<PublicTenant | null>(null);

  useEffect(() => {
    const s = tenantSlugFromHost();
    setSlug(s);
    if (s) getPublicTenant(s).then(setBrand).catch(() => setBrand(null));
  }, []);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      // Com 1 igreja entra direto; com >1 o provider guarda a selecao pendente.
      const entered = await login(email, password, code || undefined, slug || undefined);
      if (entered) router.replace("/dashboard");
      else setLoading(false);
    } catch (err) {
      const msg = err instanceof Error ? err.message : "";
      if (msg.includes("mfa_required")) {
        setMfa(true);
        setError("Informe o codigo do autenticador.");
      } else if (msg.includes("mfa_invalid")) {
        setError("Codigo de verificacao invalido.");
      } else if (msg.includes("tenant_forbidden")) {
        setError("Sua conta nao tem acesso a esta igreja.");
      } else {
        setError("Credenciais invalidas. Tente novamente.");
      }
      setLoading(false);
    }
  }

  async function chooseTenant(id: string) {
    setError(null);
    setLoading(true);
    try {
      await selectTenant(id);
      router.replace("/dashboard");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nao foi possivel selecionar a igreja.");
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          <div
            className="flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-sky-600 to-sky-400 text-white shadow-lg shadow-sky-500/30"
            style={brand?.brand_color ? { background: brand.brand_color } : undefined}
          >
            <Church className="h-8 w-8" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{brand?.name ?? "Chosen ERP"}</h1>
            <p className="text-sm text-zinc-500">
              {brand ? "Acesso da igreja" : "Gestao eclesiastica completa"}
            </p>
          </div>
        </div>

        {pendingSelection ? (
          <div className="card space-y-3">
            <p className="text-sm text-zinc-500">Escolha a igreja que deseja acessar:</p>
            <div className="space-y-2">
              {pendingSelection.tenants.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  disabled={loading}
                  onClick={() => chooseTenant(t.id)}
                  className="flex w-full items-center gap-3 rounded-lg border border-zinc-200 px-3 py-2.5 text-left transition-colors hover:bg-zinc-50 disabled:opacity-60 dark:border-zinc-800 dark:hover:bg-zinc-900"
                >
                  <Building2 className="h-4 w-4 shrink-0 text-sky-500" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{t.name}</span>
                    <span className="block truncate text-xs text-zinc-400">{t.role}</span>
                  </span>
                  {loading && <Loader2 className="h-4 w-4 animate-spin text-zinc-400" />}
                </button>
              ))}
            </div>
            {error && <p className="text-sm text-red-600">{error}</p>}
          </div>
        ) : (
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
            {mfa && (
              <div className="space-y-1">
                <label className="label" htmlFor="code">Codigo do autenticador</label>
                <div className="relative">
                  <ShieldCheck className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
                  <input
                    id="code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    maxLength={6}
                    className="input pl-9 tracking-widest"
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  />
                </div>
              </div>
            )}
            {error && <p className="text-sm text-red-600">{error}</p>}
            <button type="submit" className="btn-primary btn-base w-full" disabled={loading}>
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Entrar"}
            </button>
          </form>
        )}

        {MODO_DEV && (
          <div className="mt-4 rounded-lg bg-zinc-100 p-3 text-center text-xs text-zinc-500">
            Dev: <b>admin@demo.local</b> - senha em <code>.env</code> (<code>DEMO_ADMIN_PASSWORD</code>)
          </div>
        )}
      </div>
    </main>
  );
}
