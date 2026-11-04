"use client";

import { useCallback, useEffect, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Select } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getMonthlyBalance, listAccounts, type BankAccount, type MonthlyPoint } from "@/lib/api";
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
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [accountId, setAccountId] = useState("");
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [openingBalance, setOpeningBalance] = useState(0);
  const [closingBalance, setClosingBalance] = useState(0);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string, acc: string) => {
    setLoading(true);
    try {
      const s = await getMonthlyBalance(f, t, acc);
      setSeries(s.series);
      setOpeningBalance(s.opening_balance);
      setClosingBalance(s.closing_balance);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    listAccounts()
      .then((r) => setAccounts(r.accounts))
      .catch(() => setAccounts([]));
  }, []);

  useEffect(() => {
    load("", "", "");
  }, [load]);

  const monthly = series.map((p) => ({
    name: monthLabel(p.month),
    "Saldo real": p.balance,
    Resultado: p.net,
    Entradas: p.income,
    Saidas: p.expense,
  }));
  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;
  if (accountId) params.account_id = accountId;
  const totalIncome = series.reduce((a, p) => a + p.income, 0);
  const totalExpense = series.reduce((a, p) => a + p.expense, 0);
  const totalNet = totalIncome - totalExpense;
  const last = series[series.length - 1];
  const margin = totalIncome > 0 ? (totalNet / totalIncome) * 100 : 0;
  const transferAdjust = closingBalance - (openingBalance + totalNet);

  return (
    <div className="page">
      <PageHeader
        title="Balancete mensal"
        description="Entradas, saidas, resultado e saldo real por mes"
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
        onChange={(f, t) => load(f, t, accountId)}
        extra={
          <div>
            <label className="label text-xs">Conta bancaria</label>
            <Select
              className="h-8 min-w-[180px] text-sm"
              value={accountId}
              onChange={(e) => {
                const v = e.target.value;
                setAccountId(v);
                load(from, to, v);
              }}
            >
              <option value="">Todas as contas</option>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </Select>
          </div>
        }
        actions={<span className="text-xs text-zinc-400">{series.length} mes(es)</span>}
      />

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <>
          <div className="mb-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <KpiCard label="Entradas no periodo" value={currency(totalIncome)} icon={ArrowUpRight} tone="green" spark={series.map((p) => p.income)} />
            <KpiCard label="Saidas no periodo" value={currency(totalExpense)} icon={ArrowDownRight} tone="red" spark={series.map((p) => p.expense)} />
            <KpiCard
              label="Resultado do periodo"
              value={currency(totalNet)}
              tone="sky"
              hint={totalIncome > 0 ? `margem de sobra de ${margin.toFixed(1)}%` : undefined}
              spark={series.map((p) => p.net)}
            />
            <KpiCard
              label="Saldo real final"
              value={currency(closingBalance)}
              icon={Wallet}
              tone="indigo"
              delta={openingBalance !== 0 ? ((closingBalance - openingBalance) / Math.abs(openingBalance)) * 100 : undefined}
              deltaLabel="no periodo"
              hint="saldo em conta ao fim do periodo"
              spark={series.map((p) => p.balance)}
            />
          </div>

          <div className="mb-4 grid gap-4 lg:grid-cols-2">
            <ReportSection
              title="Evolucao do saldo"
              subtitle="Saldo real acumulado e resultado mes a mes"
              insight={last ? `Saldo real ao fim de ${monthLabel(last.month)}: ${currency(last.balance)}.` : undefined}
            >
              <TrendChart
                data={monthly}
                series={[
                  { key: "Saldo real", label: "Saldo real", color: COLORS.sky, type: "area" },
                  { key: "Resultado", label: "Resultado", color: COLORS.net, type: "line" },
                ]}
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
            subtitle="Do saldo inicial ao saldo real final"
            className="mb-4"
          >
            <WaterfallChart
              items={[
                { name: "Saldo inicial", value: openingBalance, type: "total" },
                { name: "Entradas", value: totalIncome, type: "positive" },
                { name: "Saidas", value: -totalExpense, type: "negative" },
                ...(Math.abs(transferAdjust) >= 0.01
                  ? [{ name: "Transferencias", value: transferAdjust, type: transferAdjust >= 0 ? "positive" as const : "negative" as const }]
                  : []),
                { name: "Saldo final", value: closingBalance, type: "total" },
              ]}
              height={200}
            />
          </ReportSection>

          <Card className="overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">Serie mensal</h3>
            <Table>
              <THead><TRow><TH>Mes</TH><TH className="text-right">Entradas</TH><TH className="text-right">Saidas</TH><TH className="text-right">Resultado</TH><TH className="text-right">Saldo real</TH></TRow></THead>
              <TBody>
                {series.map((p) => (
                  <TRow key={p.month}>
                    <TD className="font-medium">{monthLabel(p.month)}</TD>
                    <TD className="text-right text-emerald-600">{currency(p.income)}</TD>
                    <TD className="text-right text-red-600">{currency(p.expense)}</TD>
                    <TD className={`text-right font-semibold ${p.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(p.net)}</TD>
                    <TD className={`text-right font-semibold ${p.balance >= 0 ? "text-indigo-700 dark:text-indigo-400" : "text-red-600"}`}>{currency(p.balance)}</TD>
                  </TRow>
                ))}
                {series.length === 0 && <TRow><TD colSpan={5} className="py-8 text-center text-zinc-400">Sem lancamentos no periodo.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      )}
    </div>
  );
}
