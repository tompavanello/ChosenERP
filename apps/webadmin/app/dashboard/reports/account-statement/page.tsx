"use client";

import { Fragment, useCallback, useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Eye, Layers, Paperclip, Pencil, Plus, Trash2 } from "lucide-react";
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
import { currency, datePt, todayISO } from "@/lib/format";
import { ACCOUNT_TYPES } from "@/lib/constants";
import { ExportButtons } from "@/components/reports/export-buttons";
import { TransactionForm, type TransactionFormState } from "@/components/finance/transaction-form";
import { GroupDetailDrawer } from "@/components/finance/transaction-group";
import { GroupMembersInline } from "@/components/finance/group-members-inline";
import { AttachmentDrawer, type AttachmentOwner } from "@/components/finance/attachments";

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
  const [view, setView] = useState<"bank" | "detail">("detail");
  const [st, setSt] = useState<AccountStatement | null>(null);
  const [loading, setLoading] = useState(false);
  const [groupDetailId, setGroupDetailId] = useState<string | undefined>(undefined);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [attachOwner, setAttachOwner] = useState<AttachmentOwner | null>(null);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Transaction | null>(null);
  const [yieldMode, setYieldMode] = useState(false);

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
      setSt(await getAccountStatement({ account_id: accountId, from, to, order, view }));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar extrato", "error");
      setSt(null);
    } finally {
      setLoading(false);
    }
  }, [accountId, from, to, order, view, toast]);

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
    setYieldMode(false);
    setFormOpen(true);
  }

  function openYield() {
    setEditing(null);
    setYieldMode(true);
    setFormOpen(true);
  }

  function openEdit(l: AccountStatementLine) {
    setEditing(l);
    setYieldMode(false);
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
    : yieldMode && account
      ? {
          type: "income",
          account_id: account.id,
          occurred_at: todayISO(),
          description: "Rendimento de aplicacao",
        }
      : undefined;

  const lineDay = (l: AccountStatementLine) => l.occurred_at.slice(0, 10);

  function toggleGroupExpanded(id: string) {
    setExpandedGroups((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }  const linesByDay = useMemo(() => {
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
        title={account ? `Extrato de ${(ACCOUNT_TYPES[account.account_type] ?? "conta").toLowerCase()}` : "Extrato de conta"}
        description={`${account?.name ?? ""} - movimento da conta, saldo anterior e saldo apos cada lancamento`}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {canWrite && account && (
              <Button
                variant="outline"
                onClick={() => {
                  const kind = account.account_type === "investment" ? "redeem" : "deposit";
                  window.location.href = `/dashboard/finance/account-transfers?kind=${kind}&from=${account.id}`;
                }}
              >
                {account.account_type === "investment" ? "Resgatar" : "Aplicar"}
              </Button>
            )}
            {canWrite && account?.account_type === "investment" && (
              <Button variant="outline" onClick={openYield}>
                Linha de rendimento
              </Button>
            )}
            {canWrite && account && (
              <Button onClick={openNew}>
                <Plus className="h-4 w-4" /> Novo lancamento
              </Button>
            )}
            <ExportButtons
              path="/api/v1/reports/account-statement/export"
              params={{ account_id: accountId, from, to, order, view }}
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
                {a.name} - {ACCOUNT_TYPES[a.account_type] ?? a.account_type}{a.is_active ? "" : " (inativa)"}
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
        <div>
          <label className="label">Visao</label>
          <div className="flex gap-1">
            <Button
              variant={view === "detail" ? "primary" : "outline"}
              size="sm"
              onClick={() => setView("detail")}
              title="Um lancamento por linha (data de cada contribuicao)"
            >
              Analitica
            </Button>
            <Button
              variant={view === "bank" ? "primary" : "outline"}
              size="sm"
              onClick={() => setView("bank")}
              title="Uma linha por movimentacao bancaria (data do deposito)"
            >
              Banco
            </Button>
          </div>
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
                  <TH className="w-20 text-center">Anexos</TH>
                  <TH className="text-right">Valor</TH>
                  <TH className="text-right">Saldo apos</TH>
                  {canWrite && <TH className="w-24 text-right">Acoes</TH>}
                </TRow>
              </THead>
              <TBody>
                {st.days.length === 0 && (
                  <TRow>
                    <TD colSpan={canWrite ? 9 : 8} className="py-8 text-center text-zinc-400">
                      Nenhum lancamento no periodo.
                    </TD>
                  </TRow>
                )}
                {st.days.map((day) => {
                  const dayLines = linesByDay.get(day.date) ?? [];
                  return (
                    <Fragment key={day.date}>
                      {dayLines.map((l) => {
                        const isGroupLine = !!l.is_group;
                        return (
                          <Fragment key={l.id}>
                          <TRow className={isGroupLine ? "bg-sky-50/50 dark:bg-sky-950/20" : undefined}>
                            <TD className="whitespace-nowrap text-zinc-500">{datePt(lineDay(l))}</TD>
                            <TD className="text-xs text-zinc-400">{l.entry_seq ?? ""}</TD>
                            <TD className="font-medium">
                              {isGroupLine ? (
                                <span className="inline-flex flex-wrap items-center gap-1.5 text-sky-800 dark:text-sky-300">
                                  <button
                                    type="button"
                                    onClick={() => toggleGroupExpanded(l.id)}
                                    title={expandedGroups.has(l.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
                                    aria-label={expandedGroups.has(l.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
                                    className="text-sky-500 hover:text-sky-700"
                                  >
                                    {expandedGroups.has(l.id) ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                                  </button>
                                  <Layers className="h-3.5 w-3.5" />
                                  {l.description || "Agrupamento de lancamentos"}
                                  <Badge tone="indigo" className="text-[10px]">
                                    {l.group_member_count ?? 0} itens
                                  </Badge>
                                  {l.group_status === "conciliado" && (
                                    <Badge tone="green" className="text-[10px]">Conciliado</Badge>
                                  )}
                                </span>
                              ) : (
                                <>
                                  {l.description || l.donor_name || l.category_name || "-"}
                                  {l.group_id && (
                                    <button
                                      type="button"
                                      onClick={() => setGroupDetailId(l.group_id)}
                                      className="ml-2 inline-flex"
                                      title="Ver agrupamento e anexos"
                                    >
                                      <Badge tone="indigo" className="text-[10px]">Agrupado</Badge>
                                    </button>
                                  )}
                                  {l.donor_name && l.description ? (
                                    <span className="block text-xs text-zinc-400">{l.donor_name}</span>
                                  ) : null}
                                </>
                              )}
                            </TD>
                            <TD className="text-zinc-500">
                              {isGroupLine ? <span className="text-xs">Linha bancaria</span> : (l.category_name ?? "-")}
                            </TD>
                            <TD>
                              <Badge tone={l.type === "income" ? "green" : "red"}>
                                {TYPE_LABEL[l.type] ?? l.type}
                              </Badge>
                            </TD>
                            <TD className="text-center">
                              {(l.attachment_count ?? 0) > 0 || isGroupLine ? (
                                <button
                                  type="button"
                                  onClick={() => setAttachOwner(isGroupLine ? { groupId: l.group_id ?? l.id } : { transactionId: l.id })}
                                  className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
                                  title={(l.attachment_count ?? 0) > 0 ? `Ver ${l.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
                                  aria-label={(l.attachment_count ?? 0) > 0 ? `Ver ${l.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
                                >
                                  <Paperclip className="h-3.5 w-3.5" />
                                  {(l.attachment_count ?? 0) > 0 && <span className="text-[10px] font-medium">{l.attachment_count}</span>}
                                </button>
                              ) : (
                                <span className="text-zinc-300">-</span>
                              )}
                            </TD>
                            <TD className={`text-right font-semibold tabular-nums ${l.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                              {l.type === "income" ? "+" : "-"} {currency(l.amount)}
                            </TD>
                            <TD className="text-right font-medium tabular-nums">{currency(l.balance_after)}</TD>
                            {canWrite && (
                              <TD className="text-right">
                                {isGroupLine ? (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => setGroupDetailId(l.group_id ?? l.id)}
                                    title="Ver lancamentos do grupo"
                                  >
                                    <Eye className="h-3.5 w-3.5" /> Grupo
                                  </Button>
                                ) : l.group_id ? (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => setGroupDetailId(l.group_id)}
                                    title="Ver grupo"
                                  >
                                    <Layers className="h-3.5 w-3.5" />
                                  </Button>
                                ) : (
                                  <div className="flex justify-end gap-1">
                                    <Button variant="ghost" size="sm" onClick={() => openEdit(l)} title="Editar">
                                      <Pencil className="h-3.5 w-3.5" />
                                    </Button>
                                    <Button variant="ghost" size="sm" onClick={() => remove(l)} title="Excluir">
                                      <Trash2 className="h-3.5 w-3.5 text-red-600" />
                                    </Button>
                                  </div>
                                )}
                              </TD>
                            )}
                          </TRow>
                          {isGroupLine && expandedGroups.has(l.id) && (
                            <TRow>
                              <TD colSpan={canWrite ? 9 : 8} className="p-3">
                                <GroupMembersInline
                                  groupId={l.group_id ?? l.id}
                                  from={from}
                                  to={to}
                                  onOpenGroup={(id) => setGroupDetailId(id)}
                                  onOpenTransaction={(m) => setAttachOwner({ transactionId: m.id })}
                                />
                              </TD>
                            </TRow>
                          )}
                          </Fragment>
                        );
                      })}
                      <TRow className="border-y border-sky-100 bg-sky-50/70 dark:border-sky-900/40 dark:bg-sky-950/20">
                        <TD colSpan={canWrite ? 9 : 8} className="py-2">
                          <div className="flex flex-wrap items-center justify-between gap-x-6 gap-y-1 text-xs">
                            <div className="font-semibold text-sky-800 dark:text-sky-300">
                              Total do dia
                              <span className="ml-2 font-normal text-zinc-500 dark:text-zinc-400">
                                {datePt(day.date)} · {day.count} lancamento(s)
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
          {view === "bank" && (
            <p className="mt-1 text-xs text-sky-700 dark:text-sky-400">
              Visao banco: cada agrupamento aparece como uma unica linha, na data do deposito. Os totais
              podem divergir dos relatorios mensais, que usam a data de cada contribuicao (visao analitica).
            </p>
          )}
        </>
      )}

      <Drawer
        open={formOpen}
        onClose={() => {
          setFormOpen(false);
          setEditing(null);
          setYieldMode(false);
        }}
        size="xl"
        title={editing ? "Editar lancamento" : "Novo lancamento"}
      >
        <TransactionForm
          open={formOpen}
          onClose={() => {
            setFormOpen(false);
          setEditing(null);
          setYieldMode(false);
          }}
          onSaved={load}
          onSubmit={editing ? submitEdit : undefined}
          initialState={initialState}
          submitLabel={editing ? "Salvar alteracoes" : "Lancar"}
          entrySeqReadOnly={!!editing}
        />
      </Drawer>

      <GroupDetailDrawer
        open={!!groupDetailId}
        groupId={groupDetailId}
        onClose={() => setGroupDetailId(undefined)}
        onChanged={load}
        canWrite={canWrite}
        canReconcile={hasPerm("finance.reconcile")}
      />

      <AttachmentDrawer
        open={!!attachOwner}
        onClose={() => setAttachOwner(null)}
        owner={attachOwner ?? {}}
        canUpload={canWrite}
        onUploaded={load}
      />
    </div>
  );
}

