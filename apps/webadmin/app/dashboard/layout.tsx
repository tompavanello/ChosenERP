"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  Church, LayoutDashboard, Users, Wallet, LogOut, HeartHandshake, DoorOpen,
  BarChart3, Menu, Building2, ArrowLeftRight, Bell, Cake, PieChart, TrendingUp, UserCog, FileSpreadsheet, CalendarDays, Gavel, Settings, CalendarClock, Baby, Truck, ShieldCheck, ChevronDown, ChevronRight,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/components/providers/auth-provider";
import { useBranches } from "@/lib/swr-hooks";
import { getBranchContext, setBranchContext } from "@/lib/api";
import { ThemeToggle } from "@/components/theme-toggle";
import { Avatar } from "@/components/ui/avatar";
import { Breadcrumbs } from "@/components/ui/breadcrumbs";

type NavItem = {
  key: string;
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  perms: string[];
};

// Secoes do menu. "Familias" saiu daqui: a gestao de familia passou a viver
// dentro do cadastro do membro (aba Familia).
const NAV_SECTIONS: { title: string | null; items: NavItem[] }[] = [
  {
    title: null,
    items: [
      { key: "overview", href: "/dashboard", label: "Visao Geral", icon: LayoutDashboard, perms: [] },
    ],
  },
  {
    title: "Pessoas",
    items: [
      { key: "members", href: "/dashboard/members", label: "Membros", icon: Users, perms: ["members.read"] },
      { key: "visitors", href: "/dashboard/visitors", label: "Visitantes", icon: DoorOpen, perms: ["members.read"] },
      { key: "benefactors", href: "/dashboard/benefactors", label: "Benfeitores", icon: HeartHandshake, perms: ["members.read"] },
      { key: "suppliers", href: "/dashboard/suppliers", label: "Fornecedores", icon: Truck, perms: ["finance.read"] },
    ],
  },
  {
    title: "Financeiro",
    items: [
      { key: "finance", href: "/dashboard/finance", label: "Financeiro", icon: Wallet, perms: ["finance.read"] },
      { key: "transfers", href: "/dashboard/transfers", label: "Repasses", icon: ArrowLeftRight, perms: ["finance.write"] },
    ],
  },
  {
    title: "Organizacao",
    items: [
      { key: "ministries", href: "/dashboard/ministries", label: "Ministerios", icon: Church, perms: ["ministries.read"] },
      { key: "rosters", href: "/dashboard/rosters", label: "Escalas", icon: CalendarClock, perms: ["ministries.read"] },
      { key: "kids", href: "/dashboard/kids", label: "Kids", icon: Baby, perms: ["members.read"] },
      { key: "events", href: "/dashboard/events", label: "Eventos", icon: CalendarDays, perms: ["members.read"] },
      { key: "governance", href: "/dashboard/governance", label: "Governanca", icon: Gavel, perms: ["governance.read"] },
      { key: "users", href: "/dashboard/users", label: "Usuarios", icon: UserCog, perms: ["users.read"] },
      { key: "settings", href: "/dashboard/settings", label: "Configuracoes", icon: Settings, perms: ["settings.read"] },
      { key: "announcements", href: "/dashboard/announcements", label: "Comunicados", icon: Bell, perms: [] },
    ],
  },
  {
    title: "Relatorios",
    items: [
      { key: "rep-balance", href: "/dashboard/reports/balance", label: "Balancete mensal", icon: BarChart3, perms: ["finance.read"] },
      { key: "rep-dre", href: "/dashboard/reports/dre", label: "DRE", icon: TrendingUp, perms: ["finance.read"] },
      { key: "rep-statement", href: "/dashboard/reports/monthly-statement", label: "Demonstrativo Mensal", icon: FileSpreadsheet, perms: ["finance.read"] },
      { key: "rep-assembly", href: "/dashboard/reports/assembly", label: "Demonstrativo (Assembleia)", icon: FileSpreadsheet, perms: ["finance.read"] },
      { key: "rep-audit", href: "/dashboard/reports/audit", label: "Auditoria financeira", icon: ShieldCheck, perms: ["finance.read"] },
      { key: "rep-inc-exp", href: "/dashboard/reports/income-expense", label: "Entradas x Saidas", icon: ArrowLeftRight, perms: ["finance.read"] },
      { key: "rep-consolidated", href: "/dashboard/reports/consolidated", label: "Consolidado Sede > Filiais", icon: Building2, perms: ["finance.read"] },
      { key: "rep-birthdays", href: "/dashboard/reports/birthdays", label: "Aniversariantes", icon: Cake, perms: ["members.read"] },
      { key: "rep-demographics", href: "/dashboard/reports/demographics", label: "Demograficos", icon: PieChart, perms: ["members.read"] },
    ],
  },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, ready, logout, hasPerm, switchTenant } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);
  const { data: branchData } = useBranches();
  const [branchCtx, setBranchCtx] = useState("");
  useEffect(() => setBranchCtx(getBranchContext()), []);
  const isHQ = user?.role === "super_admin" || user?.role === "admin_sede";
  // Igrejas da identidade: o switcher so aparece quando ha mais de uma.
  const memberships = (user?.memberships ?? []).filter((m) => m.is_active);

  // Secoes do menu recolhiveis (persistidas por navegador).
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  useEffect(() => {
    try {
      const raw = localStorage.getItem("chosen_nav_collapsed");
      if (raw) setCollapsed(JSON.parse(raw));
    } catch {
      /* ignora storage invalido */
    }
  }, []);
  function toggleSection(title: string) {
    setCollapsed((c) => {
      const next = { ...c, [title]: !c[title] };
      try {
        localStorage.setItem("chosen_nav_collapsed", JSON.stringify(next));
      } catch {
        /* ignora */
      }
      return next;
    });
  }

  useEffect(() => {
    if (ready && !user) router.replace("/");
  }, [ready, user, router]);

  useEffect(() => setMobileOpen(false), [pathname]);

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
        Carregando sessao...
      </div>
    );
  }
  if (!user) return null;

  // Nome da filial em vez do UUID cru que aparecia na topbar.
  const branchName =
    branchData?.branches.find((b) => b.id === user.branch_id)?.name ??
    (user.branch_id ? null : "Sede");

  // Item ativo tambem nas sub-rotas (/dashboard/members/123 destaca "Membros").
  const isActive = (href: string) =>
    href === "/dashboard" ? pathname === href : pathname === href || pathname.startsWith(`${href}/`);

  const Sidebar = (
    <aside className="flex h-full w-56 shrink-0 flex-col border-r border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-950">
      <div className="flex items-center gap-2 px-4 py-3.5">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-sky-600 to-sky-500 text-white">
          <Church className="h-4 w-4" />
        </span>
        <span className="truncate text-[15px] font-semibold tracking-tight">Chosen ERP</span>
      </div>

      <nav className="flex-1 overflow-y-auto px-2 pb-2">
        {NAV_SECTIONS.map((section, i) => {
          const items = section.items.filter(
            (n) => n.perms.length === 0 || n.perms.some((p) => hasPerm(p)),
          );
          if (items.length === 0) return null;
          const open = !section.title || !collapsed[section.title];
          return (
            <div key={section.title ?? `section-${i}`}>
              {section.title && (
                <button
                  type="button"
                  onClick={() => toggleSection(section.title as string)}
                  aria-expanded={open}
                  className="flex w-full items-center justify-between gap-2 rounded-md px-2.5 pb-1 pt-3 text-[10px] font-semibold uppercase tracking-wide text-zinc-400 transition-colors hover:text-zinc-600 dark:hover:text-zinc-300"
                >
                  <span className="truncate">{section.title}</span>
                  {open ? <ChevronDown className="h-3.5 w-3.5 shrink-0" /> : <ChevronRight className="h-3.5 w-3.5 shrink-0" />}
                </button>
              )}
              {open && (
                <div className="space-y-0.5">
                  {items.map((item) => {
                    const active = isActive(item.href);
                    const Icon = item.icon;
                    return (
                      <Link
                        key={item.key}
                        href={item.href}
                        className={cn(
                          "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                          active
                            ? "bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300"
                            : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900 dark:hover:text-zinc-200",
                        )}
                      >
                        <Icon className="h-4 w-4 shrink-0" />
                        <span className="truncate">{item.label}</span>
                      </Link>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </nav>

      <div className="border-t border-zinc-200 p-2 dark:border-zinc-800">
        <Link href="/dashboard/profile" className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-zinc-100 dark:hover:bg-zinc-900">
          <Avatar name={user.full_name} size="sm" />
          <div className="min-w-0">
            <p className="truncate text-[13px] font-medium">{user.full_name}</p>
            <p className="truncate text-[11px] text-zinc-400">{user.role}</p>
          </div>
        </Link>
        <button
          onClick={logout}
          className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-[13px] text-zinc-500 transition-colors hover:bg-zinc-100 dark:hover:bg-zinc-900"
        >
          <LogOut className="h-4 w-4 shrink-0" /> Sair
        </button>
      </div>
    </aside>
  );

  return (
    <div className="flex min-h-screen">
      {/* Sidebar fixa: antes ela rolava junto com o conteudo da pagina. */}
      <div className="sticky top-0 hidden h-screen lg:block">{Sidebar}</div>

      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div className="absolute inset-0 bg-black/40" onClick={() => setMobileOpen(false)} />
          <div className="absolute left-0 top-0 h-full">{Sidebar}</div>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b border-zinc-200 bg-white/80 px-4 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
          <button
            className="rounded-md p-2 text-zinc-500 hover:bg-zinc-100 lg:hidden dark:hover:bg-zinc-800"
            onClick={() => setMobileOpen(true)}
          >
            <Menu className="h-5 w-5" />
          </button>

          <Breadcrumbs className="min-w-0 flex-1" />

          <div className="ml-auto flex shrink-0 items-center gap-2">
            {memberships.length > 1 && (
              <select
                value={user.tenant_id}
                onChange={(e) => switchTenant(e.target.value)}
                title="Igreja ativa"
                className="hidden rounded-md border border-zinc-200 bg-white px-2.5 py-1.5 text-xs text-zinc-600 sm:block dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300"
              >
                {memberships.map((m) => (
                  <option key={m.tenant_id} value={m.tenant_id}>{m.tenant_name}</option>
                ))}
              </select>
            )}
            {isHQ ? (
              <select
                value={branchCtx || "all"}
                onChange={(e) => {
                  setBranchContext(e.target.value);
                  window.location.reload();
                }}
                title="Filial de trabalho (lancamentos e consultas)"
                className="hidden rounded-md border border-zinc-200 bg-white px-2.5 py-1.5 text-xs text-zinc-600 sm:block dark:border-zinc-800 dark:bg-zinc-900 dark:text-zinc-300"
              >
                <option value="all">Sede (todas as filiais)</option>
                {(branchData?.branches ?? [])
                  .filter((b) => b.is_active !== false)
                  .map((b) => (
                    <option key={b.id} value={b.id}>{b.name}</option>
                  ))}
              </select>
            ) : branchName ? (
              <div className="hidden items-center gap-1.5 rounded-md border border-zinc-200 px-2.5 py-1.5 text-xs text-zinc-500 sm:flex dark:border-zinc-800">
                <Building2 className="h-3.5 w-3.5" />
                <span className="max-w-32 truncate">{branchName}</span>
              </div>
            ) : null}
            <ThemeToggle />
            <Link href="/dashboard/profile" title="Meu perfil" className="rounded-full transition-opacity hover:opacity-80">
              <Avatar name={user.full_name} size="sm" />
            </Link>
          </div>
        </header>

        <main className="flex-1 p-4 sm:p-5 lg:p-6">{children}</main>
      </div>
    </div>
  );
}
