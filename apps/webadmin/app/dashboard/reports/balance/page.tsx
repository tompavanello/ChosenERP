"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getMonthlyBalance, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";
import { ReportToolbar } from "@/components/reports/report-toolbar";
import { ReportSection } from "@/components/reports/report-section";
import { KpiCard } from "@/components/reports/kpi-card";
import { TrendChart } from "@/components/reports/trend-chart";
import { WaterfallChart } from "@/components/reports/waterfall-chart";
import { COLORS, currencyCompact } from "@/components/reports/chart-theme";

export default function BalanceReportPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      const s = await getMonthlyBalance(f, t);
      setSeries(s.series);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load("", "");
  }, [load]);

  const monthly = series.map((p) => ({ name: monthLabel(p.month), Saldo: p.net, Entradas: p.income, Saidas: p.expense }));
  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;
  const totalIncome = series.reduce((a, p) => a + p.income, 0);
  const totalExpense = series.reduce((a, p) => a + p.expense, 0);
  const totalNet = totalIncome - totalExpense;
  const last = series[series.length - 1];
  const prev = series[series.length - 2];
  const deltaNet = prev && prev.net !== 0 ? ((last.net - prev.net) / Math.abs(prev.net)) * 100 : undefined;
  const margin = totalIncome > 0 ? (totalNet / totalIncome) * 100 : 0;

  return (
    <div className="page">
      <PageHeader
        title="Balancete mensal"
        description="Entradas, saidas e saldo por mes"
        actions={
          <ExportButtons
            path="/api/v1/reports/balance/export"
            params={params}
            filenameBase={`balancete-${from || "inicio"}-${to || "atual"}`}
          />
        }
      />

      <ReportToolbar
        from={from}
        to={to}
        onFrom={setFrom}
        onTo={setTo}
        onChange={load}
        actions={<span className="text-xs text-zinc-400">{series.length} mes(es)</span>}
      />

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas no periodo" value={currency(totalIncome)} icon={ArrowUpRight} tone="green" spark={series.map((p) => p.income)} />
            <KpiCard label="Saidas no periodo" value={currency(totalExpense)} icon={ArrowDownRight} tone="red" spark={series.map((p) => p.expense)} />
            <KpiCard label="Saldo do periodo" value={currency(totalNet)} icon={Wallet} tone="sky" delta={deltaNet} deltaLabel="ultimo mes vs. anterior" spark={series.map((p) => p.net)} />
            <KpiCard label="Margem de sobra" value={`${margin.toFixed(1)}%`} tone="indigo" hint="resultado / entradas do periodo" />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection title="Evolucao do saldo" subtitle="Resultado mes a mes no periodo" insight={last ? `Ultimo mes: ${currency(last.net)} de saldo.` : undefined}>
              <TrendChart
                data={monthly}
                series={[{ key: "Saldo", label: "Saldo", color: COLORS.net, type: "area" }]}
                height={240}
                format={currencyCompact}
              />
            </ReportSection>
            <ReportSection title="Entradas x saidas" subtitle="Comparacao direta por mes" insight={totalIncome > 0 ? `As saidas consomem ${((totalExpense / totalIncome) * 100).toFixed(0)}% das entradas.` : undefined}>
              <TrendChart
                data={monthly}
                series={[
                  { key: "Entradas", label: "Entradas", color: COLORS.income, type: "bar" },
                  { key: "Saidas", label: "Saidas", color: COLORS.expense, type: "bar" },
                ]}
                height={240}
                format={currencyCompact}
              />
            </ReportSection>
          </div>

          <ReportSection
            title="Como o periodo fechou?"
            subtitle="Fluxo: entradas, saidas e o resultado final"
            className="mb-4"
          >
            <WaterfallChart
              items={[
                { name: "Entradas", value: totalIncome, type: "positive" },
                { name: "Saidas", value: -totalExpense, type: "negative" },
                { name: "Resultado", value: totalNet, type: "total" },
              ]}
              height={200}
            />
          </ReportSection>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">Serie mensal</h3>
            <Table>
              <THead><TRow><TH>Mes</TH><TH className="text-right">Entradas</TH><TH className="text-right">Saidas</TH><TH className="text-right">Saldo</TH></TRow></THead>
              <TBody>
                {series.map((p) => (
                  <TRow key={p.month}>
                    <TD className="font-medium">{monthLabel(p.month)}</TD>
                    <TD className="text-right text-emerald-600">{currency(p.income)}</TD>
                    <TD className="text-right text-red-600">{currency(p.expense)}</TD>
                    <TD className={`text-right font-semibold ${p.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(p.net)}</TD>
                  </TRow>
                ))}
                {series.length === 0 && <TRow><TD colSpan={4} className="py-8 text-center text-zinc-400">Sem lancamentos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      )}
    </div>
  );
}
