"use client";

import { Fragment, useCallback, useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Pencil, Plus, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input, Select } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listAccounts, getAccountStatement, createTransaction, updateTransaction, deleteTransaction,
  type AccountStatement, type AccountStatementLine, type BankAccount, type Transaction,
} from "@/lib/api";
import { currency, datePt } from "@/lib/format";
import { ExportButtons } from "@/components/reports/export-buttons";
import { TransactionForm, type TransactionFormState } from "@/components/finance/transaction-form";

const iso = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;

function monthRange(base = new Date()) {
  const first = new Date(base.getFullYear(), base.getMonth(), 1);
  const last = new Date(base.getFullYear(), base.getMonth() + 1, 0);
  return { from: iso(first), to: iso(last) };
}

function yearRange() {
  const y = new Date().getFullYear();
  return { from: `${y}-01-01`, to: `${y}-12-31` };
}

const ALL_TIME = { from: "1900-01-01", to: "2100-12-31" };

const TYPE_LABEL: Record<string, string> = { income: "Entrada", expense: "Saida" };

export default function AccountStatementPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("finance.write");

  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [accountId, setAccountId] = useState("");
  const [from, setFrom] = useState(monthRange().from);
  const [to, setTo] = useState(monthRange().to);
  const [order, setOrder] = useState<"asc" | "desc">("asc");
  const [st, setSt] = useState<AccountStatement | null>(null);
  const [loading, setLoading] = useState(false);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Transaction | null>(null);

  useEffect(() => {
    listAccounts()
      .then((r) => {
        setAccounts(r.accounts);
        const first = r.accounts.find((a) => a.is_active) ?? r.accounts[0];
        if (first) setAccountId((cur) => cur || first.id);
      })
      .catch(() => setAccounts([]));
  }, []);

  const load = useCallback(async () => {
    if (!accountId) {
      setSt(null);
      return;
    }
    setLoading(true);
    try {
      setSt(await getAccountStatement({ account_id: accountId, from, to, order }));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar extrato", "error");
      setSt(null);
    } finally {
      setLoading(false);
    }
  }, [accountId, from, to, order, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const account = useMemo(() => accounts.find((a) => a.id === accountId), [accounts, accountId]);
  const preset = useMemo(() => {
    const cur = monthRange();
    const prev = monthRange(new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1));
    const year = yearRange();
    if (from === cur.from && to === cur.to) return "month";
    if (from === prev.from && to === prev.to) return "prev";
    if (from === year.from && to === year.to) return "year";
    if (from === ALL_TIME.from && to === ALL_TIME.to) return "all";
    return "custom";
  }, [from, to]);

  function applyPreset(p: "month" | "prev" | "year" | "all") {
    if (p === "month") {
      const r = monthRange();
      setFrom(r.from);
      setTo(r.to);
    } else if (p === "prev") {
      const r = monthRange(new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1));
      setFrom(r.from);
      setTo(r.to);
    } else if (p === "year") {
      const r = yearRange();
      setFrom(r.from);
      setTo(r.to);
    } else {
      setFrom(ALL_TIME.from);
      setTo(ALL_TIME.to);
    }
  }

  function openNew() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(l: AccountStatementLine) {
    setEditing(l);
    setFormOpen(true);
  }

  async function submitEdit(data: Record<string, unknown>) {
    if (!editing) return;
    await updateTransaction(editing.id, data);
    toast("Lancamento corrigido.");
    setEditing(null);
    await load();
  }

  async function remove(l: AccountStatementLine) {
    if (!confirm("Excluir definitivamente este lancamento? Esta acao nao pode ser desfeita.")) return;
    try {
      await deleteTransaction(l.id);
      toast("Lancamento excluido.");
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao excluir", "error");
    }
  }

  const initialState: Partial<TransactionFormState> | undefined = editing
    ? {
        type: editing.type,
        amount: String(editing.amount),
        category_id: editing.category_id ?? "",
        account_id: editing.account_id ?? "",
        payment_method: editing.payment_method ?? "",
        description: editing.description ?? "",
        occurred_at: editing.occurred_at.slice(0, 10),
        donor_member_id: editing.donor_member_id ?? "",
        benefactor_id: editing.benefactor_id ?? "",
        donor_name: editing.donor_name ?? "",
        supplier_id: editing.supplier_id ?? "",
        is_anonymous: editing.is_anonymous,
        entry_seq: editing.entry_seq !== undefined && editing.entry_seq !== null ? String(editing.entry_seq) : "",
        due_date: editing.due_date ?? "",
        cost_center: editing.cost_center ?? "",
        paid: !!editing.paid_at,
      }
    : undefined;

  const lineDay = (l: AccountStatementLine) => l.occurred_at.slice(0, 10);
  const linesByDay = useMemo(() => {
    const m = new Map<string, AccountStatementLine[]>();
    for (const l of st?.lines ?? []) {
      const key = lineDay(l);
      const arr = m.get(key) ?? [];
      arr.push(l);
      m.set(key, arr);
    }
    return m;
  }, [st]);

  return (
    <div className="page">
      <PageHeader
        title="Extrato de conta corrente"
        description="Movimento da conta, saldo anterior e saldo apos cada lancamento"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {canWrite && account && (
              <Button onClick={openNew}>
                <Plus className="h-4 w-4" /> Novo lancamento
              </Button>
            )}
            <ExportButtons
              path="/api/v1/reports/account-statement/export"
              params={{ account_id: accountId, from, to, order }}
              filenameBase={`extrato-${account?.name ?? "conta"}-${from}-${to}`}
            />
          </div>
        }
      />

      <Card className="mb-6 flex flex-wrap items-end gap-3">
        <div className="w-full sm:w-72">
          <label className="label">Conta bancaria</label>
          <Select value={accountId} onChange={(e) => setAccountId(e.target.value)}>
            <option value="" disabled>Selecione...</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}{a.is_active ? "" : " (inativa)"}
              </option>
            ))}
          </Select>
        </div>
        <div>
          <label className="label">De</label>
          <Input type="date" className="h-8 text-sm" value={from} onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div>
          <label className="label">Ate</label>
          <Input type="date" className="h-8 text-sm" value={to} onChange={(e) => setTo(e.target.value)} />
        </div>
        <div>
          <label className="label">Ordem</label>
          <Select
            className="h-8 text-sm"
            value={order}
            onChange={(e) => setOrder(e.target.value as "asc" | "desc")}
          >
            <option value="asc">Mais antigo primeiro</option>
            <option value="desc">Mais recente primeiro</option>
          </Select>
        </div>
        <div className="flex flex-wrap gap-1">
          <Button variant={preset === "month" ? "primary" : "outline"} size="sm" onClick={() => applyPreset("month")}>
            Mes atual
          </Button>
          <Button variant={preset === "prev" ? "primary" : "outline"} size="sm" onClick={() => applyPreset("prev")}>
            Mes anterior
          </Button>
          <Button variant={preset === "year" ? "primary" : "outline"} size="sm" onClick={() => applyPreset("year")}>
            Ano atual
          </Button>
          <Button variant={preset === "all" ? "primary" : "outline"} size="sm" onClick={() => applyPreset("all")}>
            Tudo
          </Button>
        </div>
      </Card>

      {!accountId ? (
        <Card className="p-6 text-sm text-zinc-500">Selecione uma conta bancaria.</Card>
      ) : loading ? (
        <div className="grid gap-4 md:grid-cols-4">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-24" />)}
        </div>
      ) : !st ? (
        <Card className="p-6 text-sm text-zinc-500">Sem dados para o periodo.</Card>
      ) : (
        <>
          <div className="mb-6 grid gap-4 md:grid-cols-4">
            <Card>
              <p className="text-sm text-zinc-500">Saldo anterior</p>
              <p className="mt-1 text-2xl font-semibold">{currency(st.opening_balance)}</p>
              <p className="mt-1 text-xs text-zinc-400">Saldo inicial da conta: {currency(st.initial_balance)}</p>
            </Card>
            <Card>
              <p className="text-sm text-zinc-500">Entradas</p>
              <p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(st.total_income)}</p>
            </Card>
            <Card>
              <p className="text-sm text-zinc-500">Saidas</p>
              <p className="mt-1 text-2xl font-semibold text-red-600">{currency(st.total_expense)}</p>
            </Card>
            <Card>
              <p className="text-sm text-zinc-500">Saldo final</p>
              <p className={`mt-1 text-2xl font-semibold ${st.closing_balance >= 0 ? "text-sky-700" : "text-red-600"}`}>
                {currency(st.closing_balance)}
              </p>
            </Card>
          </div>

          <Card className="overflow-hidden p-0">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-100 px-4 py-3 dark:border-zinc-800">
              <div>
                <h3 className="text-sm font-semibold">{st.account_name}</h3>
                <p className="text-xs text-zinc-400">
                  Periodo de {datePt(st.from)} a {datePt(st.to)} - {st.lines.length} lancamento(s)
                </p>
              </div>
              <div className="flex items-center gap-1 text-xs text-zinc-500">
                {order === "asc" ? <ArrowUp className="h-3.5 w-3.5" /> : <ArrowDown className="h-3.5 w-3.5" />}
                {order === "asc" ? "Cronologico" : "Mais recente primeiro"}
              </div>
            </div>

            <Table>
              <THead>
                <TRow>
                  <TH>Data</TH>
                  <TH className="w-14">Seq.</TH>
                  <TH>Descricao</TH>
                  <TH>Categoria</TH>
                  <TH>Tipo</TH>
                  <TH className="text-right">Valor</TH>
                  <TH className="text-right">Saldo apos</TH>
                  {canWrite && <TH className="w-24 text-right">Acoes</TH>}
                </TRow>
              </THead>
              <TBody>
                {st.days.length === 0 && (
                  <TRow>
                    <TD colSpan={canWrite ? 8 : 7} className="py-8 text-center text-zinc-400">
                      Nenhum lancamento no periodo.
                    </TD>
                  </TRow>
                )}
                {st.days.map((day) => {
                  const dayLines = linesByDay.get(day.date) ?? [];
                  return (
                    <Fragment key={day.date}>
                      <TRow className="border-y border-sky-100 bg-sky-50/70 dark:border-sky-900/40 dark:bg-sky-950/20">
                        <TD colSpan={canWrite ? 8 : 7} className="py-2">
                          <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-1 text-xs">
                            <div className="font-semibold text-sky-800 dark:text-sky-300">
                              {datePt(day.date)}
                              <span className="ml-2 font-normal text-zinc-500 dark:text-zinc-400">
                                {day.count} lancamento(s)
                              </span>
                            </div>
                            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-zinc-600 dark:text-zinc-300">
                              <span>
                                Entradas{" "}
                                <b className="font-semibold text-emerald-600">{currency(day.income)}</b>
                              </span>
                              <span>
                                Saidas{" "}
                                <b className="font-semibold text-red-600">{currency(day.expense)}</b>
                              </span>
                              <span className="rounded bg-white/70 px-2 py-0.5 dark:bg-zinc-900/60">
                                Saldo do dia{" "}
                                <b className="font-semibold tabular-nums">{currency(day.closing_balance)}</b>
                              </span>
                            </div>
                          </div>
                        </TD>
                      </TRow>
                      {dayLines.map((l) => (
                        <TRow key={l.id}>
                          <TD className="whitespace-nowrap text-zinc-500">{datePt(lineDay(l))}</TD>
                          <TD className="text-xs text-zinc-400">{l.entry_seq ?? ""}</TD>
                          <TD className="font-medium">
                            {l.description || l.donor_name || l.category_name || "-"}
                            {l.donor_name && l.description ? (
                              <span className="block text-xs text-zinc-400">{l.donor_name}</span>
                            ) : null}
                          </TD>
                          <TD className="text-zinc-500">{l.category_name ?? "-"}</TD>
                          <TD>
                            <Badge tone={l.type === "income" ? "green" : "red"}>
                              {TYPE_LABEL[l.type] ?? l.type}
                            </Badge>
                          </TD>
                          <TD className={`text-right font-semibold tabular-nums ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                            {l.type === "income" ? "+" : "-"} {currency(l.amount)}
                          </TD>
                          <TD className="text-right font-medium tabular-nums">{currency(l.balance_after)}</TD>
                          {canWrite && (
                            <TD className="text-right">
                              <div className="flex justify-end gap-1">
                                <Button variant="ghost" size="sm" onClick={() => openEdit(l)} title="Editar">
                                  <Pencil className="h-3.5 w-3.5" />
                                </Button>
                                <Button variant="ghost" size="sm" onClick={() => remove(l)} title="Excluir">
                                  <Trash2 className="h-3.5 w-3.5 text-red-600" />
                                </Button>
                              </div>
                            </TD>
                          )}
                        </TRow>
                      ))}
                    </Fragment>
                  );
                })}
              </TBody>
            </Table>
          </Card>

          <p className="mt-4 text-xs text-zinc-400">
            O saldo inicial da conta nao possui data; ele entra no saldo anterior ao periodo. Lancamentos
            estornados nao aparecem no extrato. Saldo final = saldo anterior + entradas - saidas.
          </p>
        </>
      )}

      <Drawer
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
        }}
        size="xl"
        title={editing ? "Editar lancamento" : "Novo lancamento"}
      >
        <TransactionForm
          open={formOpen}
          onClose={() => {
            setFormOpen(false);
            setEditing(null);
          }}
          onSaved={load}
          onSubmit={editing ? submitEdit : undefined}
          initialState={initialState}
          submitLabel={editing ? "Salvar alteracoes" : "Lancar"}
          entrySeqReadOnly={!!editing}
        />
      </Drawer>
    </div>
  );
}
