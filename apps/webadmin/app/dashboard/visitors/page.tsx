"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { UserPlus } from "lucide-react";
import { listVisitors, createVisitor, updateVisitorStage, type Visitor } from "@/lib/api";

const STAGES: Record<string, string> = {
  welcome: "Boas-vindas",
  coffee_pastor: "Café com o Pastor",
  course: "Curso de princípios",
  cell: "Integrado em célula",
  converted: "Convertido",
};

export default function VisitorsPage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [visitors, setVisitors] = useState<Visitor[]>([]);
  const [form, setForm] = useState({ first_name: "", last_name: "", phone: "", source: "" });
  const [showForm, setShowForm] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async (t: string) => {
    const r = await listVisitors(t);
    setVisitors(r.visitors);
  }, []);

  useEffect(() => {
    const t = localStorage.getItem("chosen_token");
    if (!t) { router.replace("/"); return; }
    setToken(t);
    load(t).catch(() => {});
  }, [router, load]);

  async function addVisitor(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    await createVisitor(token, form);
    setForm({ first_name: "", last_name: "", phone: "", source: "" });
    setShowForm(false);
    await load(token);
    setMsg("Visitante registrado.");
    setTimeout(() => setMsg(null), 3000);
  }

  async function advance(v: Visitor) {
    if (!token) return;
    const order = ["welcome", "coffee_pastor", "course", "cell", "converted"];
    const idx = order.indexOf(v.journey_stage);
    const next = order[Math.min(idx + 1, order.length - 1)];
    await updateVisitorStage(token, v.id, next);
    await load(token);
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-semibold">Visitantes</h2>
        <button onClick={() => setShowForm((v) => !v)} className="btn-base btn-primary"><UserPlus className="h-4 w-4" /> Novo Visitante</button>
      </div>
      {msg && <p className="mb-4 text-sm text-emerald-600">{msg}</p>}

      {showForm && (
        <form onSubmit={addVisitor} className="card mb-6 grid grid-cols-2 gap-3">
          <div><label className="label">Nome</label><input className="input" required value={form.first_name} onChange={(e) => setForm({ ...form, first_name: e.target.value })} /></div>
          <div><label className="label">Sobrenome</label><input className="input" required value={form.last_name} onChange={(e) => setForm({ ...form, last_name: e.target.value })} /></div>
          <div><label className="label">Telefone</label><input className="input" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></div>
          <div><label className="label">Origem</label><input className="input" placeholder="evento, indicação..." value={form.source} onChange={(e) => setForm({ ...form, source: e.target.value })} /></div>
          <button className="btn-base btn-primary" type="submit">Salvar</button>
        </form>
      )}

      <div className="card overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 text-left text-zinc-500">
              <th className="px-4 py-3 font-medium">Nome</th>
              <th className="px-4 py-3 font-medium">Trilha</th>
              <th className="px-4 py-3 font-medium">Origem</th>
              <th className="px-4 py-3 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {visitors.map((v) => (
              <tr key={v.id} className="border-b border-zinc-100 last:border-0">
                <td className="px-4 py-3 font-medium">{v.full_name}</td>
                <td className="px-4 py-3">
                  <span className="rounded-full bg-violet-50 px-2 py-0.5 text-xs text-violet-700">{STAGES[v.journey_stage] ?? v.journey_stage}</span>
                </td>
                <td className="px-4 py-3 text-zinc-500">{v.source ?? "—"}</td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => advance(v)} className="btn-base btn-ghost">Avançar trilha</button>
                </td>
              </tr>
            ))}
            {visitors.length === 0 && <tr><td colSpan={4} className="px-4 py-8 text-center text-zinc-400">Nenhum visitante.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
