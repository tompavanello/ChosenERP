"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronRight, Home } from "lucide-react";
import { cn } from "@/lib/utils";

const BREADCRUMB_LABELS: Record<string, string> = {
  dashboard: "Dashboard",
  members: "Membros",
  visitors: "Visitantes",
  benefactors: "Benfeitores",
  finance: "Financeiro",
  transfers: "Repasses",
  ministries: "Ministérios",
  announcements: "Comunicados",
  reports: "Relatórios",
};

// Segmentos que são identificadores (UUID) e não nomes de rota.
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function Breadcrumbs({ className }: { className?: string }) {
  const pathname = usePathname();
  const segments = pathname.split("/").filter(Boolean);

  const crumbs = segments.map((segment, index) => {
    // Um UUID na URL é rota de detalhe: não vira link com o UUID no texto.
    const isId = UUID_RE.test(segment);
    const label = isId
      ? "Detalhes"
      : BREADCRUMB_LABELS[segment] ?? segment.charAt(0).toUpperCase() + segment.slice(1);

    return { label, href: "/" + segments.slice(0, index + 1).join("/"), isId };
  });

  // Não mostra breadcrumb na home do dashboard.
  if (crumbs.length <= 1) return null;

  return (
    <nav
      aria-label="breadcrumb"
      className={cn("flex min-w-0 items-center gap-1 text-sm text-zinc-500", className)}
    >
      <Link href="/dashboard" className="flex shrink-0 items-center gap-1 text-zinc-400 hover:text-zinc-700">
        <Home className="h-3.5 w-3.5" />
        Dashboard
      </Link>
      {crumbs.slice(1).map((crumb, i) => (
        <span key={crumb.href} className="flex min-w-0 items-center gap-1">
          <ChevronRight className="h-3.5 w-3.5 shrink-0 text-zinc-400" />
          {crumb.isId ? (
            <span className="font-medium text-zinc-700">{crumb.label}</span>
          ) : (
            <Link
              href={crumb.href}
              className={cn(
                "truncate hover:text-zinc-700",
                i === crumbs.length - 2 ? "font-medium text-zinc-700" : "text-zinc-400",
              )}
            >
              {crumb.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
}
