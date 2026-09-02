"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  Church, LayoutDashboard, Users, Wallet, LogOut, HeartHandshake, DoorOpen,
  BarChart3, Users2, Menu, Search, ChevronRight, Building2,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/components/providers/auth-provider";
import { ThemeToggle } from "@/components/theme-toggle";
import { Avatar } from "@/components/ui/avatar";

const NAV = [
  { key: "overview", href: "/dashboard", label: "Visão Geral", icon: LayoutDashboard, perms: [] },
  { key: "members", href: "/dashboard/members", label: "Membros", icon: Users, perms: ["members.read"] },
  { key: "families", href: "/dashboard/families", label: "Famílias", icon: Users2, perms: ["families.read"] },
  { key: "visitors", href: "/dashboard/visitors", label: "Visitantes", icon: DoorOpen, perms: ["members.read"] },
  { key: "benefactors", href: "/dashboard/benefactors", label: "Benfeitores", icon: HeartHandshake, perms: ["members.read"] },
  { key: "finance", href: "/dashboard/finance", label: "Financeiro", icon: Wallet, perms: ["finance.read"] },
  { key: "reports", href: "/dashboard/reports", label: "Relatórios", icon: BarChart3, perms: ["reports.read", "finance.read"] },
];

const RESOURCE_LABELS: Record<string, string> = {
  members: "Membros",
  families: "Famílias",
  visitors: "Visitantes",
  benefactors: "Benfeitores",
  finance: "Financeiro",
  reports: "Relatórios",
};

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, ready, logout, hasPerm } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (ready && !user) router.replace("/");
  }, [ready, user, router]);

  useEffect(() => setMobileOpen(false), [pathname]);

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center text-sm text-zinc-500">
        Carregando sessão...
      </div>
    );
  }
  if (!user) return null;

  const visibleNav = NAV.filter((n) => n.perms.length === 0 || n.perms.some((p) => hasPerm(p)));
  const seg = pathname.split("/").filter(Boolean);
  const crumb = seg.length > 1 ? RESOURCE_LABELS[seg[1]] : "Visão Geral";

  const Sidebar = (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-950">
      <div className="mb-6 flex items-center gap-2 px-2 font-semibold">
        <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white">
          <Church className="h-5 w-5" />
        </span>
        <span className="text-lg">Chosen ERP</span>
      </div>
      <nav className="space-y-1">
        {visibleNav.map((item) => {
          const active = pathname === item.href;
          const Icon = item.icon;
          return (
            <Link
              key={item.key}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition",
                active
                  ? "bg-violet-50 text-violet-700 dark:bg-violet-500/10 dark:text-violet-300"
                  : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-900",
              )}
            >
              <Icon className="h-4 w-4" /> {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="mt-auto space-y-2 border-t border-zinc-200 pt-4 dark:border-zinc-800">
        <div className="flex items-center gap-2 px-2">
          <Avatar name={user.full_name} size="sm" />
          <div className="min-w-0">
            <p className="truncate text-sm font-medium">{user.full_name}</p>
            <p className="truncate text-xs text-zinc-400">{user.role}</p>
          </div>
        </div>
        <button onClick={logout} className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-900">
          <LogOut className="h-4 w-4" /> Sair
        </button>
      </div>
    </aside>
  );

  return (
    <div className="flex min-h-screen">
      {/* Desktop sidebar */}
      <div className="hidden lg:block">{Sidebar}</div>

      {/* Mobile sidebar */}
      {mobileOpen && (
        <div className="fixed inset-0 z-40 lg:hidden">
          <div className="absolute inset-0 bg-black/40" onClick={() => setMobileOpen(false)} />
          <div className="absolute left-0 top-0 h-full">{Sidebar}</div>
        </div>
      )}

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex items-center gap-3 border-b border-zinc-200 bg-white/80 px-4 py-3 backdrop-blur dark:border-zinc-800 dark:bg-zinc-950/80">
          <button className="rounded-lg p-2 text-zinc-500 hover:bg-zinc-100 lg:hidden dark:hover:bg-zinc-800" onClick={() => setMobileOpen(true)}>
            <Menu className="h-5 w-5" />
          </button>
          <div className="flex items-center gap-1 text-sm text-zinc-500">
            <span>Dashboard</span>
            <ChevronRight className="h-3.5 w-3.5" />
            <span className="font-medium text-zinc-700 dark:text-zinc-200">{crumb}</span>
          </div>

          <div className="ml-auto flex items-center gap-2">
            <div className="relative hidden md:block">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <input className="input w-56 pl-9" placeholder="Buscar..." />
            </div>
            <div className="hidden items-center gap-1 rounded-lg border border-zinc-200 px-2.5 py-1.5 text-xs text-zinc-500 md:flex dark:border-zinc-800">
              <Building2 className="h-3.5 w-3.5" />
              {user.branch_id || "Sede"}
            </div>
            <ThemeToggle />
            <Avatar name={user.full_name} size="sm" />
          </div>
        </header>

        <main className="flex-1 p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  );
}
