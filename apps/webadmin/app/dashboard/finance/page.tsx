"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet, Plus, Eye, Send, Filter, Tags } from "lucide-react";
import {
  ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, CartesianGrid, Legend,
} from "recharts";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Drawer } from "@/components/ui/modal";
import { Tabs } from "@/components/ui/tabs";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input, Field, Select } from "@/components/ui/input";
import { Skeleton, SkeletonRows } from "@/components/ui/skeleton";
import { Pagination } from "@/components/ui/pagination";
import { EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listCategories, listTransactions, createTransaction, getBalance, getMonthlyBalance,
  getReceiptHTML, sendReceipt, createCategory, type Category, type Transaction, type Balance,
} from "@/lib/api";
import { PAYMENT_METHODS } from "@/lib/constants";
import { currency, datePt, dateTimePt, monthLabel } from "@/lib/format";

const PER_PAGE = 12;

export default function FinancePage() {
  const { hasPerm } = useAuth();
  const { toast } = useToast();
  const [tab, setTab] = useState("lançamentos");
  const [categories, setCategories] = useState<Category[]>([]);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [balance, setBalance] = useState<Balance | null>(null);
  const [series, setSeries] = useState<{ name: string; Entradas: number; Saídas: number }[]>([]);
  const [loading, setLoading] = useState(true);
  const [filterType, setFilterType] = useState("");
  const [filterCat, setFilterCat] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [detail, setDetail] = useState<Transaction | null>(null);
  const [showNew, setShowNew] = useState(false);
  const [form, setForm] = useState({ type: "income", amount: "", category_id: "", payment_method: "pix", description: "" });
  const [catForm, setCatForm] = useState({ type: "income", code: "", name: "" });
  const [showCatForm, setShowCatForm] = useState(false);

  const load = useCallback(async () => {
    const [cat, tx, bal, mb] = await Promise.all([
      listCategories(),
      listTransactions(),
      getBalance(),
      getMonthlyBalance(),
    ]);
    setCategories(cat.categories);
    setTxns(tx.transactions);
    setBalance(bal);
    setSeries(mb.series.map((p) => ({ name: monthLabel(p.month), Entradas: p.income, Saídas: p.expense })));
    setLoading(false);
  }, []);

  useEffect(() => {
    load().catch((e) => toast(e.message, "error"));
  }, [load, toast]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return txns.filter((t) => {
      return (
        (!filterType || t.type === filterType) &&
        (!filterCat || t.category_id === filterCat) &&
        (!q || t.description?.toLowerCase().includes(q))
      );
    });
  }, [txns, filterType, filterCat, query]);

  const pageItems = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);
  useEffect(() => setPage(1), [filterType, filterCat, query]);

  async function submitTxn(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createTransaction({ ...form, amount: Number(form.amount), category_id: form.category_id || undefined, description: form.description || undefined });
      toast("Lançamento registrado com recibo.");
      setShowNew(false);
      setForm({ type: "income", amount: "", category_id: "", payment_method: "pix", description: "" });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function submitCat(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createCategory(catForm);
      toast("Categoria criada.");
      setShowCatForm(false);
      setCatForm({ type: "income", code: "", name: "" });
      setCategories(await listCategories().then((r) => r.categories));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function viewReceipt(t: Transaction) {
    if (!t.receipt_id) return;
    try {
      const html = await getReceiptHTML(t.receipt_id);
      const w = window.open("", "_blank", "width=540,height=760");
      if (w) { w.document.write(html); w.document.close(); }
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function sendReceiptFor(t: Transaction, channel: "email" | "whatsapp") {
    if (!t.receipt_id) return;
    const recipient = window.prompt(channel === "email" ? "E-mail do destinatário:" : "WhatsApp (com DDD):");
    if (!recipient) return;
    try {
      await sendReceipt(t.receipt_id, channel, recipient);
      toast(`Enviado por ${channel}.`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  const incomeCats = categories.filter((c) => c.type === "income");
  const expenseCats = categories.filter((c) => c.type === "expense");

  return (
    <div className="mx-auto max-w-6xl">
      <PageHeader
        title="Financeiro"
        description="Dízimos, ofertas, despesas e plano de contas"
        actions={hasPerm("finance.write") ? (
          <Button onClick={() => setShowNew(true)}><Plus className="h-4 w-4" /> Lançamento</Button>
        ) : undefined}
      />

      <Tabs
        tabs={[
          { key: "lançamentos", label: "Lançamentos", icon: <Wallet className="h-4 w-4" /> },
          { key: "categorias", label: "Plano de Contas", icon: <Tags className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "lançamentos" && (
        <>
          {loading ? (
            <div className="grid gap-4 md:grid-cols-3">{[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}</div>
          ) : (
            <div className="mb-6 grid gap-4 md:grid-cols-3">
              <Card><p className="text-sm text-zinc-500">Entradas</p><p className="mt-1 flex items-center gap-1 text-2xl font-semibold text-emerald-600"><ArrowUpRight className="h-5 w-5" />{balance ? currency(balance.income) : "..."}</p></Card>
              <Card><p className="text-sm text-zinc-500">Saídas</p><p className="mt-1 flex items-center gap-1 text-2xl font-semibold text-red-600"><ArrowDownRight className="h-5 w-5" />{balance ? currency(balance.expense) : "..."}</p></Card>
              <Card><p className="text-sm text-zinc-500">Saldo</p><p className="mt-1 flex items-center gap-1 text-2xl font-semibold text-violet-700"><Wallet className="h-5 w-5" />{balance ? currency(balance.net) : "..."}</p></Card>
            </div>
          )}

          {series.length > 0 && (
            <Card className="mb-6">
              <h3 className="mb-4 text-sm font-semibold text-zinc-700 dark:text-zinc-200">Entradas × Saídas (mensal)</h3>
              <div className="h-64">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                    <XAxis dataKey="name" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" width={70} />
                    <Tooltip formatter={(v: unknown) => currency(Number(v))} />
                    <Legend />
                    <Bar dataKey="Entradas" fill="#10b981" radius={[4, 4, 0, 0]} />
                    <Bar dataKey="Saídas" fill="#ef4444" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </Card>
          )}

          <Card className="overflow-hidden p-0">
            <div className="flex flex-wrap items-center gap-2 border-b border-zinc-100 p-4 dark:border-zinc-800">
              <div className="relative flex-1 min-w-48">
                <Filter className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
                <Input className="pl-9" placeholder="Buscar por descrição" value={query} onChange={(e) => setQuery(e.target.value)} />
              </div>
              <Select className="w-36" value={filterType} onChange={(e) => setFilterType(e.target.value)}>
                <option value="">Todos</option><option value="income">Entradas</option><option value="expense">Saídas</option>
              </Select>
              <Select className="w-44" value={filterCat} onChange={(e) => setFilterCat(e.target.value)}>
                <option value="">Todas categorias</option>
                {categories.map((c) => (<option key={c.id} value={c.id}>{c.name}</option>))}
              </Select>
            </div>

            {loading ? (
              <SkeletonRows rows={5} />
            ) : filtered.length === 0 ? (
              <EmptyState icon={<Wallet className="h-10 w-10" />} title="Sem lançamentos" description="Ajuste os filtros ou lance uma nova movimentação." />
            ) : (
              <>
                <Table>
                  <THead><TRow><TH>Data</TH><TH>Categoria</TH><TH>Descrição</TH><TH>Recibo</TH><TH className="text-right">Valor</TH></TRow></THead>
                  <TBody>
                    {pageItems.map((t) => (
                      <TRow key={t.id}>
                        <TD className="text-zinc-500">{datePt(t.occurred_at)}</TD>
                        <TD>
                          <p className="font-medium">{t.category_name ?? "—"}</p>
                          {t.payment_method && <p className="text-xs text-zinc-400">{PAYMENT_METHODS[t.payment_method] ?? t.payment_method}</p>}
                        </TD>
                        <TD>{t.description ?? "—"}</TD>
                        <TD>
                          {t.receipt_id ? (
                            <span className="flex items-center gap-1.5">
                              <Button variant="ghost" size="sm" onClick={() => viewReceipt(t)}><Eye className="h-3.5 w-3.5" /> Ver</Button>
                              <Button variant="ghost" size="sm" onClick={() => sendReceiptFor(t, "whatsapp")}><Send className="h-3.5 w-3.5" /></Button>
                              <Button variant="ghost" size="sm" onClick={() => sendReceiptFor(t, "email")}><Send className="h-3.5 w-3.5" /></Button>
                            </span>
                          ) : <span className="text-zinc-400">—</span>}
                        </TD>
                        <TD className={`text-right font-semibold ${t.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{t.type === "income" ? "+" : "-"}{currency(t.amount)}</TD>
                      </TRow>
                    ))}
                  </TBody>
                </Table>
                <div className="border-t border-zinc-100 dark:border-zinc-800">
                  <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
                </div>
              </>
            )}
          </Card>
        </>
      )}

      {tab === "categorias" && (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-zinc-100 p-4 dark:border-zinc-800">
            <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">Plano de contas</h3>
            {hasPerm("finance.write") && <Button size="sm" onClick={() => setShowCatForm(true)}><Plus className="h-4 w-4" /> Categoria</Button>}
          </div>
          <Table>
            <THead><TRow><TH>Nome</TH><TH>Código</TH><TH>Tipo</TH></TRow></THead>
            <TBody>
              {categories.map((c) => (
                <TRow key={c.id}>
                  <TD className="font-medium">{c.name}</TD>
                  <TD className="text-zinc-500">{c.code}</TD>
                  <TD><Badge tone={c.type === "income" ? "green" : "red"}>{c.type === "income" ? "Entrada" : "Saída"}</Badge></TD>
                </TRow>
              ))}
              {categories.length === 0 && <TRow><TD colSpan={3} className="py-8 text-center text-zinc-400">Nenhuma categoria.</TD></TRow>}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={!!detail} onClose={() => setDetail(null)} title="Detalhe do lançamento">
        {detail && (
          <div className="space-y-4 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Valor</span>
              <span className={`text-2xl font-semibold ${detail.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{detail.type === "income" ? "+" : "-"}{currency(detail.amount)}</span>
            </div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Categoria</span><span>{detail.category_name ?? "—"}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Forma de pagamento</span><span>{detail.payment_method ? PAYMENT_METHODS[detail.payment_method] ?? detail.payment_method : "—"}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Data</span><span>{dateTimePt(detail.occurred_at)}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Recibo</span><span>{detail.receipt_ref ?? "—"}</span></div>
            <div className="flex items-start justify-between gap-4"><span className="text-zinc-500">Hash de integridade</span><span className="break-all text-xs text-zinc-400">{detail.hash}</span></div>
            <div className="pt-2">
              {detail.receipt_id && (
                <div className="flex gap-2">
                  <Button variant="outline" className="flex-1" onClick={() => viewReceipt(detail)}><Eye className="h-4 w-4" /> Ver recibo</Button>
                  <Button onClick={() => sendReceiptFor(detail, "whatsapp")}><Send className="h-4 w-4" /> Enviar</Button>
                </div>
              )}
            </div>
          </div>
        )}
      </Drawer>

      <Drawer open={showNew} onClose={() => setShowNew(false)} title="Novo lançamento">
        <form onSubmit={submitTxn} className="space-y-3">
          <Field label="Tipo">
            <Select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })}>
              <option value="income">Dízimo / Oferta</option><option value="expense">Despesa</option>
            </Select>
          </Field>
          <Field label="Valor *"><Input type="number" min="0.01" step="0.01" required value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></Field>
          <Field label="Categoria">
            <Select required value={form.category_id} onChange={(e) => setForm({ ...form, category_id: e.target.value })}>
              <option value="">Selecione...</option>
              {(form.type === "income" ? incomeCats : expenseCats).map((c) => (<option key={c.id} value={c.id}>{c.name}</option>))}
            </Select>
          </Field>
          <Field label="Forma de pagamento">
            <Select value={form.payment_method} onChange={(e) => setForm({ ...form, payment_method: e.target.value })}>
              {Object.entries(PAYMENT_METHODS).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
            </Select>
          </Field>
          <Field label="Descrição"><Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowNew(false)}>Cancelar</Button>
            <Button type="submit">Lançar</Button>
          </div>
        </form>
      </Drawer>

      <Drawer open={showCatForm} onClose={() => setShowCatForm(false)} title="Nova categoria">
        <form onSubmit={submitCat} className="space-y-3">
          <Field label="Tipo">
            <Select value={catForm.type} onChange={(e) => setCatForm({ ...catForm, type: e.target.value })}>
              <option value="income">Entrada</option><option value="expense">Saída</option>
            </Select>
          </Field>
          <Field label="Código *"><Input required placeholder="ex.: 1.5" value={catForm.code} onChange={(e) => setCatForm({ ...catForm, code: e.target.value })} /></Field>
          <Field label="Nome *"><Input required value={catForm.name} onChange={(e) => setCatForm({ ...catForm, name: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowCatForm(false)}>Cancelar</Button>
            <Button type="submit">Criar</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
