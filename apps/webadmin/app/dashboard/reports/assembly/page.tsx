"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Scale } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getAssemblyReport, type AssemblyReport } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";
import { ReportToolbar } from "@/components/reports/report-toolbar";
import { ReportSection } from "@/components/reports/report-section";
import { KpiCard } from "@/components/reports/kpi-card";
import { RankedBar } from "@/components/reports/ranked-bar";
import { WaterfallChart } from "@/components/reports/waterfall-chart";
import { TrendChart } from "@/components/reports/trend-chart";
import { COLORS, currencyCompact } from "@/components/reports/chart-theme";

export default function AssemblyReportPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [report, setReport] = useState<AssemblyReport | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      setReport(await getAssemblyReport(f, t));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load("", "");
  }, [load]);

  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;

  const bal = report?.balance;
  const months = report?.months ?? [];
  const income = (bal?.by_category ?? []).filter((c) => c.type === "income").map((c) => ({ name: c.category, value: c.total }));
  const expense = (bal?.by_category ?? []).filter((c) => c.type === "expense").map((c) => ({ name: c.category, value: c.total }));
  const monthly = months.map((p) => ({ name: monthLabel(p.month), Entradas: p.income, Saidas: p.expense, Resultado: p.net }));
  const margin = bal && bal.income > 0 ? (bal.net / bal.income) * 100 : 0;
  const covered = report ? `${report.from} a ${report.to}` : "";

  return (
    <div className="page">
      <PageHeader
        title="Demonstrativo para assembleia"
        description="Prestacao de contas do periodo para apresentacao em assembleia"
        actions={
          <ExportButtons
            path="/api/v1/reports/assembly/export"
            params={params}
            filenameBase={`demonstrativo-assembleia-${from || "inicio"}-${to || "atual"}`}
          />
        }
      />

      <ReportToolbar from={from} to={to} onFrom={setFrom} onTo={setTo} onChange={load} actions={<span className="text-xs text-zinc-400">{covered}</span>} />

      {loading ? (
        <div className="grid gap-4 md:grid-cols-3">{[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : !bal ? (
        <p className="text-sm text-zinc-400">Sem dados.</p>
      ) : (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas do periodo" value={currency(bal.income)} icon={ArrowUpRight} tone="green" />
            <KpiCard label="Saidas do periodo" value={currency(bal.expense)} icon={ArrowDownRight} tone="red" />
            <KpiCard label="Resultado do periodo" value={currency(bal.net)} icon={Scale} tone="sky" />
            <KpiCard label="Margem de sobra" value={`${margin.toFixed(1)}%`} tone="indigo" hint="resultado / entradas" />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection title="Entradas por conta" subtitle="Prestacao de contas das receitas" insight={income[0] ? `${income[0].name} e a maior receita.` : undefined}>
              <RankedBar data={income} total={bal.income} color={COLORS.income} emptyLabel="Sem entradas." />
            </ReportSection>
            <ReportSection title="Saidas por conta" subtitle="Prestacao de contas das despesas" insight={expense[0] ? `${expense[0].name} e a maior despesa.` : undefined}>
              <RankedBar data={expense} total={bal.expense} color={COLORS.expense} emptyLabel="Sem saidas." />
            </ReportSection>
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection title="Resumo do resultado" subtitle="Como o periodo fechou">
              <WaterfallChart
                items={[
                  { name: "Entradas", value: bal.income, type: "positive" },
                  { name: "Saidas", value: -bal.expense, type: "negative" },
                  { name: "Resultado", value: bal.net, type: "total" },
                ]}
                height={200}
              />
            </ReportSection>
            <ReportSection title="Evolucao mensal" subtitle="Entradas, saidas e resultado mes a mes">
              <TrendChart
                data={monthly}
                series={[
                  { key: "Entradas", label: "Entradas", color: COLORS.income, type: "bar" },
                  { key: "Saidas", label: "Saidas", color: COLORS.expense, type: "bar" },
                  { key: "Resultado", label: "Resultado", color: COLORS.net, type: "line" },
                ]}
                height={200}
                format={currencyCompact}
              />
            </ReportSection>
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card className="overflow-hidden p-0">
              <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">Entradas por conta</h3>
              <Table>
                <THead><TRow><TH>Conta contabil</TH><TH className="text-right">Total</TH></TRow></THead>
                <TBody>
                  {(bal.by_category ?? []).filter((c) => c.type === "income").map((c) => (
                    <TRow key={c.category_id}>
                      <TD>{c.code && <span className="mr-2 text-xs tabular-nums text-zinc-400">{c.code}</span>}{c.category}</TD>
                      <TD className="text-right text-emerald-600">{currency(c.total)}</TD>
                    </TRow>
                  ))}
                  {income.length === 0 && <TRow><TD colSpan={2} className="py-6 text-center text-zinc-400">Sem entradas.</TD></TRow>}
                </TBody>
              </Table>
            </Card>
            <Card className="overflow-hidden p-0">
              <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">Saidas por conta</h3>
              <Table>
                <THead><TRow><TH>Conta contabil</TH><TH className="text-right">Total</TH></TRow></THead>
                <TBody>
                  {(bal.by_category ?? []).filter((c) => c.type === "expense").map((c) => (
                    <TRow key={c.category_id}>
                      <TD>{c.code && <span className="mr-2 text-xs tabular-nums text-zinc-400">{c.code}</span>}{c.category}</TD>
                      <TD className="text-right text-red-600">{currency(c.total)}</TD>
                    </TRow>
                  ))}
                  {expense.length === 0 && <TRow><TD colSpan={2} className="py-6 text-center text-zinc-400">Sem saidas.</TD></TRow>}
                </TBody>
              </Table>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
