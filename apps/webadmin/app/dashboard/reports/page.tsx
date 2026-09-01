"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getDRE, getMonthlyBalance, type DRE, type MonthlyPoint } from "@/lib/api";

const fmt = (n: number) => n.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
const monthLabel = (m: string) => {
  const [y, mo] = m.split("-");
  const nome = ["jan", "fev", "mar", "abr", "mai", "jun", "jul", "ago", "set", "out", "nov", "dez"];
  return `${nome[Number(mo) - 1]}/${y}`;
};

export default function ReportsPage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [series, setSeries] = useState<MonthlyPoint[]>([]);
  const [dre, setDre] = useState<DRE | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async (t: string, f: string, tt: string) => {
    const [s, d] = await Promise.all([getMonthlyBalance(t, f, tt), getDRE(t, f, tt)]);
    setSeries(s.series);
    setDre(d);
  }, []);

  useEffect(() => {
    const t = localStorage.getItem("chosen_token");
    if (!t) { router.replace("/"); return; }
    setToken(t);
    load(t, "", "").catch((e) => setError(e instanceof Error ? e.message : "Erro"));
  }, [router, load]);

  const max = Math.max(1, ...series.map((p) => Math.max(p.income, p.expense)));
  const delta = dre?.comparison?.delta_pct ?? 0;

  return (
    <div className="mx-auto max-w-5xl">
      <h2 className="mb-6 text-2xl font-semibold">Relatórios</h2>
      {error && <p className="mb-4 text-sm text-red-600">{error}</p>}

      <div className="card mb-6 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><input type="date" className="input" value={from} onChange={(e) => setFrom(e.target.value)} /></div>
        <div><label className="label">Até</label><input type="date" className="input" value={to} onChange={(e) => setTo(e.target.value)} /></div>
        <button className="btn-base btn-primary" onClick={() => token && load(token, from, to)}>Filtrar</button>
        <button className="btn-base btn-ghost" onClick={() => { setFrom(""); setTo(""); token && load(token, "", ""); }}>Mês atual</button>
      </div>

      {dre && (
        <div className="mb-6 grid gap-4 md:grid-cols-4">
          <div className="card"><span className="text-sm text-zinc-500">Entradas</span><p className="mt-2 text-xl font-semibold text-emerald-600">{fmt(dre.income)}</p></div>
          <div className="card"><span className="text-sm text-zinc-500">Saídas</span><p className="mt-2 text-xl font-semibold text-red-600">{fmt(dre.expense)}</p></div>
          <div className="card"><span className="text-sm text-zinc-500">Resultado</span><p className={`mt-2 text-xl font-semibold ${dre.net >= 0 ? "text-violet-700" : "text-red-600"}`}>{fmt(dre.net)}</p></div>
          <div className="card">
            <span className="text-sm text-zinc-500">vs. período anterior</span>
            <p className={`mt-2 text-xl font-semibold ${delta >= 0 ? "text-emerald-600" : "text-red-600"}`}>{delta >= 0 ? "+" : ""}{delta.toFixed(1)}%</p>
          </div>
        </div>
      )}

      <div className="card mb-6">
        <h3 className="mb-4 text-sm font-medium text-zinc-600">Balanço por mês</h3>
        <div className="space-y-3">
          {series.map((p) => (
            <div key={p.month}>
              <div className="mb-1 flex justify-between text-xs text-zinc-500">
                <span>{monthLabel(p.month)}</span>
                <span>Resultado: <strong className="text-zinc-700">{fmt(p.net)}</strong></span>
              </div>
              <div className="flex gap-1">
                <div className="h-3 rounded-l bg-emerald-500" style={{ width: `${(p.income / max) * 100}%` }} title={`Entradas ${fmt(p.income)}`} />
                <div className="h-3 rounded-r bg-red-400" style={{ width: `${(p.expense / max) * 100}%` }} title={`Saídas ${fmt(p.expense)}`} />
              </div>
            </div>
          ))}
          {series.length === 0 && <p className="text-sm text-zinc-400">Sem dados no período.</p>}
        </div>
      </div>

      {dre && (
        <div className="card overflow-hidden p-0">
          <h3 className="border-b border-zinc-200 px-4 py-3 text-sm font-medium text-zinc-600">DRE por categoria</h3>
          <table className="w-full text-sm">
            <thead><tr className="border-b border-zinc-200 text-left text-zinc-500">
              <th className="px-4 py-3 font-medium">Categoria</th><th className="px-4 py-3 font-medium">Tipo</th>
              <th className="px-4 py-3 font-medium text-right">Total</th></tr></thead>
            <tbody>
              {dre.lines.map((l) => (
                <tr key={`${l.type}-${l.category_id}`} className="border-b border-zinc-100 last:border-0">
                  <td className="px-4 py-3">{l.category}</td>
                  <td className="px-4 py-3"><span className={`rounded-full px-2 py-0.5 text-xs ${l.type === "income" ? "bg-emerald-50 text-emerald-700" : "bg-red-50 text-red-700"}`}>{l.type === "income" ? "Entrada" : "Saída"}</span></td>
                  <td className={`px-4 py-3 text-right font-medium ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{fmt(l.total)}</td>
                </tr>
              ))}
              {dre.lines.length === 0 && <tr><td colSpan={3} className="px-4 py-8 text-center text-zinc-400">Sem lançamentos no período.</td></tr>}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
