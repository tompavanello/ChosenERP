"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getDRE, getMonthlyBalance, type DRE, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";
import { ReportToolbar } from "@/components/reports/report-toolbar";
import { ReportSection } from "@/components/reports/report-section";
import { KpiCard } from "@/components/reports/kpi-card";
import { RankedBar } from "@/components/reports/ranked-bar";
import { TrendChart } from "@/components/reports/trend-chart";
import { COLORS, currencyCompact } from "@/components/reports/chart-theme";

function anoAtual() {
  return new Date().getFullYear();
}

/**
 * Entradas x Saidas: comparacao por categoria (barras ranqueadas) e evolucao
 * mensal. As antigas pizzas foram substituidas por barras/composicao, que sao
 * mais precisas para comparar valores.
 */
export default function IncomeExpensePage() {
  const { toast } = useToast();
  const ano = anoAtual();
  const [from, setFrom] = useState(`${ano}-01-01`);
  const [to, setTo] = useState(`${ano}-12-31`);
  const [dre, setDre] = useState<DRE | null>(null);
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      const [d, s] = await Promise.all([getDRE(f, t), getMonthlyBalance(f, t)]);
      setDre(d);
      setSeries(s.series);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load(from, to);
  }, [load, from, to]);

  const income = (dre?.lines ?? []).filter((l) => l.type === "income").map((l) => ({ name: l.category, value: l.total }));
  const expense = (dre?.lines ?? []).filter((l) => l.type === "expense").map((l) => ({ name: l.category, value: l.total }));
  const monthly = series.map((p) => ({ name: monthLabel(p.month), Entradas: p.income, Saidas: p.expense, Resultado: p.net }));
  const cmp = dre?.comparison;
  const deltaIncome = cmp && cmp.prev_income !== 0 ? ((dre!.income - cmp.prev_income) / Math.abs(cmp.prev_income)) * 100 : 0;
  const deltaExpense = cmp && cmp.prev_expense !== 0 ? ((dre!.expense - cmp.prev_expense) / Math.abs(cmp.prev_expense)) * 100 : 0;

  return (
    <div className="page">
      <PageHeader
        title="Entradas x Saidas de Recursos"
        description="Composicao das receitas/despesas e comparativo por periodo"
        actions={
          <ExportButtons
            path="/api/v1/reports/dre/export"
            params={{ from, to }}
            filenameBase={`entradas-saidas-${from}-${to}`}
          />
        }
      />

      <ReportToolbar
        from={from}
        to={to}
        onFrom={setFrom}
        onTo={setTo}
        onChange={load}
        actions={<span className="text-xs text-zinc-400">{dre?.lines.length ?? 0} conta(s)</span>}
      />

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2">{[0, 1].map((i) => <Skeleton key={i} className="h-48" />)}</div>
      ) : dre ? (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas" value={currency(dre.income)} icon={ArrowUpRight} tone="green" delta={deltaIncome} deltaLabel="vs. periodo anterior" />
            <KpiCard label="Saidas" value={currency(dre.expense)} icon={ArrowDownRight} tone="red" delta={deltaExpense} positiveIsGood={false} deltaLabel="vs. periodo anterior" />
            <KpiCard label="Resultado" value={currency(dre.net)} icon={Wallet} tone="sky" delta={cmp?.delta_pct} deltaLabel="vs. periodo anterior" />
            <KpiCard
              label="Margem de sobra"
              value={dre.income > 0 ? `${(((dre.income - dre.expense) / dre.income) * 100).toFixed(1)}%` : "-"}
              tone="indigo"
              hint="resultado / entradas"
            />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection
              title="De onde vem a receita?"
              subtitle="Categorias de entrada ordenadas por volume"
              insight={income.length > 0 ? `A maior fonte representa ${((income[0].value / Math.max(dre.income, 1)) * 100).toFixed(0)}% das entradas.` : undefined}
            >
              <RankedBar data={income} total={dre.income} color={COLORS.income} emptyLabel="Sem entradas no periodo." />
            </ReportSection>
            <ReportSection
              title="Para onde vai o dinheiro?"
              subtitle="Categorias de saida ordenadas por volume"
              insight={expense.length > 0 ? `A maior despesa representa ${((expense[0].value / Math.max(dre.expense, 1)) * 100).toFixed(0)}% das saidas.` : undefined}
            >
              <RankedBar data={expense} total={dre.expense} color={COLORS.expense} emptyLabel="Sem saidas no periodo." />
            </ReportSection>
          </div>

          <ReportSection
            title="Entradas, saidas e resultado por mes"
            subtitle="Barras comparam valores; a linha mostra o resultado do mes"
            className="mb-4"
          >
            <TrendChart
              data={monthly}
              series={[
                { key: "Entradas", label: "Entradas", color: COLORS.income, type: "bar" },
                { key: "Saidas", label: "Saidas", color: COLORS.expense, type: "bar" },
                { key: "Resultado", label: "Resultado", color: COLORS.net, type: "line" },
              ]}
              format={currencyCompact}
            />
          </ReportSection>

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Comparativo label="Entradas" atual={dre.income} anterior={cmp?.prev_income ?? 0} positivo />
            <Comparativo label="Saidas" atual={dre.expense} anterior={cmp?.prev_expense ?? 0} />
            <Comparativo label="Resultado" atual={dre.net} anterior={cmp?.prev_net ?? 0} positivo />
            <Card className="p-3">
              <p className="text-xs text-zinc-500">Variacao do resultado</p>
              <p className={`mt-1 text-xl font-semibold ${(cmp?.delta_pct ?? 0) >= 0 ? "text-emerald-600" : "text-red-600"}`}>
                {(cmp?.delta_pct ?? 0) >= 0 ? "+" : ""}{(cmp?.delta_pct ?? 0).toFixed(1)}%
              </p>
              <p className="mt-1 text-xs text-zinc-400">sobre {currency(cmp?.prev_net ?? 0)}</p>
            </Card>
          </div>
        </>
      ) : null}
    </div>
  );
}

function Comparativo({ label, atual, anterior, positivo }: { label: string; atual: number; anterior: number; positivo?: boolean }) {
  const delta = atual - anterior;
  const bom = positivo ? delta >= 0 : delta <= 0;
  return (
    <Card className="p-3">
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="mt-1 text-xl font-semibold">{currency(atual)}</p>
      <p className={`text-xs ${bom ? "text-emerald-600" : "text-red-600"}`}>
        {delta >= 0 ? "+" : ""}{currency(delta)} vs. anterior ({currency(anterior)})
      </p>
    </Card>
  );
}
