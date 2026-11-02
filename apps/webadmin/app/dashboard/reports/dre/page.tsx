"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getDRE, type DRE } from "@/lib/api";
import { currency } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";
import { ReportToolbar } from "@/components/reports/report-toolbar";
import { ReportSection } from "@/components/reports/report-section";
import { KpiCard } from "@/components/reports/kpi-card";
import { RankedBar } from "@/components/reports/ranked-bar";
import { WaterfallChart } from "@/components/reports/waterfall-chart";
import { COLORS } from "@/components/reports/chart-theme";

export default function DreReportPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [dre, setDre] = useState<DRE | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      setDre(await getDRE(f, t));
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
  const delta = dre?.comparison?.delta_pct ?? 0;
  const income = (dre?.lines ?? []).filter((l) => l.type === "income").map((l) => ({ name: l.category, value: l.total }));
  const expense = (dre?.lines ?? []).filter((l) => l.type === "expense").map((l) => ({ name: l.category, value: l.total }));
  const margin = dre && dre.income > 0 ? (dre.net / dre.income) * 100 : 0;

  return (
    <div className="page">
      <PageHeader
        title="DRE - Demonstrativo do Resultado"
        description="Resultado por categoria e comparativo de periodo"
        actions={
          <ExportButtons
            path="/api/v1/reports/dre/export"
            params={params}
            filenameBase={`dre-${from || "inicio"}-${to || "atual"}`}
          />
        }
      />

      <ReportToolbar from={from} to={to} onFrom={setFrom} onTo={setTo} onChange={load} />

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : dre ? (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas" value={currency(dre.income)} icon={ArrowUpRight} tone="green" />
            <KpiCard label="Saidas" value={currency(dre.expense)} icon={ArrowDownRight} tone="red" />
            <KpiCard label="Resultado" value={currency(dre.net)} icon={Wallet} tone="sky" delta={delta} deltaLabel="vs. periodo anterior" />
            <KpiCard label="Margem de sobra" value={`${margin.toFixed(1)}%`} tone="indigo" hint={`anterior: ${currency(dre.comparison?.prev_net ?? 0)}`} />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection title="Receitas por conta" subtitle="Ordenadas da maior para a menor" insight={income[0] ? `${income[0].name} lidera as entradas.` : undefined}>
              <RankedBar data={income} total={dre.income} color={COLORS.income} emptyLabel="Sem entradas no periodo." />
            </ReportSection>
            <ReportSection title="Despesas por conta" subtitle="Ordenadas da maior para a menor" insight={expense[0] ? `${expense[0].name} lidera as saidas.` : undefined}>
              <RankedBar data={expense} total={dre.expense} color={COLORS.expense} emptyLabel="Sem saidas no periodo." />
            </ReportSection>
          </div>

          <ReportSection title="Resultado do periodo" subtitle="Entradas, saidas e o que sobra" className="mb-4">
            <WaterfallChart
              items={[
                { name: "Entradas", value: dre.income, type: "positive" },
                { name: "Saidas", value: -dre.expense, type: "negative" },
                { name: "Resultado", value: dre.net, type: "total" },
              ]}
              height={200}
            />
          </ReportSection>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">DRE por categoria</h3>
            <Table>
              <THead><TRow><TH>Conta contabil</TH><TH>Tipo</TH><TH className="text-right">Total</TH></TRow></THead>
              <TBody>
                {dre.lines.map((l) => (
                  <TRow key={`${l.type}-${l.category_id}`}>
                    <TD className="font-medium">
                      {l.code && <span className="mr-2 text-xs font-normal tabular-nums text-zinc-400">{l.code}</span>}
                      {l.category}
                    </TD>
                    <TD><Badge tone={l.type === "income" ? "green" : "red"}>{l.type === "income" ? "Entrada" : "Saida"}</Badge></TD>
                    <TD className={`text-right font-semibold ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(l.total)}</TD>
                  </TRow>
                ))}
                {dre.lines.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Sem lancamentos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      ) : null}
    </div>
  );
}
