"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getAssemblyReport, type AssemblyReport } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

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
  const income = bal?.by_category.filter((c) => c.type === "income") ?? [];
  const expense = bal?.by_category.filter((c) => c.type === "expense") ?? [];
  const monthly = months.map((p) => ({ name: monthLabel(p.month), Entradas: p.income, Saídas: p.expense }));

  return (
    <div className="page">
      <PageHeader
        title="Demonstrativo para assembleia"
        description="Prestação de contas do período para apresentação em assembleia"
        actions={
          <ExportButtons
            path="/api/v1/reports/assembly/export"
            params={params}
            filenameBase={`demonstrativo-assembleia-${from || "inicio"}-${to || "atual"}`}
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
        <div className="grid gap-4 md:grid-cols-3">{[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}</div>
      ) : !bal ? (
        <p className="text-sm text-zinc-400">Sem dados.</p>
      ) : (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-3">
            <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(bal.income)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Saídas</p><p className="mt-1 text-2xl font-semibold text-red-600">{currency(bal.expense)}</p></Card>
            <Card><p className="text-sm text-zinc-500">Resultado do período</p><p className={`mt-1 text-2xl font-semibold ${bal.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(bal.net)}</p></Card>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card className="overflow-hidden p-0">
              <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">Entradas por conta</h3>
              <Table>
                <THead><TRow><TH>Conta contábil</TH><TH className="text-right">Total</TH></TRow></THead>
                <TBody>
                  {income.map((c) => (
                    <TRow key={c.category_id}><TD>{c.category}</TD><TD className="text-right text-emerald-600">{currency(c.total)}</TD></TRow>
                  ))}
                  {income.length === 0 && <TRow><TD colSpan={2} className="py-6 text-center text-zinc-400">Sem entradas.</TD></TRow>}
                </TBody>
              </Table>
            </Card>
            <Card className="overflow-hidden p-0">
              <h3 className="border-b border-zinc-100 px-4 py-3 text-sm font-semibold text-zinc-700">Saídas por conta</h3>
              <Table>
                <THead><TRow><TH>Conta contábil</TH><TH className="text-right">Total</TH></TRow></THead>
                <TBody>
                  {expense.map((c) => (
                    <TRow key={c.category_id}><TD>{c.category}</TD><TD className="text-right text-red-600">{currency(c.total)}</TD></TRow>
                  ))}
                  {expense.length === 0 && <TRow><TD colSpan={2} className="py-6 text-center text-zinc-400">Sem saídas.</TD></TRow>}
                </TBody>
              </Table>
            </Card>
          </div>

          <Card className="mt-6">
            <h3 className="mb-4 text-sm font-semibold text-zinc-700">Entradas × Saídas por mês</h3>
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={monthly}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                  <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                  <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={60} />
                  <Tooltip formatter={(v: unknown) => currency(Number(v))} />
                  <Legend />
                  <Bar dataKey="Entradas" fill="#10b981" />
                  <Bar dataKey="Saídas" fill="#ef4444" />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
