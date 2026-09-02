"use client";

import { useCallback, useEffect, useState } from "react";
import { Download } from "lucide-react";
import {
  ResponsiveContainer, LineChart, Line, AreaChart, Area, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getDRE, getMonthlyBalance, type DRE, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";

export default function ReportsPage() {
  const { toast } = useToast();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [dre, setDre] = useState<DRE | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (f: string, t: string) => {
    setLoading(true);
    try {
      const [s, d] = await Promise.all([getMonthlyBalance(f, t), getDRE(f, t)]);
      setSeries(s.series);
      setDre(d);
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
  const max = Math.max(1, ...series.map((p) => Math.max(p.income, p.expense)));
  const delta = dre?.comparison?.delta_pct ?? 0;

  function exportCSV() {
    if (!dre) return;
    const rows = [["Periodo", dre.from, "a", dre.to], [], ["Tipo", "Categoria", "Total"]];
    dre.lines.forEach((l) => rows.push([l.type, l.category, String(l.total)]));
    rows.push([], ["Entradas", "", String(dre.income)], ["Saídas", "", String(dre.expense)], ["Resultado", "", String(dre.net)]);
    const csv = rows.map((r) => r.join(";")).join("\n");
    const blob = new Blob(["\uFEFF" + csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `relatorio-dre-${dre.from}-${dre.to}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader
        title="Relatórios"
        description="Balancete mensal e DRE por categoria"
        actions={<Button variant="outline" onClick={exportCSV}><Download className="h-4 w-4" /> Exportar CSV</Button>}
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Até</label><Input type="date" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <Button onClick={() => load(from, to)}>Filtrar</Button>
        <Button variant="ghost" onClick={() => { setFrom(""); setTo(""); load("", ""); }}>Mês atual</Button>
      </Card>

      {loading ? (
        <div className="grid gap-4 md:grid-cols-4">{[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : dre ? (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-4">
            <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(dre.income)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saídas</p><p className="mt-1 text-2xl font-semibold text-red-600">{currency(dre.expense)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Resultado</p><p className={`mt-1 text-2xl font-semibold ${dre.net >= 0 ? "text-violet-700" : "text-red-600"}`}>{currency(dre.net)}</p></Card>
            <Card>
              <p className="text-sm text-zinc-500">vs. período anterior</p>
              <p className={`mt-1 text-2xl font-semibold ${delta >= 0 ? "text-emerald-600" : "text-red-600"}`}>{delta >= 0 ? "+" : ""}{delta.toFixed(1)}%</p>
              <p className="text-xs text-zinc-400">anterior: {currency(dre.comparison?.prev_net ?? 0)}</p>
            </Card>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Saldo por mês</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={monthly}><defs><linearGradient id="rnet" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4} /><stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} /></linearGradient></defs><CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} /><XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" /><YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} /><Tooltip formatter={(v: unknown) => currency(Number(v))} /><Area type="monotone" dataKey="Saldo" stroke="#8b5cf6" strokeWidth={2.5} fill="url(#rnet)" /></AreaChart>
                </ResponsiveContainer>
              </div>
            </Card>
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Entradas × Saídas</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={monthly}><CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} /><XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" /><YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} /><Tooltip formatter={(v: unknown) => currency(Number(v))} /><Legend /><Bar dataKey="Entradas" fill="#10b981" /><Bar dataKey="Saídas" fill="#ef4444" /></BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          <Card className="mt-6 overflow-hidden p-0">
            <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700 dark:border-zinc-800 dark:text-zinc-200">DRE por categoria</h3>
            <Table>
              <THead><TRow><TH>Categoria</TH><TH>Tipo</TH><TH className="text-right">Total</TH></TRow></THead>
              <TBody>
                {dre.lines.map((l) => (
                  <TRow key={`${l.type}-${l.category_id}`}>
                    <TD className="font-medium">{l.category}</TD>
                    <TD><Badge tone={l.type === "income" ? "green" : "red"}>{l.type === "income" ? "Entrada" : "Saída"}</Badge></TD>
                    <TD className={`text-right font-semibold ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(l.total)}</TD>
                  </TRow>
                ))}
                {dre.lines.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Sem lançamentos no período.</TD></TRow>}
              </TBody>
            </Table>
          </Card>
        </>
      ) : null}
    </div>
  );
}
