"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { 
  ArrowDownRight, ArrowUpRight, Wallet, Users, DoorOpen, 
  TrendingUp, Activity, UserPlus, BarChart3, HandHeart, FileText 
} from "lucide-react";
import {
  ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { StatCard } from "@/components/ui/stat-card";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/components/providers/auth-provider";
import { getBalance, getMonthlyBalance, listMembers, listVisitors, type Balance, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel, number } from "@/lib/format";
import { Skeleton } from "@/components/ui/skeleton";
import { BirthdaysSummaryCard } from "@/components/reports/birthdays-summary-card";

// Quick nav shortcuts
const QUICK_ACTIONS = [
  { label: "Novo Membro", icon: UserPlus, href: "/dashboard/members", perm: "members.write" },
  { label: "Novo Visitante", icon: DoorOpen, href: "/dashboard/visitors", perm: "members.write" },
  { label: "Novo Benfeitor", icon: HandHeart, href: "/dashboard/benefactors", perm: "members.write" },
  // "Nova Familia" saiu: a familia agora e criada dentro da ficha do membro
  // (aba Familia), que e onde o agrupamento tem contexto.
  { label: "Ver Familias", icon: Users, href: "/dashboard/members", perm: "families.read" },
  { label: "Lancar Financeiro", icon: Wallet, href: "/dashboard/finance", perm: "finance.write" },
  { label: "Relatorios", icon: BarChart3, href: "/dashboard/reports", perm: "reports.read" },
];

export default function OverviewPage() {
  const { hasPerm } = useAuth();
  const [balance, setBalance] = useState<Balance | null>(null);
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [members, setMembers] = useState<number | null>(null);
  const [visitors, setVisitors] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      if (hasPerm("finance.read")) {
        const [bal, s, m, v] = await Promise.all([
          getBalance(),
          getMonthlyBalance(),
          listMembers().then((r) => r.members.length).catch(() => null),
          listVisitors().then((r) => r.visitors.length).catch(() => null),
        ]);
        setBalance(bal);
        setSeries(s.series);
        setMembers(m);
        setVisitors(v);
      } else {
        const [m, v] = await Promise.all([
          listMembers().then((r) => r.members.length).catch(() => null),
          listVisitors().then((r) => r.visitors.length).catch(() => null),
        ]);
        setMembers(m);
        setVisitors(v);
      }
      setLoading(false);
    };
    load().catch(() => setLoading(false));
  }, [hasPerm]);

  if (loading) {
    return (
      <div>
        <PageHeader title="Visao Geral" />
        <div className="grid gap-4 md:grid-cols-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}
        </div>
      </div>
    );
  }

  const chartData = series.map((p) => ({ name: monthLabel(p.month), Saldo: p.net, Entradas: p.income, Saidas: p.expense }));
  const visibleActions = QUICK_ACTIONS.filter((a) => hasPerm(a.perm));

  return (
    <div className="page space-y-6">
      <PageHeader title="Visao Geral" description="Resumo operacional e financeiro da sua unidade." />

      {/* Quick Nav Cards */}
      <div className="mb-4">
        <h3 className="text-sm font-medium text-zinc-500 mb-3">Acesso rapido</h3>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
          {visibleActions.map((action) => {
            const Icon = action.icon;
            return (
              <Link key={action.label} href={action.href}>
                <Card className="p-4 text-center transition-all hover:shadow-md hover:border-sky-200 group">
                  <div className="flex flex-col items-center gap-2">
                    <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-50 text-sky-600">
                      <Icon className="h-5 w-5 group-hover:scale-110 transition-transform" />
                    </div>
                    <span className="text-xs font-medium text-zinc-700">{action.label}</span>
                  </div>
                </Card>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {hasPerm("finance.read") ? (
          <>
            <StatCard label="Entradas" value={balance ? currency(balance.income) : "..."} icon={ArrowUpRight} tone="green" />
            <StatCard label="Saidas" value={balance ? currency(balance.expense) : "..."} icon={ArrowDownRight} tone="red" />
            <StatCard label="Saldo" value={balance ? currency(balance.net) : "..."} icon={Wallet} tone="sky" />
          </>
        ) : (
          <StatCard label="Membros" value={members != null ? number(members) : "..."} icon={Users} tone="sky" />
        )}
        <StatCard label="Membros" value={members != null ? number(members) : "..."} icon={Users} tone="sky" />
        <StatCard label="Visitantes" value={visitors != null ? number(visitors) : "..."} icon={DoorOpen} tone="amber" />
      </div>

      {/* Aniversariantes do mes - resumo (total do mes e quantos hoje) */}
      {hasPerm("members.read") && <BirthdaysSummaryCard className="mt-6" />}

      {/* Chart */}
      {hasPerm("finance.read") && series.length > 0 && (
        <Card className="p-6">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-zinc-700">
              <TrendingUp className="h-4 w-4 text-sky-600" /> Evolucao do saldo (mensal)
            </h3>
            <span className="flex items-center gap-2 text-xs text-zinc-400">
              <Activity className="h-3.5 w-3.5" /> {series.length} mes(es)
            </span>
          </div>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="net" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} />
                <Tooltip formatter={(v: unknown) => currency(Number(v))} labelStyle={{ color: "#111" }} />
                <Area type="monotone" dataKey="Saldo" stroke="#8b5cf6" strokeWidth={2.5} fill="url(#net)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </Card>
      )}

      {/* Recent activity placeholder */}
      {hasPerm("finance.read") && balance && balance.net > 0 && (
        <Card className="p-6">
          <h3 className="mb-3 text-sm font-semibold text-zinc-700">Ultimos lancamentos</h3>
          <p className="text-sm text-zinc-500">Sem lancamentos recentes.</p>
        </Card>
      )}
    </div>
  );
}
