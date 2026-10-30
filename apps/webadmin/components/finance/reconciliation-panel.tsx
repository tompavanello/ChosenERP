"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Search, Lock, ArrowLeft, Trash2, CheckCircle2, ChevronDown, ChevronRight, ChevronUp, Layers, Paperclip } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Input, Field } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listReconciliations, createReconciliation, getReconciliation,
  conciliateReconciliation, deleteReconciliation, reconcileTransaction, reconcileTransactions,
  reconcileTransactionGroup, reorderFinanceLines,
  type FinancialReconciliation, type ReconciliationItem,
} from "@/lib/api";
import { currency, datePt, dateTimePt, todayISO } from "@/lib/format";
import { GroupMembersInline } from "@/components/finance/group-members-inline";
import { AttachmentDrawer, type AttachmentOwner } from "@/components/finance/attachments";

type PeriodRow = {
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
  member_count: number;
  total_members: number;
  attachment_count: number;
  tx?: ReconciliationItem;
};

function periodScopeKey(row: PeriodRow): string {
  return `${row.branch_id}|${row.account_id ?? ""}|${row.occurred_at.slice(0, 10)}`;
}

export function ReconciliationPanel() {
  const { toast } = useToast();
  const { hasPerm, user } = useAuth();
  const canReconcile = hasPerm("finance.reconcile");
  const canWrite = hasPerm("finance.write");
  const isHQ = user?.role === "super_admin" || user?.role === "admin_sede";

  const [list, setList] = useState<FinancialReconciliation[] | null>(null);
  const [selected, setSelected] = useState<FinancialReconciliation | null>(null);
  const [items, setItems] = useState<ReconciliationItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);
  const [query, setQuery] = useState("");
  const [reconcilingId, setReconcilingId] = useState("");
  const [reconcilingAll, setReconcilingAll] = useState(false);
  const [sortColumn, setSortColumn] = useState("occurred_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [attachOwner, setAttachOwner] = useState<AttachmentOwner | null>(null);

  const [showNew, setShowNew] = useState(false);
  const now = new Date();
  const [form, setForm] = useState({
    title: "Conciliacao financeira",
    period_start: todayISO(new Date(now.getFullYear(), now.getMonth(), 1)),
    period_end: todayISO(now),
    notes: "",
  });
  const [saving, setSaving] = useState(false);
  const [confirmConciliate, setConfirmConciliate] = useState(false);
  const [conciliating, setConciliating] = useState(false);

  const load = useCallback(async () => {
    try {
      setList((await listReconciliations()).reconciliations);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar conciliacoes", "error");
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  async function openReconciliation(rec: FinancialReconciliation) {
    setSelected(rec);
    setLoadingItems(true);
    setSortColumn("occurred_at");
    setSortDirection("asc");
    setExpandedRows(new Set());
    try {
      const res = await getReconciliation(rec.id);
      setSelected(res.reconciliation);
      setItems(res.items);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoadingItems(false);
    }
  }

  async function reloadItems() {
    if (!selected) return;
    const res = await getReconciliation(selected.id);
    setSelected(res.reconciliation);
    setItems(res.items);
  }

  async function createNew(e: React.FormEvent) {
    e.preventDefault();
    if (form.period_end < form.period_start) {
      toast("O fim do periodo nao pode ser antes do inicio.", "error");
      return;
    }
    setSaving(true);
    try {
      const rec = await createReconciliation({ ...form, notes: form.notes || undefined });
      toast("Conciliacao criada.");
      setShowNew(false);
      await load();
      await openReconciliation(rec);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar conciliacao", "error");
    } finally {
      setSaving(false);
    }
  }

  async function doConciliate() {
    if (!selected) return;
    setConciliating(true);
    try {
      const rec = await conciliateReconciliation(selected.id);
      setSelected(rec);
      setConfirmConciliate(false);
      await load();
      await openReconciliation(rec);
      toast("Periodo conciliado e travado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao conciliar", "error");
    } finally {
      setConciliating(false);
    }
  }

  function toggleExpanded(id: string) {
    setExpandedRows((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function toggleRow(row: PeriodRow) {
    if (!selected || selected.status === "conciliada") return;
    if (row.kind === "group") {
      if (!row.group_id) return;
      if (row.total_members > row.member_count) {
        toast("O grupo tem lancamentos fora deste periodo; concilie-o pelo periodo completo.", "error");
        return;
      }
      setReconcilingId(row.id);
      try {
        await reconcileTransactionGroup(row.group_id, !row.reconciled);
        await reloadItems();
      } catch (err) {
        toast(err instanceof Error ? err.message : "Erro ao conciliar grupo", "error");
      } finally {
        setReconcilingId("");
      }
      return;
    }
    if (!row.tx) return;
    setReconcilingId(row.id);
    try {
      await reconcileTransaction(row.tx.id, !row.reconciled);
      await reloadItems();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao conciliar lancamento", "error");
    } finally {
      setReconcilingId("");
    }
  }

  async function moveRow(row: PeriodRow, dir: -1 | 1) {
    if (!selected || selected.status === "conciliada" || !canWrite) return;
    const key = periodScopeKey(row);
    const lines = periodRows
      .filter((r) => periodScopeKey(r) === key)
      .sort((a, b) =>
        (a.entry_seq ?? Number.MAX_SAFE_INTEGER) - (b.entry_seq ?? Number.MAX_SAFE_INTEGER) ||
        a.occurred_at.localeCompare(b.occurred_at) ||
        a.created_at.localeCompare(b.created_at) ||
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
      await reloadItems();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao reordenar", "error");
    }
  }

  async function reconcileAllPending() {
    if (!selected) return;
    const pending = periodRows.filter((row) => !row.reconciled);
    if (pending.length === 0) return;
    const fullGroups = pending.filter(
      (row) => row.kind === "group" && row.group_id && row.total_members <= row.member_count,
    );
    const partialGroups = pending.filter(
      (row) => row.kind === "group" && row.total_members > row.member_count,
    );
    const txIDs = pending
      .filter((row) => row.kind === "tx" && row.tx)
      .map((row) => row.tx!.id);
    if (fullGroups.length === 0 && txIDs.length === 0) {
      toast("Os grupos pendentes tem lancamentos fora do periodo; concilie pelo periodo completo.", "error");
      return;
    }
    if (!confirm(`Conciliar ${pending.length} linha(s) pendente(s) deste periodo?`)) return;
    setReconcilingAll(true);
    try {
      for (const row of fullGroups) {
        if (row.group_id) await reconcileTransactionGroup(row.group_id, true);
      }
      if (txIDs.length > 0) await reconcileTransactions(txIDs, true);
      await reloadItems();
      toast(
        partialGroups.length > 0
          ? `Linhas conciliadas. ${partialGroups.length} grupo(s) com lancamentos fora do periodo ficaram pendentes.`
          : "Linhas pendentes conciliadas.",
      );
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao conciliar lancamentos", "error");
    } finally {
      setReconcilingAll(false);
    }
  }

  const handleItemSort = (col: string) => {
    if (sortColumn === col) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortColumn(col);
      setSortDirection("asc");
    }
  };

  async function remove(rec: FinancialReconciliation) {
    if (!confirm(`Excluir a conciliacao "${rec.title}" (${datePt(rec.period_start)} a ${datePt(rec.period_end)})? O periodo sera DESTRAVADO.`)) return;
    try {
      await deleteReconciliation(rec.id);
      toast("Conciliacao excluida.");
      setSelected(null);
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (list ?? []).filter((r) => !q || r.title.toLowerCase().includes(q));
  }, [list, query]);

  const columns: Column<FinancialReconciliation>[] = [
    { key: "title", label: "Titulo", sortable: true, render: (r) => <span className="font-medium">{r.title}</span> },
    {
      key: "period_start",
      label: "Periodo",
      sortable: true,
      render: (r) => <span className="text-zinc-500">{datePt(r.period_start)} a {datePt(r.period_end)}</span>,
    },
    {
      key: "status",
      label: "Situacao",
      sortable: true,
      width: "w-32",
      render: (r) => (
        <Badge tone={r.status === "conciliada" ? "green" : "amber"}>
          {r.status === "conciliada" ? "Encerrado" : "Aberto"}
        </Badge>
      ),
    },
    { key: "total_items", label: "Lancamentos", sortable: true, align: "right", width: "w-28", render: (r) => <span className="text-zinc-500">{r.total_items}</span> },
    {
      key: "reconciled_items",
      label: "Conciliados",
      sortable: true,
      align: "right",
      width: "w-32",
      render: (r) => (
        <span className={r.pending_items > 0 ? "text-amber-600" : "text-emerald-600"}>
          {r.reconciled_items}/{r.total_items}
          {r.pending_items > 0 ? ` (${r.pending_items} pend.)` : ""}
        </span>
      ),
    },
    {
      key: "net",
      label: "Saldo",
      align: "right",
      width: "w-36",
      sortValue: (r) => r.total_income - r.total_expense,
      render: (r) => (
        <span className={r.total_income - r.total_expense >= 0 ? "text-emerald-600" : "text-red-600"}>
          {currency(r.total_income - r.total_expense)}
        </span>
      ),
    },
    {
      key: "actions",
      label: "",
      align: "right",
      width: "w-40",
      render: (r) => (
        <div className="flex justify-end gap-1">
          <Button size="sm" variant="outline" onClick={() => openReconciliation(r)}>
            <Search className="h-4 w-4" /> Abrir
          </Button>
          {isHQ && (
            <Button size="sm" variant="ghost" onClick={() => remove(r)} aria-label="Excluir conciliacao" title="Excluir (destrava o periodo)">
              <Trash2 className="h-4 w-4 text-red-500" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  const periodRows = useMemo<PeriodRow[]>(() => {
    const branchID = selected?.branch_id ?? "";
    const out: PeriodRow[] = [];
    const groups = new Map<string, ReconciliationItem[]>();
    for (const it of items) {
      if (it.group_id) {
        const arr = groups.get(it.group_id) ?? [];
        arr.push(it);
        groups.set(it.group_id, arr);
        continue;
      }
      out.push({
        id: it.id,
        kind: "tx",
        branch_id: branchID,
        account_id: it.account_id,
        occurred_at: it.occurred_at,
        created_at: it.created_at,
        entry_seq: it.entry_seq,
        type: it.type,
        category_name: it.category_name,
        description: it.description,
        account_name: it.account_name,
        amount: it.amount,
        reconciled: it.reconciled,
        member_count: 1,
        total_members: 1,
        attachment_count: it.attachment_count,
        tx: it,
      });
    }
    for (const [gid, members] of groups) {
      const first = members[0];
      const seqs = members
        .map((m) => m.entry_seq)
        .filter((n): n is number => typeof n === "number");
      out.push({
        id: `group:${gid}`,
        kind: "group",
        group_id: gid,
        branch_id: branchID,
        account_id: first.account_id,
        occurred_at: first.occurred_at,
        created_at: first.created_at,
        entry_seq: first.group_entry_seq ?? (seqs.length ? Math.min(...seqs) : undefined),
        type: first.type,
        description: first.group_description ?? first.description,
        account_name: first.account_name,
        amount: members.reduce((sum, m) => sum + m.amount, 0),
        reconciled: members.every((m) => m.reconciled),
        member_count: members.length,
        total_members: first.group_member_count && first.group_member_count > 0
          ? first.group_member_count
          : members.length,
        attachment_count: first.group_attachment_count ?? members.reduce((sum, m) => sum + m.attachment_count, 0),
      });
    }
    return out;
  }, [items, selected]);

  const itemBalances = useMemo(() => {
    const ordered = periodRows.slice().sort((a, b) =>
      `${a.occurred_at}|${String(a.entry_seq ?? 999999999).padStart(9, "0")}|${a.created_at}|${a.id}`
        .localeCompare(`${b.occurred_at}|${String(b.entry_seq ?? 999999999).padStart(9, "0")}|${b.created_at}|${b.id}`));
    const out = new Map<string, number>();
    let running = 0;
    for (const row of ordered) {
      running += row.type === "income" ? row.amount : -row.amount;
      out.set(row.id, running);
    }
    return out;
  }, [periodRows]);

  const itemColumns: Column<PeriodRow>[] = [
    {
      key: "occurred_at",
      label: "Data",
      sortable: true,
      width: "w-28",
      sortValue: (row) => `${row.occurred_at}|${String(row.entry_seq ?? 999999999).padStart(9, "0")}|${row.created_at}`,
      render: (row) => <span className="text-zinc-500">{datePt(row.occurred_at)}</span>,
    },
    {
      key: "entry_seq",
      label: "Seq.",
      sortable: true,
      width: "w-16",
      sortValue: (row) => row.entry_seq ?? 999999999,
      render: (row) => (
        <div className="flex items-center gap-1">
          <span className="text-xs tabular-nums text-zinc-400">{row.entry_seq ?? "-"}</span>
          {canWrite && selected?.status !== "conciliada" && !row.reconciled && (
            <div className="flex flex-col">
              <button
                type="button"
                aria-label="Mover para cima"
                title="Mover para cima"
                className="text-zinc-400 hover:text-sky-600"
                onClick={() => moveRow(row, -1)}
              >
                <ChevronUp className="h-3 w-3" />
              </button>
              <button
                type="button"
                aria-label="Mover para baixo"
                title="Mover para baixo"
                className="text-zinc-400 hover:text-sky-600"
                onClick={() => moveRow(row, 1)}
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
      render: (row) => <Badge tone={row.type === "income" ? "green" : "red"}>{row.type === "income" ? "Entrada" : "Saida"}</Badge>,
    },
    {
      key: "category_name",
      label: "Conta contabil",
      sortable: true,
      render: (row) =>
        row.kind === "group" ? (
          <Badge tone="indigo" className="gap-1">
            <Layers className="h-3 w-3" />{" "}
            {row.total_members > row.member_count
              ? `${row.member_count}/${row.total_members}`
              : row.member_count}{" "}
            itens
          </Badge>
        ) : (
          <span className="text-zinc-500">{row.category_name ?? "-"}</span>
        ),
    },
    {
      key: "description",
      label: "Descricao",
      sortable: true,
      render: (row) => (
        <div className="flex items-center gap-1">
          {row.kind === "group" && (
            <button
              type="button"
              onClick={() => toggleExpanded(row.id)}
              title={expandedRows.has(row.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
              aria-label={expandedRows.has(row.id) ? "Recolher lancamentos" : "Ver lancamentos do agrupamento"}
              className="text-zinc-400 hover:text-sky-600"
            >
              {expandedRows.has(row.id) ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
            </button>
          )}
          <span className="text-zinc-500">{row.description ?? "-"}</span>
        </div>
      ),
    },
    {
      key: "attachment_count",
      label: "Anexos",
      align: "center",
      width: "w-20",
      render: (row) => {
        if (row.kind === "group" && row.group_id) {
          return (
            <button
              type="button"
              onClick={() => setAttachOwner({ groupId: row.group_id })}
              className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
              title={row.attachment_count ? `Ver ${row.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
              aria-label={row.attachment_count ? `Ver ${row.attachment_count} anexo(s)` : "Anexar comprovante ao agrupamento"}
            >
              <Paperclip className="h-3.5 w-3.5" />
              {!!row.attachment_count && <span className="text-[10px] font-medium">{row.attachment_count}</span>}
            </button>
          );
        }
        if (row.tx && row.attachment_count > 0) {
          return (
            <button
              type="button"
              onClick={() => setAttachOwner({ transactionId: row.tx!.id })}
              className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
              title={`Ver ${row.attachment_count} anexo(s)`}
              aria-label={`Ver ${row.attachment_count} anexo(s)`}
            >
              <Paperclip className="h-3.5 w-3.5" />
              <span className="text-[10px] font-medium">{row.attachment_count}</span>
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
      render: (row) => <span className={`font-medium ${row.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(row.amount)}</span>,
    },
    {
      key: "balance_after",
      label: "Saldo apos",
      sortable: true,
      align: "right",
      width: "w-32",
      sortValue: (row) => itemBalances.get(row.id) ?? 0,
      render: (row) => <span className="font-medium tabular-nums">{currency(itemBalances.get(row.id) ?? 0)}</span>,
    },
    {
      key: "reconciled",
      label: "Conciliado",
      align: "right",
      width: "w-24",
      render: (row) => {
        const partial = row.kind === "group" && row.total_members > row.member_count;
        return (
          <input
            type="checkbox"
            className="h-4 w-4 rounded border-zinc-300 text-sky-600"
            checked={row.reconciled}
            disabled={
              selected?.status === "conciliada" ||
              !canReconcile ||
              reconcilingId === row.id ||
              partial
            }
            onChange={() => toggleRow(row)}
            title={
              partial
                ? "O grupo tem lancamentos fora deste periodo; concilie-o pelo periodo completo."
                : row.kind === "group"
                  ? (row.reconciled ? "Desconciliar grupo" : "Conciliar grupo")
                  : (row.reconciled ? "Desconciliar" : "Conciliar")
            }
          />
        );
      },
    },
  ];

  if (selected) {
    const locked = selected.status === "conciliada";
    const net = selected.total_income - selected.total_expense;
    return (
      <div>
        <PageHeader
          title={selected.title}
          description={`Periodo: ${datePt(selected.period_start)} a ${datePt(selected.period_end)}`}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => { setSelected(null); load(); }}>
                <ArrowLeft className="h-4 w-4" /> Voltar
              </Button>
              {isHQ && (
                <Button variant="ghost" onClick={() => remove(selected)} title="Excluir (destrava o periodo)">
                  <Trash2 className="h-4 w-4 text-red-500" /> Excluir
                </Button>
              )}
            </div>
          }
        />

        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Badge tone={locked ? "green" : "amber"}>{locked ? "Encerrado" : "Aberto"}</Badge>
          <span className="text-sm text-zinc-500">
            {periodRows.length} linha(s) - {selected.total_items} lancamento(s) - Entradas {currency(selected.total_income)} - Saidas {currency(selected.total_expense)} - Saldo {currency(net)}
          </span>
          <span className="text-sm text-zinc-500">
            Conciliados {selected.reconciled_items}/{selected.total_items}
            {selected.pending_items > 0 ? ` - ${selected.pending_items} pendente(s)` : ""}
          </span>
          {canReconcile && !locked && (
            <div className="ml-auto flex gap-2">
              {selected.pending_items > 0 && (
                <Button size="sm" variant="outline" onClick={reconcileAllPending} disabled={reconcilingAll}>
                  <CheckCircle2 className="h-4 w-4" /> {reconcilingAll ? "Conciliando..." : "Conciliar pendentes"}
                </Button>
              )}
              <Button
                size="sm"
                onClick={() => setConfirmConciliate(true)}
                disabled={selected.pending_items > 0}
                title={selected.pending_items > 0 ? "Concilie todos os lancamentos antes de encerrar" : "Encerrar periodo"}
              >
                <Lock className="h-4 w-4" /> Encerrar periodo
              </Button>
            </div>
          )}
        </div>

        {selected.pending_items > 0 && !locked && (
          <p className="mb-3 text-xs text-amber-600">
            Encerramento bloqueado: concilie os {selected.pending_items} lancamento(s) pendente(s) primeiro.
          </p>
        )}

        {locked && (
          <Card className="mb-4 text-sm">
            <p className="flex items-center gap-2 font-medium text-emerald-700">
              <CheckCircle2 className="h-4 w-4" /> Periodo encerrado e travado
            </p>
            <p className="mt-1 text-zinc-500">
              Nenhum lancamento pode ser criado, alterado, estornado ou excluido neste periodo
              {selected.reconciled_at ? ` (encerrado em ${dateTimePt(selected.reconciled_at)})` : ""}.
            </p>
          </Card>
        )}

        <DataTable
          columns={itemColumns}
          data={periodRows}
          keyExtractor={(row) => row.id}
          loading={loadingItems}
          sort={{ column: sortColumn, direction: sortDirection, onSort: handleItemSort }}
          expansion={{
            expandedKeys: expandedRows,
            onToggleExpand: toggleExpanded,
            isExpandable: (row) => row.kind === "group",
            renderExpanded: (row) =>
              row.group_id ? (
                <GroupMembersInline
                  groupId={row.group_id}
                  from={selected?.period_start}
                  to={selected?.period_end}
                  onOpenTransaction={(m) => setAttachOwner({ transactionId: m.id })}
                />
              ) : null,
          }}
          emptyMessage="Sem lancamentos"
          emptyDescription="Nenhum lancamento no periodo desta conciliacao."
          maxHeight="max(280px, calc(100vh - 340px))"
          compact
        />

        <Modal open={confirmConciliate} onClose={() => setConfirmConciliate(false)} title="Encerrar periodo">
          <div className="space-y-3 text-sm">
            <p className="text-zinc-600">
              Ao encerrar, o periodo <strong>{datePt(selected.period_start)} a {datePt(selected.period_end)}</strong> sera
              <strong> travado</strong>: nao sera possivel lancar, editar, estornar ou excluir lancamentos nesse intervalo.
              Todos os lancamentos devem estar conciliados antes do encerramento.
              Apenas a Sede podera excluir esta conciliacao para destravar.
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setConfirmConciliate(false)}>Cancelar</Button>
              <Button onClick={doConciliate} disabled={conciliating || selected.pending_items > 0}>
                {conciliating ? "Encerrando..." : "Encerrar periodo"}
              </Button>
            </div>
          </div>
        </Modal>

        <AttachmentDrawer
          open={!!attachOwner}
          onClose={() => setAttachOwner(null)}
          owner={attachOwner ?? {}}
          canUpload={canWrite}
          onUploaded={reloadItems}
        />
      </div>
    );
  }

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <div className="relative min-w-[220px] flex-1">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input className="pl-8" placeholder="Buscar por titulo..." value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
        {canReconcile && (
          <Button onClick={() => setShowNew(true)}>
            <Plus className="h-4 w-4" /> Nova conciliacao
          </Button>
        )}
      </div>

      <DataTable
        columns={columns}
        data={filtered}
        keyExtractor={(r) => r.id}
        loading={list === null}
        emptyMessage={list && list.length > 0 ? "Nenhuma conciliacao no filtro" : "Nenhuma conciliacao"}
        emptyDescription={list && list.length > 0 ? "Ajuste a busca." : "Crie uma conciliacao para um periodo e concilie."}
        maxHeight="max(240px, calc(100vh - 260px))"
      />

      <Drawer open={showNew} onClose={() => setShowNew(false)} title="Nova conciliacao">
        <form onSubmit={createNew} className="space-y-3">
          <Field label="Titulo"><Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="Periodo - de *"><Input type="date" required value={form.period_start} onChange={(e) => setForm({ ...form, period_start: e.target.value })} /></Field>
          <Field label="Periodo - ate *"><Input type="date" required value={form.period_end} onChange={(e) => setForm({ ...form, period_end: e.target.value })} /></Field>
          <Field label="Observacoes"><Input value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowNew(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Criando..." : "Criar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
