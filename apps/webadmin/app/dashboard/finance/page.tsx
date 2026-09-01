"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, Send } from "lucide-react";
import {
  listCategories,
  listTransactions,
  createTransaction,
  getBalance,
  getReceiptHTML,
  sendReceipt,
  type Category,
  type Transaction,
  type Balance,
} from "@/lib/api";

const fmt = (n: number) => n.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });

export default function FinancePage() {
  const router = useRouter();
  const [token, setToken] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [balance, setBalance] = useState<Balance | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<string | null>(null);

  const [form, setForm] = useState({ type: "income", amount: "", category_id: "", payment_method: "pix", description: "" });

  useEffect(() => {
    const t = localStorage.getItem("chosen_token");
    if (!t) {
      router.replace("/");
      return;
    }
    setToken(t);
    refresh(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [router]);

  async function refresh(t: string) {
    const [cat, tx, bal] = await Promise.all([
      listCategories(t),
      listTransactions(t),
      getBalance(t),
    ]);
    setCategories(cat.categories);
    setTxns(tx.transactions);
    setBalance(bal);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!token) return;
    try {
      const res = await createTransaction(token, {
        type: form.type,
        amount: Number(form.amount),
        category_id: form.category_id || undefined,
        payment_method: form.payment_method,
        description: form.description || undefined,
      });
      setReceipt(`${res.receipt_ref} — token ${res.receipt_token}`);
      setForm({ ...form, amount: "", description: "" });
      await refresh(token);
      setTimeout(() => setReceipt(null), 5000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro");
    }
  }

  async function viewReceipt(t: Transaction) {
    if (!token || !t.receipt_id) return;
    const html = await getReceiptHTML(token, t.receipt_id);
    const w = window.open("", "_blank", "width=520,height=720");
    if (w) {
      w.document.write(html);
      w.document.close();
    }
  }

  async function sendReceiptFor(t: Transaction, channel: "email" | "whatsapp") {
    if (!token || !t.receipt_id) return;
    const recipient = window.prompt(channel === "email" ? "E-mail do destinatário:" : "WhatsApp (com DDD):");
    if (!recipient) return;
    try {
      const d = await sendReceipt(token, t.receipt_id, channel, recipient);
      setReceipt(`Enviado por ${channel} para ${d.recipient} (${d.status}).`);
      setTimeout(() => setReceipt(null), 4000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro");
    }
  }

  const incomeTypes = categories.filter((c) => c.type === "income");
  const expenseTypes = categories.filter((c) => c.type === "expense");
  const list = form.type === "income" ? incomeTypes : expenseTypes;

  return (
    <div className="mx-auto max-w-5xl">
      <h2 className="mb-6 text-2xl font-semibold">Financeiro</h2>
      {error && <p className="mb-4 text-sm text-red-600">{error}</p>}
      {receipt && <p className="mb-4 text-sm text-emerald-600">Recibo gerado: {receipt}</p>}

      <div className="mb-6 grid gap-4 md:grid-cols-3">
        <div className="card"><span className="text-sm text-zinc-500">Entradas</span><p className="mt-2 text-2xl font-semibold text-emerald-600">{balance ? fmt(balance.income) : "..."}</p></div>
        <div className="card"><span className="text-sm text-zinc-500">Saídas</span><p className="mt-2 text-2xl font-semibold text-red-600">{balance ? fmt(balance.expense) : "..."}</p></div>
        <div className="card"><span className="text-sm text-zinc-500">Saldo</span><p className="mt-2 text-2xl font-semibold text-violet-700">{balance ? fmt(balance.net) : "..."}</p></div>
      </div>

      <div className="card mb-6">
        <h3 className="mb-4 text-sm font-medium text-zinc-600">Novo lançamento</h3>
        <form onSubmit={onSubmit} className="grid gap-3 md:grid-cols-5">
          <select className="input" value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>
            <option value="income">Dízimo / Oferta</option>
            <option value="expense">Despesa</option>
          </select>
          <input className="input" type="number" min="0.01" step="0.01" placeholder="Valor" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} required />
          <select className="input" value={form.category_id} onChange={(e) => setForm({ ...form, category_id: e.target.value })} required>
            <option value="">Categoria...</option>
            {list.map((c) => (<option key={c.id} value={c.id}>{c.name}</option>))}
          </select>
          <select className="input" value={form.payment_method} onChange={(e) => setForm({ ...form, payment_method: e.target.value })}>
            <option value="pix">PIX</option><option value="card">Cartão</option><option value="boleto">Boleto</option><option value="cash">Espécie</option>
          </select>
          <button className="btn-base btn-primary" type="submit"><Plus className="h-4 w-4" /> Lançar</button>
          <input className="input md:col-span-5" placeholder="Descrição" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        </form>
      </div>

      <div className="card overflow-hidden p-0">
        <h3 className="border-b border-zinc-200 px-4 py-3 text-sm font-medium text-zinc-600">Lançamentos recentes</h3>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-zinc-200 text-left text-zinc-500">
              <th className="px-4 py-3 font-medium">Data</th>
              <th className="px-4 py-3 font-medium">Categoria</th>
              <th className="px-4 py-3 font-medium">Descrição</th>
              <th className="px-4 py-3 font-medium">Recibo</th>
              <th className="px-4 py-3 font-medium text-right">Valor</th>
            </tr>
          </thead>
          <tbody>
            {txns.map((t) => (
              <tr key={t.id} className="border-b border-zinc-100 last:border-0">
                <td className="px-4 py-3 text-zinc-500">{new Date(t.occurred_at).toLocaleDateString("pt-BR")}</td>
                <td className="px-4 py-3">{t.category_name ?? "—"}</td>
                <td className="px-4 py-3">{t.description ?? "—"}</td>
                <td className="px-4 py-3">
                  {t.receipt_id ? (
                    <span className="flex items-center gap-1">
                      <button onClick={() => viewReceipt(t)} className="btn-base btn-ghost text-violet-700 text-xs">Ver</button>
                      <button onClick={() => sendReceiptFor(t, "whatsapp")} title="Enviar WhatsApp" className="btn-base btn-ghost text-xs"><Send className="h-3 w-3" />Zap</button>
                      <button onClick={() => sendReceiptFor(t, "email")} title="Enviar e-mail" className="btn-base btn-ghost text-xs"><Send className="h-3 w-3" />Mail</button>
                    </span>
                  ) : <span className="text-zinc-400">—</span>}
                </td>
                <td className={`px-4 py-3 text-right font-medium ${t.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                  {t.type === "income" ? "+" : "-"}{fmt(t.amount)}
                </td>
              </tr>
            ))}
            {txns.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-zinc-400">Nenhum lançamento.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
