"use client";

import { useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet, Users, DoorOpen, TrendingUp, Activity } from "lucide-react";
import {
  ResponsiveContainer, AreaChart, Area, XAxis, YAxis, Tooltip, CartesianGrid,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { StatCard } from "@/components/ui/stat-card";
import { useAuth } from "@/components/providers/auth-provider";
import { getBalance, getMonthlyBalance, listMembers, listVisitors, type Balance, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel, number } from "@/lib/format";
import { Skeleton } from "@/components/ui/skeleton";

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
        <PageHeader title="Visão Geral" />
        <div className="grid gap-4 md:grid-cols-3">
          {[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}
        </div>
      </div>
    );
  }

  const chartData = series.map((p) => ({ name: monthLabel(p.month), Saldo: p.net, Entradas: p.income, Saídas: p.expense }));

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader title="Visão Geral" description="Resumo operacional e financeiro da sua unidade." />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {hasPerm("finance.read") ? (
          <>
            <StatCard label="Entradas" value={balance ? currency(balance.income) : "..."} icon={ArrowUpRight} tone="green" />
            <StatCard label="Saídas" value={balance ? currency(balance.expense) : "..."} icon={ArrowDownRight} tone="red" />
            <StatCard label="Saldo" value={balance ? currency(balance.net) : "..."} icon={Wallet} tone="violet" />
          </>
        ) : (
          <StatCard label="Membros" value={members != null ? number(members) : "..."} icon={Users} tone="sky" />
        )}
        <StatCard label="Membros" value={members != null ? number(members) : "..."} icon={Users} tone="sky" />
        <StatCard label="Visitantes" value={visitors != null ? number(visitors) : "..."} icon={DoorOpen} tone="amber" />
      </div>

      {hasPerm("finance.read") && (
        <div className="card mt-6">
          <div className="mb-4 flex items-center justify-between">
            <h3 className="flex items-center gap-2 text-sm font-semibold text-zinc-700 dark:text-zinc-200">
              <TrendingUp className="h-4 w-4 text-violet-600" /> Evolução do saldo (mensal)
            </h3>
            <span className="flex items-center gap-2 text-xs text-zinc-400">
              <Activity className="h-3.5 w-3.5" /> {series.length} mês(es)
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
        </div>
      )}
    </div>
  );
}
