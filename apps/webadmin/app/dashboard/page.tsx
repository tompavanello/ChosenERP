"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  ArrowUpRight, Wallet, Users, DoorOpen,
  UserPlus, BarChart3, HandHeart,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/components/providers/auth-provider";
import { getBalance, getInsights, getMonthlyBalance, listMembers, listVisitors, type Balance, type Insight, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel, number } from "@/lib/format";
import { ACCOUNT_TYPES } from "@/lib/constants";
import { Skeleton } from "@/components/ui/skeleton";
import { BirthdaysSummaryCard } from "@/components/reports/birthdays-summary-card";
import { OnboardingChecklist } from "@/components/help/onboarding";
import { KpiCard } from "@/components/reports/kpi-card";
import { InsightList } from "@/components/reports/insight-card";
import { ReportSection } from "@/components/reports/report-section";
import { TrendChart } from "@/components/reports/trend-chart";
import { COLORS, currencyCompact } from "@/components/reports/chart-theme";

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
  const [insights, setInsights] = useState<Insight[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const load = async () => {
      if (hasPerm("finance.read")) {
        const [bal, s, m, v] = await Promise.all([
          getBalance().catch(() => null),
          hasPerm("reports.read") ? getMonthlyBalance().catch(() => null) : Promise.resolve(null),
          listMembers().then((r) => r.members.length).catch(() => null),
          listVisitors().then((r) => r.visitors.length).catch(() => null),
        ]);
        setBalance(bal);
        setSeries(s?.series ?? []);
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
      if (hasPerm("reports.read")) {
        getInsights({ scope: "all" })
          .then((r) => setInsights(r.insights.slice(0, 3)))
          .catch(() => setInsights([]));
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

      <OnboardingChecklist />

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
        <KpiCard label="Membros" value={members != null ? number(members) : "..."} icon={Users} tone="sky" />
        <KpiCard label="Visitantes" value={visitors != null ? number(visitors) : "..."} icon={DoorOpen} tone="amber" />
        {hasPerm("finance.read") && (
          <>
            <KpiCard label="Entradas" value={balance ? currency(balance.income) : "..."} icon={ArrowUpRight} tone="green" spark={series.map((p) => p.income)} />
            <KpiCard label="Saldo" value={balance ? currency(balance.net) : "..."} icon={Wallet} tone="sky" spark={series.map((p) => p.net)} />
          </>
        )}
      </div>

      {hasPerm("finance.read") && (balance?.by_account_type?.length ?? 0) > 0 && (
        <div className="mt-3 flex flex-wrap gap-2 text-xs">
          {(balance?.by_account_type ?? []).map((b) => (
            <span key={b.account_type} className="rounded border border-zinc-200 bg-white px-2.5 py-1 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-950 dark:text-zinc-300">
              {ACCOUNT_TYPES[b.account_type] ?? b.account_type}: <b className="tabular-nums">{currency(b.net)}</b>
            </span>
          ))}
        </div>
      )}

      {/* Diagnostico do periodo (regras deterministicas) */}
      {hasPerm("reports.read") && insights.length > 0 && (
        <div className="mt-2">
          <div className="mb-2 flex items-center justify-between">
            <h3 className="text-sm font-medium text-zinc-500">O que merece atencao</h3>
            <Link href="/dashboard/reports/insights" className="text-xs text-sky-600 hover:underline">
              Ver diagnostico completo {"->"}
            </Link>
          </div>
          <InsightList insights={insights} />
        </div>
      )}

      {/* Aniversariantes do mes - resumo (total do mes e quantos hoje) */}
      {hasPerm("members.read") && <BirthdaysSummaryCard className="mt-6" />}

      {/* Chart */}
      {hasPerm("finance.read") && series.length > 0 && (
        <ReportSection
          title="Evolucao do saldo (mensal)"
          subtitle={`${series.length} mes(es) no periodo`}
          insight={
            series.length > 1
              ? `Ultimo mes: ${currency(series[series.length - 1].net)}. Variacao de ${(
                  ((series[series.length - 1].net - series[series.length - 2].net) /
                    Math.max(Math.abs(series[series.length - 2].net), 1)) *
                  100
                ).toFixed(1)}% sobre o mes anterior.`
              : undefined
          }
        >
          <TrendChart
            data={chartData}
            series={[{ key: "Saldo", label: "Saldo", color: COLORS.net, type: "area" }]}
            height={260}
            format={currencyCompact}
          />
        </ReportSection>
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
