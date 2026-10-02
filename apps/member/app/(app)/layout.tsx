"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import Link from "next/link";
import { Bell, Calendar, Church, Heart, Home, LogOut, User } from "lucide-react";
import {
  clearSession,
  fetchMe,
  getPublicTenant,
  getToken,
  tenantSlugFromHost,
  type PublicTenant,
} from "@/lib/api";

const NAV = [
  { href: "/home", label: "Inicio", icon: Home },
  { href: "/agenda", label: "Agenda", icon: Calendar },
  { href: "/avisos", label: "Avisos", icon: Bell },
  { href: "/oracao", label: "Oracao", icon: Heart },
  { href: "/perfil", label: "Perfil", icon: User },
];

export default function AppShell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);
  const [tenant, setTenant] = useState<PublicTenant | null>(null);

  useEffect(() => {
    if (!getToken()) {
      router.replace("/");
      return;
    }
    setReady(true);
    // Senha provisoria: forca a troca antes de usar o app.
    fetchMe()
      .then((me) => {
        if (me.must_change_password) router.replace("/trocar-senha");
      })
      .catch(() => {});
    const slug = tenantSlugFromHost();
    if (slug) {
      getPublicTenant(slug).then(setTenant).catch(() => {});
    }
  }, [router]);

  function logout() {
    clearSession();
    router.replace("/");
  }

  if (!ready) return null;
  const brand = tenant?.brand_color || undefined;

  return (
    <div
      className="mx-auto flex min-h-screen max-w-md flex-col bg-[var(--paper)]"
      style={brand ? ({ ["--brand" as string]: brand } as React.CSSProperties) : undefined}
    >
      <header className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--line)] bg-[var(--card)] px-4 py-3">
        <div className="flex items-center gap-2">
          {tenant?.logo_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={tenant.logo_url} alt={tenant.name} className="h-8 w-8 rounded-lg object-contain" />
          ) : (
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--brand)] text-white">
              <Church className="h-4 w-4" />
            </span>
          )}
          <span className="text-sm font-semibold">{tenant?.name ?? "App do Membro"}</span>
        </div>
        <button onClick={logout} className="text-[var(--muted)]" aria-label="Sair">
          <LogOut className="h-5 w-5" />
        </button>
      </header>

      <main className="flex-1 px-4 pb-24 pt-4">{children}</main>

      <nav className="fixed inset-x-0 bottom-0 z-10 mx-auto flex max-w-md items-center justify-around border-t border-[var(--line)] bg-[var(--card)] py-2">
        {NAV.map(({ href, label, icon: Icon }) => {
          const active = pathname === href;
          return (
            <Link
              key={href}
              href={href}
              className={`flex flex-col items-center gap-0.5 px-3 text-[11px] ${
                active ? "text-[var(--brand)]" : "text-[var(--muted)]"
              }`}
            >
              <Icon className="h-5 w-5" />
              {label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
