"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Church, LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Field } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import {
  fetchMe,
  getPublicTenant,
  getToken,
  login,
  tenantSlugFromHost,
  type PublicTenant,
} from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [slug, setSlug] = useState("");
  const [tenant, setTenant] = useState<PublicTenant | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const s = tenantSlugFromHost();
    setSlug(s);
    if (getToken()) {
      fetchMe()
        .then(() => router.replace("/home"))
        .catch(() => {});
    }
    if (s) {
      getPublicTenant(s)
        .then(setTenant)
        .catch(() => {});
    }
  }, [router]);

  async function submit(e: React.FormEvent) {
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

      <Card>
        <form onSubmit={submit} className="space-y-4">
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
      </Card>

      <p className="mt-6 text-center text-xs text-[var(--muted)]">
        Chosen ERP · App do Membro
      </p>
    </main>
  );
}
