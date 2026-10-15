"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Church, KeyRound, LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Field } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import {
  fetchMe,
  getPublicTenant,
  getToken,
  login,
  requestVisitorCode,
  tenantSlugFromHost,
  verifyVisitorCode,
  VISITOR_ROLE,
  type PublicTenant,
} from "@/lib/api";

type Mode = "member" | "visitor";

export default function LoginPage() {
  const router = useRouter();
  const [slug, setSlug] = useState("");
  const [tenant, setTenant] = useState<PublicTenant | null>(null);
  const [mode, setMode] = useState<Mode>("member");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const s = tenantSlugFromHost();
    setSlug(s);
    if (getToken()) {
      fetchMe()
        .then((me) => router.replace(me.role === VISITOR_ROLE ? "/visita" : "/home"))
        .catch(() => {});
    }
    if (s) {
      getPublicTenant(s)
        .then(setTenant)
        .catch(() => {});
    }
  }, [router]);

  async function submitMember(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      const res = await login(email.trim(), password);
      if (res.requires_tenant_selection) {
        setError("Sua conta tem mais de uma igreja. Use o link direto da igreja desejada.");
        return;
      }
      const me = await fetchMe();
      if (me.must_change_password) {
        router.replace("/trocar-senha");
        return;
      }
      router.replace("/home");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nao foi possivel entrar.");
    } finally {
      setLoading(false);
    }
  }

  async function requestCode(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await requestVisitorCode(phone.trim(), name.trim());
      setCodeSent(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Nao foi possivel enviar o codigo.");
    } finally {
      setLoading(false);
    }
  }

  async function submitCode(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    try {
      await verifyVisitorCode(phone.trim(), code.trim());
      await fetchMe();
      router.replace("/visita");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Codigo invalido.";
      setError(msg === "use_password" ? "Este telefone ja tem acesso. Entre com e-mail/telefone e senha." : msg);
    } finally {
      setLoading(false);
    }
  }

  const brand = tenant?.brand_color || undefined;

  return (
    <main
      className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-10"
      style={brand ? ({ ["--brand" as string]: brand } as React.CSSProperties) : undefined}
    >
      <header className="mb-8 flex flex-col items-center text-center">
        {tenant?.logo_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={tenant.logo_url} alt={tenant.name} className="mb-4 h-16 w-16 rounded-2xl object-contain" />
        ) : (
          <span className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-[var(--brand)] text-white">
            <Church className="h-7 w-7" />
          </span>
        )}
        <h1 className="text-xl font-bold">{tenant?.name ?? "App do Membro"}</h1>
        <p className="mt-1 text-sm text-[var(--muted)]">
          {slug ? `Bem-vindo a ${tenant?.name ?? slug}` : "Entre com sua conta"}
        </p>
      </header>

      <div className="mb-4 flex gap-2">
        <button
          type="button"
          onClick={() => {
            setMode("member");
            setError(null);
          }}
          className={`flex-1 rounded-lg py-2 text-sm font-medium ${
            mode === "member" ? "bg-[var(--brand)] text-white" : "bg-black/5 text-[var(--muted)]"
          }`}
        >
          Sou membro
        </button>
        <button
          type="button"
          onClick={() => {
            setMode("visitor");
            setError(null);
          }}
          className={`flex-1 rounded-lg py-2 text-sm font-medium ${
            mode === "visitor" ? "bg-[var(--brand)] text-white" : "bg-black/5 text-[var(--muted)]"
          }`}
        >
          Sou visitante
        </button>
      </div>

      <Card>
        {mode === "member" ? (
          <form onSubmit={submitMember} className="space-y-4">
            <Field label="E-mail ou telefone">
              <Input
                type="text"
                autoComplete="username"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="voce@email.com ou (00) 00000-0000"
              />
            </Field>
            <Field label="Senha">
              <Input
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
              />
            </Field>
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" disabled={loading} className="w-full">
              <LogIn className="h-4 w-4" />
              {loading ? "Entrando..." : "Entrar"}
            </Button>
          </form>
        ) : !codeSent ? (
          <form onSubmit={requestCode} className="space-y-4">
            <Field label="Seu nome">
              <Input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="Como podemos te chamar?"
              />
            </Field>
            <Field label="WhatsApp">
              <Input
                type="tel"
                autoComplete="tel"
                required
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="(00) 00000-0000"
              />
            </Field>
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" disabled={loading || !phone.trim()} className="w-full">
              <KeyRound className="h-4 w-4" />
              {loading ? "Enviando..." : "Receber código por WhatsApp"}
            </Button>
            <p className="text-center text-xs text-[var(--muted)]">
              Enviaremos um código de 6 dígitos para o seu WhatsApp.
            </p>
          </form>
        ) : (
          <form onSubmit={submitCode} className="space-y-4">
            <p className="text-sm text-[var(--muted)]">
              Digite o código enviado para <span className="font-medium">{phone}</span>.
            </p>
            <Field label="Código">
              <Input
                type="text"
                inputMode="numeric"
                autoComplete="one-time-code"
                required
                value={code}
                onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                placeholder="000000"
                className="tnum text-center text-lg tracking-[0.4em]"
              />
            </Field>
            {error && <p className="text-sm text-red-600">{error}</p>}
            <Button type="submit" disabled={loading || code.length < 6} className="w-full">
              <LogIn className="h-4 w-4" />
              {loading ? "Entrando..." : "Entrar"}
            </Button>
            <button
              type="button"
              onClick={() => {
                setCodeSent(false);
                setCode("");
                setError(null);
              }}
              className="w-full text-center text-xs text-[var(--muted)]"
            >
              Trocar número / reenviar
            </button>
          </form>
        )}
      </Card>

      <p className="mt-6 text-center text-xs text-[var(--muted)]">Chosen ERP · App do Membro</p>
    </main>
  );
}
