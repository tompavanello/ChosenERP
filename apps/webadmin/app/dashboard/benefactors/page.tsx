"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { HeartHandshake, Plus } from "lucide-react";
import { listBenefactors, createBenefactor, type Benefactor } from "@/lib/api";

export default function BenefactorsPage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [benefactors, setBenefactors] = useState<Benefactor[]>([]);
  const [form, setForm] = useState({ name: "", email: "", phone: "", notes: "" });
  const [showForm, setShowForm] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async (t: string) => {
    const r = await listBenefactors(t);
    setBenefactors(r.benefactors);
  }, []);

  useEffect(() => {
    const t = localStorage.getItem("chosen_token");
    if (!t) { router.replace("/"); return; }
    setToken(t);
    load(t).catch(() => {});
  }, [router, load]);

  async function add(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    await createBenefactor(token, form);
    setForm({ name: "", email: "", phone: "", notes: "" });
    setShowForm(false);
    await load(token);
    setMsg("Benfeitor cadastrado.");
    setTimeout(() => setMsg(null), 3000);
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-semibold">Benfeitores</h2>
        <button onClick={() => setShowForm((v) => !v)} className="btn-base btn-primary"><Plus className="h-4 w-4" /> Novo Benfeitor</button>
      </div>
      {msg && <p className="mb-4 text-sm text-emerald-600">{msg}</p>}

      {showForm && (
        <form onSubmit={add} className="card mb-6 grid grid-cols-2 gap-3">
          <div><label className="label">Nome</label><input className="input" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} /></div>
          <div><label className="label">E-mail</label><input className="input" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></div>
          <div><label className="label">Telefone</label><input className="input" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} /></div>
          <div><label className="label">Observações</label><input className="input" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></div>
          <button className="btn-base btn-primary" type="submit">Salvar</button>
        </form>
      )}

      <div className="card overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 text-left text-zinc-500">
              <th className="px-4 py-3 font-medium">Nome</th>
              <th className="px-4 py-3 font-medium">Contato</th>
              <th className="px-4 py-3 font-medium">Observações</th>
            </tr>
          </thead>
          <tbody>
            {benefactors.map((b) => (
              <tr key={b.id} className="border-b border-zinc-100 last:border-0">
                <td className="px-4 py-3 font-medium">{b.name}</td>
                <td className="px-4 py-3 text-zinc-500">{b.email ?? "—"} {b.phone ? `· ${b.phone}` : ""}</td>
                <td className="px-4 py-3 text-zinc-500">{b.notes ?? "—"}</td>
              </tr>
            ))}
            {benefactors.length === 0 && <tr><td colSpan={3} className="px-4 py-8 text-center text-zinc-400"><HeartHandshake className="mx-auto mb-2 h-6 w-6" />Nenhum benfeitor cadastrado.</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}
