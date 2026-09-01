"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Church, LayoutDashboard, Users, Wallet, LogOut, HeartHandshake, DoorOpen, BarChart3 } from "lucide-react";

const nav = [
  { href: "/dashboard", label: "Visão Geral", icon: LayoutDashboard },
  { href: "/dashboard/members", label: "Membros", icon: Users },
  { href: "/dashboard/visitors", label: "Visitantes", icon: DoorOpen },
  { href: "/dashboard/benefactors", label: "Benfeitores", icon: HeartHandshake },
  { href: "/dashboard/finance", label: "Financeiro", icon: Wallet },
  { href: "/dashboard/reports", label: "Relatórios", icon: BarChart3 },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const user = typeof window !== "undefined" ? localStorage.getItem("chosen_user") : null;

  function logout() {
    localStorage.removeItem("chosen_token");
    localStorage.removeItem("chosen_user");
    router.replace("/");
  }

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-200 bg-white p-5">
        <div className="mb-8 flex items-center gap-2 font-semibold">
          <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-violet-700 text-white">
            <Church className="h-5 w-5" />
          </span>
          <span className="text-lg">Chosen ERP</span>
        </div>
        <nav className="space-y-1">
          {nav.map((item) => {
            const active = pathname === item.href;
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                  active ? "bg-violet-50 text-violet-700" : "text-zinc-600 hover:bg-zinc-100"
                }`}
              >
                <Icon className="h-4 w-4" /> {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="mt-auto border-t border-zinc-200 pt-4">
          {user && (
            <p className="mb-2 truncate px-3 text-xs text-zinc-500">
              {JSON.parse(user).full_name}
            </p>
          )}
          <button onClick={logout} className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm text-zinc-500 hover:bg-zinc-100">
            <LogOut className="h-4 w-4" /> Sair
          </button>
        </div>
      </aside>
      <main className="flex-1 p-8">{children}</main>
    </div>
  );
}
