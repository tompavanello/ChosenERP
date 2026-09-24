"use client";

import { useCallback, useEffect, useState } from "react";
import {
  ResponsiveContainer, PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, Tooltip,
  CartesianGrid, Legend,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { getDRE, getMonthlyBalance, type DRE, type MonthlyPoint } from "@/lib/api";
import { currency, monthLabel } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";

const CORES = ["#0ea5e9", "#10b981", "#8b5cf6", "#f59e0b", "#ef4444", "#14b8a6", "#ec4899", "#6366f1", "#84cc16", "#f97316"];

function anoAtual() {
  return new Date().getFullYear();
}

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

  const incomePie = (dre?.lines ?? []).filter((l) => l.type === "income").map((l) => ({ name: l.category, value: l.total }));
  const expensePie = (dre?.lines ?? []).filter((l) => l.type === "expense").map((l) => ({ name: l.category, value: l.total }));
  const monthly = series.map((p) => ({ name: monthLabel(p.month), Entradas: p.income, Saídas: p.expense }));
  const cmp = dre?.comparison;

  return (
    <div className="page">
      <PageHeader
        title="Entradas × Saídas de Recursos"
        description="Composição das receitas/despesas e comparativo por período"
        actions={
          <ExportButtons
            path="/api/v1/reports/dre/export"
            params={{ from, to }}
            filenameBase={`entradas-saidas-${from}-${to}`}
          />
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Até</label><Input type="date" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <Button variant="ghost" onClick={() => { setFrom(`${ano}-01-01`); setTo(`${ano}-12-31`); }}>Ano atual</Button>
      </Card>

      {loading ? (
        <div className="grid gap-4 lg:grid-cols-2">{[0, 1].map((i) => <Skeleton key={i} className="h-72" />)}</div>
      ) : dre ? (
        <>
          <div className="mb-6 grid gap-6 lg:grid-cols-2">
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Entradas de Recursos</h3>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={incomePie} dataKey="value" nameKey="name" outerRadius={100} label={(e) => e.name}>
                      {incomePie.map((_, i) => <Cell key={i} fill={CORES[i % CORES.length]} />)}
                    </Pie>
                    <Tooltip formatter={(v: unknown) => currency(Number(v))} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </Card>
            <Card>
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Saídas de Recursos</h3>
              <div className="h-72">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={expensePie} dataKey="value" nameKey="name" outerRadius={100} label={(e) => e.name}>
                      {expensePie.map((_, i) => <Cell key={i} fill={CORES[(i + 3) % CORES.length]} />)}
                    </Pie>
                    <Tooltip formatter={(v: unknown) => currency(Number(v))} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
            </Card>
          </div>

          <Card className="mb-6">
            <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Entradas × Saídas por mês</h3>
            <div className="h-72">
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

          <Card>
            <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Comparativo com o período anterior</h3>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <Comparativo label="Entradas" atual={dre.income} anterior={cmp?.prev_income ?? 0} positivo />
              <Comparativo label="Saídas" atual={dre.expense} anterior={cmp?.prev_expense ?? 0} />
              <Comparativo label="Resultado" atual={dre.net} anterior={cmp?.prev_net ?? 0} positivo />
              <Card>
                <p className="text-sm text-zinc-500">Variação do resultado</p>
                <p className={`mt-1 text-2xl font-semibold ${(cmp?.delta_pct ?? 0) >= 0 ? "text-emerald-600" : "text-red-600"}`}>
                  {(cmp?.delta_pct ?? 0) >= 0 ? "+" : ""}{(cmp?.delta_pct ?? 0).toFixed(1)}%
                </p>
              </Card>
            </div>
          </Card>
        </>
      ) : null}
    </div>
  );
}

function Comparativo({ label, atual, anterior, positivo }: { label: string; atual: number; anterior: number; positivo?: boolean }) {
  const delta = atual - anterior;
  const bom = positivo ? delta >= 0 : delta <= 0;
  return (
    <Card>
      <p className="text-sm text-zinc-500">{label}</p>
      <p className="mt-1 text-xl font-semibold">{currency(atual)}</p>
      <p className={`text-xs ${bom ? "text-emerald-600" : "text-red-600"}`}>
        {delta >= 0 ? "+" : ""}{currency(delta)} vs. anterior ({currency(anterior)})
      </p>
    </Card>
  );
}
