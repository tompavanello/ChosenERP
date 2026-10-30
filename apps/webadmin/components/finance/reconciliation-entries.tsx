"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { CheckCircle2, ChevronDown, ChevronRight, ChevronUp, Circle, Layers, Lock, Paperclip, Search } from "lucide-react";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input, Field, Select } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listAccounts, listTransactionGroups, listTransactions, reconcileTransaction,
  reconcileTransactionGroup, reorderFinanceLines, listReconciliations,
  createReconciliation, conciliateReconciliation, reconcileTransactions,
  type BankAccount, type FinancialReconciliation, type Transaction, type TransactionGroup,
} from "@/lib/api";
import { currency, datePt, todayISO } from "@/lib/format";
import { GroupMembersInline } from "@/components/finance/group-members-inline";
import { AttachmentDrawer, type AttachmentOwner } from "@/components/finance/attachments";

type ReconRow = {
  id: string;
  kind: "tx" | "group";
  group_id?: string;
  branch_id: string;
  account_id?: string;
  occurred_at: string;
  created_at: string;
  entry_seq?: number;
  type: "income" | "expense";
  category_name?: string;
  description?: string;
  account_name?: string;
  amount: number;
  reconciled: boolean;
  member_count?: number;
  attachment_count?: number;
  tx?: Transaction;
  group?: TransactionGroup;
};

function reconScopeKey(row: ReconRow): string {
  return `${row.branch_id}|${row.account_id ?? ""}|${row.occurred_at.slice(0, 10)}`;
}

/**
 * Visao de lancamentos avulsos da conciliacao: permite marcar/desmarcar um
 * unico lancamento sem precisar criar um periodo. Lancamentos agrupados
 * aparecem como uma unica linha (o grupo) e sao conciliados em bloco.
 */
export function ReconciliationEntries() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canReconcile = hasPerm("finance.reconcile");
  const canWrite = hasPerm("finance.write");

  const now = new Date();
  const [from, setFrom] = useState(todayISO(new Date(now.getFullYear(), now.getMonth(), 1)));
  const [to, setTo] = useState(todayISO(now));
  const [status, setStatus] = useState("");
  const [query, setQuery] = useState("");
  const [accountId, setAccountId] = useState("");
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [groups, setGroups] = useState<TransactionGroup[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [reconciliations, setReconciliations] = useState<FinancialReconciliation[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState("");
  const [sortColumn, setSortColumn] = useState("occurred_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [attachOwner, setAttachOwner] = useState<AttachmentOwner | null>(null);

  const [showClose, setShowClose] = useState(false);
  const [closeForm, setCloseForm] = useState({ title: "Conciliacao financeira", period_start: from, period_end: to });
  const [closing, setClosing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [r, g, a, rec] = await Promise.all([
        listTransactions(),
        listTransactionGroups().catch(() => ({ groups: [] as TransactionGroup[] })),
        listAccounts().catch(() => ({ accounts: [] })),
        listReconciliations().catch(() => ({ reconciliations: [] as FinancialReconciliation[] })),
      ]);
      setTxns(Array.isArray(r.transactions) ? r.transactions : []);
      setGroups(Array.isArray(g.groups) ? g.groups : []);
      setAccounts(Array.isArray(a.accounts) ? a.accounts : []);
      setReconciliations(Array.isArray(rec.reconciliations) ? rec.reconciliations : []);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar lancamentos", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const closedRanges = useMemo(
    () => reconciliations
      .filter((r) => r.status === "conciliada")
      .map((r) => ({ start: r.period_start, end: r.period_end, title: r.title })),
    [reconciliations],
  );

  function isDateClosed(date: string) {
    const d = date.slice(0, 10);
    return closedRanges.some((r) => d >= r.start && d <= r.end);
  }

  async function toggle(row: ReconRow) {
    setBusyId(row.id);
    try {
      if (row.kind === "group" && row.group_id) {
        await reconcileTransactionGroup(row.group_id, !row.reconciled);
        toast(row.reconciled ? "Grupo desconciliado." : "Grupo conciliado e travado.");
      } else if (row.tx) {
        await reconcileTransaction(row.tx.id, !row.reconciled);
        toast(row.reconciled ? "Lancamento desconciliado." : "Lancamento conciliado e travado.");
      }
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao alterar conciliacao", "error");
    } finally {
      setBusyId("");
    }
  }

  const handleSort = (col: string) => {
    if (sortColumn === col) setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    else { setSortColumn(col); setSortDirection("asc"); }
  };

  function toggleExpanded(id: string) {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function moveRow(row: ReconRow, dir: -1 | 1) {
    if (!canWrite) return;
    const key = reconScopeKey(row);
    const lines = rows
      .filter((r) => reconScopeKey(r) === key)
      .sort((a, b) =>
        (a.entry_seq ?? Number.MAX_SAFE_INTEGER) - (b.entry_seq ?? Number.MAX_SAFE_INTEGER) ||
        a.occurred_at.localeCompare(b.occurred_at) ||
        a.id.localeCompare(b.id));
    if (lines.some((r) => r.reconciled)) {
      toast("O escopo tem linha conciliada; desconcilie antes de reordenar.", "error");
      return;
    }
    const idx = lines.findIndex((r) => r.id === row.id);
    const swap = idx + dir;
    if (idx < 0 || swap < 0 || swap >= lines.length) return;
    [lines[idx], lines[swap]] = [lines[swap], lines[idx]];
    try {
      await reorderFinanceLines(
        lines.map((r) => ({
          kind: r.kind === "group" ? "group" : "transaction",
          id: r.kind === "group" ? (r.group_id ?? r.id) : (r.tx?.id ?? r.id),
        })),
      );
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao reordenar", "error");
    }
  }

  const rows = useMemo<ReconRow[]>(() => {
    const knownGroups = new Map(groups.map((g) => [g.id, g]));
    const out: ReconRow[] = [];
    const emittedGroups = new Set<string>();

    for (const g of groups) {
      emittedGroups.add(g.id);
      out.push({
        id: `group:${g.id}`,
        kind: "group",
        group_id: g.id,
        branch_id: g.branch_id,
        account_id: g.account_id,
        occurred_at: g.occurred_at,
        created_at: g.created_at,
        entry_seq: g.entry_seq,
        type: g.type,
        category_name: undefined,
        description: g.description,
        account_name: g.account_name,
        amount: g.total_amount,
        reconciled: g.reconciled,
        member_count: g.member_count,
        attachment_count: g.attachment_count,
        group: g,
      });
    }

    for (const t of txns) {
      if (t.voided_at) continue;
      if (t.group_id) {
        if (knownGroups.has(t.group_id)) continue;
        // Grupo fora do retorno (fallback): agrega os membros por group_id.
        if (emittedGroups.has(t.group_id)) continue;
        const members = txns.filter((m) => m.group_id === t.group_id && !m.voided_at);
        if (members.length > 0) {
          emittedGroups.add(t.group_id);
          const seqs = members
            .map((m) => m.entry_seq)
            .filter((n): n is number => typeof n === "number");
          out.push({
            id: `group:${t.group_id}`,
            kind: "group",
            group_id: t.group_id,
            branch_id: t.branch_id,
            account_id: t.account_id,
            occurred_at: t.occurred_at,
            created_at: t.created_at ?? "",
            entry_seq: seqs.length ? Math.min(...seqs) : undefined,
            type: t.type,
            description: t.description,
            account_name: t.account_name,
            amount: members.reduce((sum, m) => sum + m.amount, 0),
            reconciled: members.every((m) => m.reconciled),
            member_count: members.length,
            attachment_count: members.reduce((sum, m) => sum + (m.attachment_count ?? 0), 0),
          });
        }
        continue;
      }
      out.push({
        id: t.id,
        kind: "tx",
        branch_id: t.branch_id,
        account_id: t.account_id,
        occurred_at: t.occurred_at,
        created_at: t.created_at ?? "",
        entry_seq: t.entry_seq,
        type: t.type,
        category_name: t.category_name,
        description: t.description,
        account_name: t.account_name,
        amount: t.amount,
        reconciled: !!t.reconciled,
        attachment_count: t.attachment_count,
        tx: t,
      });
    }
    return out;
  }, [txns, groups]);

  const accountRows = useMemo(
    () => (accountId ? rows.filter((r) => r.account_id === accountId) : rows),
    [rows, accountId],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return accountRows.filter((t) => {
      const d = t.occurred_at.slice(0, 10);
      if (from && d < from) return false;
      if (to && d > to) return false;
      if (status === "reconciled" && !t.reconciled) return false;
      if (status === "pending" && t.reconciled) return false;
      if (q) {
        const hay = [t.description, t.category_name, t.account_name, t.tx?.donor_name, t.tx?.supplier_name]
          .filter(Boolean).join(" ").toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [accountRows, from, to, status, query]);

  const summary = useMemo(() => {
    const selectedAccounts = accountId ? accounts.filter((a) => a.id === accountId) : accounts;
    const initial = selectedAccounts.reduce((sum, a) => sum + a.initial_balance, 0);
    let opening = initial;
    let income = 0;
    let expense = 0;
    for (const r of accountRows) {
      const d = r.occurred_at.slice(0, 10);
      const signed = r.type === "income" ? r.amount : -r.amount;
      if (from && d < from) {
        opening += signed;
        continue;
      }
      if (to && d > to) continue;
      if (r.type === "income") income += r.amount;
      else expense += r.amount;
    }
    return { opening, income, expense, closing: opening + income - expense };
  }, [accounts, accountId, accountRows, from, to]);

  // Saldo apos cada linha, na mesma logica do extrato: acumula por conta
  // bancaria a partir do saldo inicial, na ordem cronologica do razao.
  const balanceAfter = useMemo(() => {
    const initial = new Map<string, number>();
    for (const a of accounts) initial.set(a.id, a.initial_balance);
    const ordered = rows
      .slice()
      .sort((a, b) =>
        `${a.occurred_at.slice(0, 10)}|${String(a.entry_seq ?? 999999999).padStart(9, "0")}|${a.created_at}|${a.id}`
          .localeCompare(`${b.occurred_at.slice(0, 10)}|${String(b.entry_seq ?? 999999999).padStart(9, "0")}|${b.created_at}|${b.id}`));
    const current = new Map<string, number>();
    const out = new Map<string, number>();
    for (const r of ordered) {
      const key = r.account_id ?? "";
      const prev = current.get(key) ?? initial.get(key) ?? 0;
      const next = prev + (r.type === "income" ? r.amount : -r.amount);
      current.set(key, next);
      out.set(r.id, next);
    }
    return out;
  }, [rows, accounts]);

  const periodRows = useMemo(
    () => rows.filter((r) => {
      const d = r.occurred_at.slice(0, 10);
      return (!closeForm.period_start || d >= closeForm.period_start) && (!closeForm.period_end || d <= closeForm.period_end);
    }),
    [rows, closeForm.period_start, closeForm.period_end],
  );
  const periodPending = useMemo(() => periodRows.filter((r) => !r.reconciled).length, [periodRows]);

  function openClosePeriod() {
    setCloseForm({ title: "Conciliacao financeira", period_start: from, period_end: to });
    setShowClose(true);
  }

  async function submitClose(e: React.FormEvent) {
    e.preventDefault();
    if (!closeForm.period_start || !closeForm.period_end) {
      toast("Informe o inicio e o fim do periodo.", "error");
      return;
    }
    if (closeForm.period_end < closeForm.period_start) {
      toast("O fim do periodo nao pode ser antes do inicio.", "error");
      return;
    }
    const overlapping = reconciliations.filter(
      (r) => r.period_start <= closeForm.period_end && r.period_end >= closeForm.period_start,
    );
    if (overlapping.some((r) => r.status === "conciliada")) {
      toast("Ja existe periodo encerrado sobreposto a este intervalo.", "error");
      return;
    }
    const exact = overlapping.find(
      (r) => r.status === "aberta" && r.period_start === closeForm.period_start && r.period_end === closeForm.period_end,
    );
    if (overlapping.length > 0 && !exact) {
      toast("Ja existe conciliacao aberta com periodo sobreposto. Ajuste as datas.", "error");
      return;
    }
    if (periodPending > 0 && !confirm(`Existem ${periodPending} linha(s) pendente(s). Conciliar tudo e encerrar o periodo?`)) {
      return;
    }
    setClosing(true);
    try {
      let rec = exact;
      if (!rec) {
        rec = await createReconciliation({
          title: closeForm.title || "Conciliacao financeira",
          period_start: closeForm.period_start,
          period_end: closeForm.period_end,
        });
      }
      // Concilia grupos e lancamentos avulsos do periodo antes de encerrar.
      for (const r of periodRows) {
        if (r.reconciled) continue;
        if (r.kind === "group" && r.group_id) {
          await reconcileTransactionGroup(r.group_id, true);
        }
      }
      const txIDs = periodRows
        .filter((r) => !r.reconciled && r.kind === "tx" && r.tx)
        .map((r) => r.tx!.id);
      for (let i = 0; i < txIDs.length; i += 500) {
        await reconcileTransactions(txIDs.slice(i, i + 500), true);
      }
      await conciliateReconciliation(rec.id);
      toast("Periodo conciliado e travado.");
      setShowClose(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao encerrar periodo", "error");
    } finally {
      setClosing(false);
    }
  }

  const activeClosed = useMemo(
    () => closedRanges.filter((r) => (!from || r.end >= from) && (!to || r.start <= to)),
    [closedRanges, from, to],
  );

  const columns: Column<ReconRow>[] = [
    {
      key: "occurred_at",
      label: "Data",
      sortable: true,
      width: "w-28",
      sortValue: (t) => `${t.occurred_at.slice(0, 10)}|${String(t.entry_seq ?? 999999999).padStart(9, "0")}`,
      render: (t) => <span className="text-zinc-500">{datePt(t.occurred_at)}</span>,
    },
    {
      key: "entry_seq",
      label: "Seq.",
      sortable: true,
      width: "w-16",
      sortValue: (t) => t.entry_seq ?? 999999999,
      render: (t) => (
        <div className="flex items-center gap-1">
          <span className="text-xs tabular-nums text-zinc-400">{t.entry_seq ?? "-"}</span>
          {canWrite && !t.reconciled && !isDateClosed(t.occurred_at) && (
            <div className="flex flex-col">
              <button
                type="button"
                aria-label="Mover para cima"
                title="Mover para cima"
                className="text-zinc-400 hover:text-sky-600"
                onClick={() => moveRow(t, -1)}
              >
                <ChevronUp className="h-3 w-3" />
              </button>
              <button
                type="button"
                aria-label="Mover para baixo"
                title="Mover para baixo"
                className="text-zinc-400 hover:text-sky-600"
                onClick={() => moveRow(t, 1)}
              >
                <ChevronDown className="h-3 w-3" />
              </button>
            </div>
          )}
        </div>
      ),
    },
    {
      key: "type",
      label: "Tipo",
      sortable: true,
      width: "w-24",
      render: (t) => <Badge tone={t.type === "income" ? "green" : "red"}>{t.type === "income" ? "Entrada" : "Saida"}</Badge>,
    },
    {
      key: "category_name",
      label: "Conta contabil",
      sortable: true,
      render: (t) =>
        t.kind === "group" ? (
          <Badge tone="indigo" className="gap-1">
            <Layers className="h-3 w-3" /> {t.member_count ?? 0} itens
          </Badge>
        ) : (
          <span className="text-zinc-500">{t.category_name ?? "-"}</span>
        ),
    },
    {
      key: "description",
      label: "Descricao",
      sortable: true,
      render: (t) => (
        <div className="flex items-center gap-1">
          {t.kind === "group" && (
            <button
              type="button"
              onClick={() => toggleExpanded(t.id)}
              title={expandedRows.has(t.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
              aria-label={expandedRows.has(t.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
              className="text-zinc-400 hover:text-sky-600"
            >
              {expandedRows.has(t.id) ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            </button>
          )}
          <span className="text-zinc-500">{t.description ?? "-"}</span>
        </div>
      ),
    },
    { key: "account_name", label: "Conta", sortable: true, render: (t) => <span className="text-zinc-500">{t.account_name ?? "-"}</span> },
    {
      key: "attachment_count",
      label: "Anexos",
      align: "center",
      width: "w-20",
      render: (t) => {
        if (t.kind === "group" && t.group_id) {
          return (
            <button
              type="button"
              onClick={() => setAttachOwner({ groupId: t.group_id })}
              className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
              title={t.attachment_count ? `Ver ${t.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
              aria-label={t.attachment_count ? `Ver ${t.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
            >
              <Paperclip className="h-3.5 w-3.5" />
              {!!t.attachment_count && <span className="text-[10px] font-medium">{t.attachment_count}</span>}
            </button>
          );
        }
        if (t.tx && t.attachment_count) {
          return (
            <button
              type="button"
              onClick={() => setAttachOwner({ transactionId: t.tx!.id })}
              className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
              title={`Ver ${t.attachment_count} anexo(s)`}
              aria-label={`Ver ${t.attachment_count} anexo(s)`}
            >
              <Paperclip className="h-3.5 w-3.5" />
              <span className="text-[10px] font-medium">{t.attachment_count}</span>
            </button>
          );
        }
        return <span className="text-zinc-300">-</span>;
      },
    },
    {
      key: "amount",
      label: "Valor",
      sortable: true,
      align: "right",
      render: (t) => <span className={`font-medium ${t.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(t.amount)}</span>,
    },
    {
      key: "balance_after",
      label: "Saldo apos",
      sortable: true,
      align: "right",
      width: "w-32",
      sortValue: (t) => balanceAfter.get(t.id) ?? 0,
      render: (t) => <span className="font-medium tabular-nums">{currency(balanceAfter.get(t.id) ?? 0)}</span>,
    },
    {
      key: "reconciled",
      label: "Conciliado",
      align: "right",
      width: "w-28",
      render: (t) => {
        const closed = isDateClosed(t.occurred_at);
        const disabled = !canReconcile || busyId === t.id || closed;
        return (
          <button
            type="button"
            onClick={() => toggle(t)}
            disabled={disabled}
            aria-label={t.reconciled ? "Desconciliar" : "Conciliar"}
            title={
              closed
                ? "Periodo encerrado: nao e possivel alterar a conciliacao."
                : t.kind === "group"
                  ? (t.reconciled ? "Desconciliar grupo" : "Conciliar grupo")
                  : (t.reconciled ? "Desconciliar" : "Conciliar")
            }
            className={`inline-flex items-center gap-1.5 rounded px-1.5 py-0.5 text-xs ${disabled ? "cursor-not-allowed opacity-60" : "hover:bg-zinc-100 dark:hover:bg-zinc-800"}`}
          >
            {t.reconciled ? (
              <CheckCircle2 className="h-5 w-5 text-emerald-600" />
            ) : (
              <Circle className="h-5 w-5 text-zinc-300" />
            )}
            <span className={t.reconciled ? "text-emerald-700 dark:text-emerald-400" : "text-zinc-400"}>
              {t.reconciled ? "Conciliado" : "Pendente"}
            </span>
          </button>
        );
      },
    },
  ];

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <div className="relative min-w-[220px] flex-1">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input className="pl-8" placeholder="Buscar descricao, conta, doador..." value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        <Select className="w-52" value={accountId} onChange={(e) => setAccountId(e.target.value)} aria-label="Conta bancaria">
          <option value="">Todas as contas</option>
          {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
        </Select>
        <span className="text-xs text-zinc-500">Periodo</span>
        <Input type="date" className="w-40" value={from} onChange={(e) => setFrom(e.target.value)} aria-label="Data inicial" />
        <span className="text-xs text-zinc-400">a</span>
        <Input type="date" className="w-40" value={to} onChange={(e) => setTo(e.target.value)} aria-label="Data final" />
        <Select className="w-40" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Conciliacao</option>
          <option value="reconciled">Conciliados</option>
          <option value="pending">Pendentes</option>
        </Select>
        <Button variant="ghost" size="sm" onClick={() => { setQuery(""); setStatus(""); }}>
          Limpar
        </Button>
        {canReconcile && (
          <Button size="sm" onClick={openClosePeriod}>
            <Lock className="h-4 w-4" /> Encerrar periodo
          </Button>
        )}
      </div>

      <div className="mb-4 grid gap-4 md:grid-cols-4">
        <Card>
          <p className="text-sm text-zinc-500">Saldo anterior</p>
          <p className="mt-1 text-2xl font-semibold">{currency(summary.opening)}</p>
        </Card>
        <Card>
          <p className="text-sm text-zinc-500">Entradas</p>
          <p className="mt-1 text-2xl font-semibold text-emerald-600">{currency(summary.income)}</p>
        </Card>
        <Card>
          <p className="text-sm text-zinc-500">Saidas</p>
          <p className="mt-1 text-2xl font-semibold text-red-600">{currency(summary.expense)}</p>
        </Card>
        <Card>
          <p className="text-sm text-zinc-500">Saldo final</p>
          <p className={`mt-1 text-2xl font-semibold ${summary.closing >= 0 ? "text-sky-700" : "text-red-600"}`}>
            {currency(summary.closing)}
          </p>
        </Card>
      </div>

      {activeClosed.length > 0 && (
        <Card className="mb-3 border-emerald-200 dark:border-emerald-900/50">
          <p className="flex items-center gap-2 text-sm font-medium text-emerald-700 dark:text-emerald-400">
            <CheckCircle2 className="h-4 w-4" /> Periodo encerrado
          </p>
          <p className="mt-1 text-xs text-zinc-500">
            {activeClosed.map((r) => `${r.title} (${datePt(r.start)} a ${datePt(r.end)})`).join(" · ")}.
            Nenhum lancamento pode ser criado, alterado ou excluido nesse intervalo.
          </p>
        </Card>
      )}

      <p className="mb-2 flex items-center gap-1 text-xs text-zinc-500">
        <CheckCircle2 className="h-3.5 w-3.5" />
        Marque um lancamento individualmente pelo icone; ele fica travado imediatamente. Lancamentos agrupados sao
        conciliados em bloco (uma unica linha). O encerramento do periodo pode ser feito aqui mesmo.
      </p>
      <DataTable
        columns={columns}
        data={filtered}
        keyExtractor={(t) => t.id}
        loading={loading}
        sort={{ column: sortColumn, direction: sortDirection, onSort: handleSort }}
        expansion={{
          expandedKeys: expandedRows,
          onToggleExpand: toggleExpanded,
          isExpandable: (t) => t.kind === "group",
          renderExpanded: (t) => (t.group_id ? <GroupMembersInline groupId={t.group_id} from={from} to={to} onOpenTransaction={(m) => setAttachOwner({ transactionId: m.id })} /> : null),
        }}
        emptyMessage="Nenhum lancamento no filtro"
        emptyDescription="Ajuste o periodo, o status ou a busca."
        maxHeight="max(280px, calc(100vh - 380px))"
        compact
      />

      <Modal open={showClose} onClose={() => { if (!closing) setShowClose(false); }} title="Encerrar periodo">
        <form onSubmit={submitClose} className="space-y-3 text-sm">
          <p className="text-zinc-600 dark:text-zinc-300">
            Ao encerrar, o periodo sera <strong>travado</strong>: nao sera possivel lancar, editar, estornar ou excluir
            lancamentos nesse intervalo. Todos os lancamentos serao conciliados automaticamente antes do encerramento.
            Apenas a Sede podera excluir a conciliacao para destravar.
          </p>
          <Field label="Titulo">
            <Input value={closeForm.title} onChange={(e) => setCloseForm({ ...closeForm, title: e.target.value })} />
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Periodo - de *">
              <Input type="date" required value={closeForm.period_start} onChange={(e) => setCloseForm({ ...closeForm, period_start: e.target.value })} />
            </Field>
            <Field label="Periodo - ate *">
              <Input type="date" required value={closeForm.period_end} onChange={(e) => setCloseForm({ ...closeForm, period_end: e.target.value })} />
            </Field>
          </div>
          <p className="text-xs text-zinc-500">
            {periodRows.length} linha(s) no periodo · {periodPending} pendente(s).
          </p>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowClose(false)} disabled={closing}>Cancelar</Button>
            <Button type="submit" disabled={closing}>
              <Lock className="h-4 w-4" /> {closing ? "Encerrando..." : "Conciliar e encerrar"}
            </Button>
          </div>
        </form>
      </Modal>

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
