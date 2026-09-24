"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ResponsiveContainer, AreaChart, Area, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getMonthlyBalance, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

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

  const monthly = series.map((p) => ({ name: monthLabel(p.month), Saldo: p.net, Entradas: p.income, Saídas: p.expense }));
  const params: Record<string, string> = {};
  if (from) params.from = from;
  if (to) params.to = to;
  const totalIncome = series.reduce((a, p) => a + p.income, 0);
  const totalExpense = series.reduce((a, p) => a + p.expense, 0);

  return (
    <div className="page">
      <PageHeader
        title="Balancete mensal"
        description="Entradas, saídas e saldo por mês"
        actions={
          <ExportButtons
            path="/api/v1/reports/balance/export"
            params={params}
            filenameBase={`balancete-${from || "inicio"}-${to || "atual"}`}
          />
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Até</label><Input type="date" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <Button onClick={() => load(from, to)}>Filtrar</Button>
        <Button variant="ghost" onClick={() => { setFrom(""); setTo(""); load("", ""); }}>Mês atual</Button>
      </Card>

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-3">
            <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(totalIncome)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saídas</p><p className="mt-1 text-2xl font-semibold text-red-600">{currency(totalExpense)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saldo</p><p className={`mt-1 text-2xl font-semibold ${totalIncome - totalExpense >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(totalIncome - totalExpense)}</p></Card>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700">Saldo por mês</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={monthly}><defs><linearGradient id="bnet" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4} /><stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} /></linearGradient></defs><CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} /><XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" /><YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} /><Tooltip formatter={(v: unknown) => currency(Number(v))} /><Area type="monotone" dataKey="Saldo" stroke="#8b5cf6" strokeWidth={2.5} fill="url(#bnet)" /></AreaChart>
                </ResponsiveContainer>
              </div>
            </Card>
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700">Entradas × Saídas</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={monthly}><CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} /><XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" /><YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} /><Tooltip formatter={(v: unknown) => currency(Number(v))} /><Legend /><Bar dataKey="Entradas" fill="#10b981" /><Bar dataKey="Saídas" fill="#ef4444" /></BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          <Card className="mt-6 overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">Série mensal</h3>
            <Table>
              <THead><TRow><TH>Mês</TH><TH className="text-right">Entradas</TH><TH className="text-right">Saídas</TH><TH className="text-right">Saldo</TH></TRow></THead>
              <TBody>
                {series.map((p) => (
                  <TRow key={p.month}>
                    <TD className="font-medium">{monthLabel(p.month)}</TD>
                    <TD className="text-right text-emerald-600">{currency(p.income)}</TD>
                    <TD className="text-right text-red-600">{currency(p.expense)}</TD>
                    <TD className={`text-right font-semibold ${p.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(p.net)}</TD>
                  </TRow>
                ))}
                {series.length === 0 && <TRow><TD colSpan={4} className="py-8 text-center text-zinc-400">Sem lançamentos no período.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      )}
    </div>
  );
}
