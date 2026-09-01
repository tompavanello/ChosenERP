"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { QrCode, UserPlus, X } from "lucide-react";
import {
  listMembers,
  createMember,
  issueCard,
  getMemberTree,
  updateMember,
  addRelationship,
  type Member,
  type Relationship,
} from "@/lib/api";

const REL_KINDS = [
  ["spouse", "Cônjuge"],
  ["parent", "Pai/Mãe"],
  ["child", "Filho(a)"],
  ["disciple", "Discípulo(a)"],
  ["discipler", "Discipulador(a)"],
] as const;

export default function MembersPage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [selected, setSelected] = useState<Member | null>(null);
  const [tree, setTree] = useState<Relationship[]>([]);
  const [edit, setEdit] = useState(false);
  const [form, setForm] = useState<Record<string, string>>({});
  const [relForm, setRelForm] = useState({ relate_member_id: "", kind: "spouse" });
  const [error, setError] = useState<string | null>(null);
  const [msg, setMsg] = useState<string | null>(null);

  const load = useCallback(async (t: string) => {
    const r = await listMembers(t);
    setMembers(r.members);
  }, []);

  useEffect(() => {
    const t = localStorage.getItem("chosen_token");
    if (!t) {
      router.replace("/");
      return;
    }
    setToken(t);
    load(t).catch((e) => setError(e instanceof Error ? e.message : "Erro"));
  }, [router, load]);

  async function addMember(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    try {
      await createMember(token, { first_name: form.first_name || "", last_name: form.last_name || "", membership_status: "member" } as Partial<Member>);
      setMsg("Membro adicionado.");
      setForm({});
      await load(token);
      setTimeout(() => setMsg(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro");
    }
  }

  async function openDetail(m: Member) {
    if (!token) return;
    setSelected(m);
    setEdit(false);
    setForm({});
    setRelForm({ relate_member_id: "", kind: "spouse" });
    try {
      const t = await getMemberTree(token, m.id);
      setTree(t.relationships);
    } catch {
      setTree([]);
    }
  }

  async function saveEdit(e: React.FormEvent) {
    e.preventDefault();
    if (!token || !selected) return;
    try {
      await updateMember(token, selected.id, form);
      setSelected({ ...selected, ...(form as unknown as Member) });
      setEdit(false);
      setMsg("Perfil atualizado.");
      await load(token);
      setTimeout(() => setMsg(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro");
    }
  }

  async function doLink(e: React.FormEvent) {
    e.preventDefault();
    if (!token || !selected || !relForm.relate_member_id) return;
    try {
      await addRelationship(token, selected.id, relForm.relate_member_id, relForm.kind);
      const t = await getMemberTree(token, selected.id);
      setTree(t.relationships);
      setMsg("Vínculo adicionado.");
      setTimeout(() => setMsg(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro");
    }
  }

  async function onIssueCard(m: Member) {
    if (!token) return;
    try {
      const res = await issueCard(token, m.id);
      setMsg(`Carteirinha emitida para ${res.member} (${res.card_ref}).`);
      setTimeout(() => setMsg(null), 4000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Erro");
    }
  }

  return (
    <div className="mx-auto max-w-5xl">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-2xl font-semibold">Membros</h2>
        <button onClick={() => setForm({ ...form, open_create: form.open_create === "1" ? "0" : "1" })} className="btn-base btn-primary">
          <UserPlus className="h-4 w-4" /> Novo Membro
        </button>
      </div>

      {msg && <p className="mb-4 text-sm text-emerald-600">{msg}</p>}
      {error && <p className="mb-4 text-sm text-red-600">{error}</p>}

      {form.open_create === "1" && (
        <form onSubmit={addMember} className="card mb-6 flex flex-wrap items-end gap-3">
          <div><label className="label">Nome</label><input className="input" value={form.first_name || ""} onChange={(e) => setForm({ ...form, first_name: e.target.value })} required /></div>
          <div><label className="label">Sobrenome</label><input className="input" value={form.last_name || ""} onChange={(e) => setForm({ ...form, last_name: e.target.value })} required /></div>
          <button className="btn-base btn-primary" type="submit">Salvar</button>
        </form>
      )}

      <div className="card overflow-hidden p-0">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 text-left text-zinc-500">
              <th className="px-4 py-3 font-medium">Nome</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Cadastrado em</th>
              <th className="px-4 py-3 font-medium"></th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr key={m.id} className="border-b border-zinc-100 last:border-0">
                <td className="px-4 py-3 font-medium"><button onClick={() => openDetail(m)} className="hover:text-violet-700">{m.full_name}</button></td>
                <td className="px-4 py-3"><span className="rounded-full bg-zinc-100 px-2 py-0.5 text-xs text-zinc-600">{m.membership_status}</span></td>
                <td className="px-4 py-3 text-zinc-500">{new Date(m.created_at).toLocaleDateString("pt-BR")}</td>
                <td className="px-4 py-3 text-right">
                  <button onClick={() => onIssueCard(m)} className="btn-base btn-ghost text-violet-700"><QrCode className="h-4 w-4" /> Carteirinha</button>
                </td>
              </tr>
            ))}
            {members.length === 0 && <tr><td colSpan={4} className="px-4 py-8 text-center text-zinc-400">Nenhum membro no escopo.</td></tr>}
          </tbody>
        </table>
      </div>

      {selected && (
        <div className="fixed inset-0 z-10 flex items-start justify-center overflow-y-auto bg-black/30 p-6">
          <div className="card w-full max-w-2xl">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-lg font-semibold">{selected.full_name}</h3>
              <button onClick={() => setSelected(null)} className="text-zinc-400 hover:text-zinc-700"><X className="h-5 w-5" /></button>
            </div>

            <div className="mb-4 flex gap-2">
              <button onClick={() => setEdit((v) => !v)} className="btn-base btn-primary">Editar perfil</button>
              <button onClick={() => onIssueCard(selected)} className="btn-base btn-ghost text-violet-700"><QrCode className="h-4 w-4" /> Carteirinha</button>
            </div>

            {edit ? (
              <form onSubmit={saveEdit} className="mb-4 grid grid-cols-2 gap-3">
                {[
                  ["phone", "Telefone"], ["whatsapp", "WhatsApp"], ["profession", "Profissão"],
                  ["office", "Cargo"], ["membership_status", "Status"],
                ].map(([k, label]) => (
                  <div key={k}><label className="label">{label}</label>
                    <input className="input" defaultValue={(selected as unknown as Record<string, string>)[k] ?? ""} onChange={(e) => setForm({ ...form, [k]: e.target.value })} /></div>
                ))}
                <div className="col-span-2 flex gap-2">
                  <button className="btn-base btn-primary" type="submit">Salvar</button>
                  <button type="button" onClick={() => setEdit(false)} className="btn-base btn-ghost">Cancelar</button>
                </div>
              </form>
            ) : (
              <dl className="mb-4 grid grid-cols-2 gap-3 text-sm">
                <dt className="text-zinc-500">Status</dt><dd>{selected.membership_status}</dd>
                <dt className="text-zinc-500">Telefone</dt><dd>{selected.phone ?? "—"}</dd>
                <dt className="text-zinc-500">WhatsApp</dt><dd>{selected.whatsapp ?? "—"}</dd>
              </dl>
            )}

            <h4 className="mb-2 text-sm font-medium text-zinc-600">Vínculos (família / discipulado)</h4>
            <ul className="mb-4 space-y-1">
              {tree.map((r) => (
                <li key={r.id} className="flex justify-between rounded-md bg-zinc-50 px-3 py-2 text-sm">
                  <span className="font-medium">{r.related_name}</span>
                  <span className="text-zinc-500">{r.relation}</span>
                </li>
              ))}
              {tree.length === 0 && <li className="text-sm text-zinc-400">Nenhum vínculo registrado.</li>}
            </ul>

            <form onSubmit={doLink} className="flex flex-wrap items-end gap-2">
              <div className="flex-1">
                <label className="label">Vincular com</label>
                <select className="input" value={relForm.relate_member_id} onChange={(e) => setRelForm({ ...relForm, relate_member_id: e.target.value })}>
                  <option value="">Selecione...</option>
                  {members.filter((m) => m.id !== selected.id).map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
                </select>
              </div>
              <div>
                <label className="label">Relacionamento</label>
                <select className="input" value={relForm.kind} onChange={(e) => setRelForm({ ...relForm, kind: e.target.value })}>
                  {REL_KINDS.map(([v, l]) => (<option key={v} value={v}>{l}</option>))}
                </select>
              </div>
              <button className="btn-base btn-primary" type="submit">Vincular</button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
