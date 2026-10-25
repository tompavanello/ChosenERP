"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { CheckCircle2, Search } from "lucide-react";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Badge } from "@/components/ui/badge";
import { Input, Select } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { listAccounts, listTransactions, reconcileTransaction, type BankAccount, type Transaction } from "@/lib/api";
import { currency, datePt, todayISO } from "@/lib/format";

/**
 * Visao de lancamentos avulsos da conciliacao: permite marcar/desmarcar um
 * unico lancamento sem precisar criar um periodo. A ordenacao segue a mesma
 * regra da auditoria (data, sequencia, criacao).
 */
export function ReconciliationEntries() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canReconcile = hasPerm("finance.reconcile");

  const now = new Date();
  const [from, setFrom] = useState(todayISO(new Date(now.getFullYear(), now.getMonth(), 1)));
  const [to, setTo] = useState(todayISO(now));
  const [status, setStatus] = useState("");
  const [query, setQuery] = useState("");
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [busyId, setBusyId] = useState("");
  const [sortColumn, setSortColumn] = useState("occurred_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");

  const load = useCallback(async () => {
    try {
      const [r, a] = await Promise.all([
        listTransactions(),
        listAccounts().catch(() => ({ accounts: [] })),
      ]);
      setTxns(Array.isArray(r.transactions) ? r.transactions : []);
      setAccounts(Array.isArray(a.accounts) ? a.accounts : []);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar lancamentos", "error");
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  async function toggle(t: Transaction) {
    setBusyId(t.id);
    try {
      await reconcileTransaction(t.id, !t.reconciled);
      toast(t.reconciled ? "Lancamento desconciliado." : "Lancamento conciliado e travado.");
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

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return txns.filter((t) => {
      if (t.voided_at) return false;
      if (from && t.occurred_at < from) return false;
      if (to && t.occurred_at > to) return false;
      if (status === "reconciled" && !t.reconciled) return false;
      if (status === "pending" && t.reconciled) return false;
      if (q) {
        const hay = [t.description, t.category_name, t.account_name, t.donor_name, t.supplier_name]
          .filter(Boolean).join(" ").toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [txns, from, to, status, query]);

  // Saldo apos cada lancamento, na mesma logica do extrato: acumula por conta
  // bancaria a partir do saldo inicial, na ordem cronologica do razao.
  const balanceAfter = useMemo(() => {
    const initial = new Map<string, number>();
    for (const a of accounts) initial.set(a.id, a.initial_balance);
    const ordered = txns
      .filter((t) => !t.voided_at)
      .slice()
      .sort((a, b) =>
        `${a.occurred_at}|${String(a.entry_seq ?? 999999999).padStart(9, "0")}|${a.created_at ?? ""}`
          .localeCompare(`${b.occurred_at}|${String(b.entry_seq ?? 999999999).padStart(9, "0")}|${b.created_at ?? ""}`));
    const current = new Map<string, number>();
    const out = new Map<string, number>();
    for (const t of ordered) {
      const key = t.account_id ?? "";
      const prev = current.get(key) ?? initial.get(key) ?? 0;
      const next = prev + (t.type === "income" ? t.amount : -t.amount);
      current.set(key, next);
      out.set(t.id, next);
    }
    return out;
  }, [txns, accounts]);

  const columns: Column<Transaction>[] = [
    {
      key: "occurred_at",
      label: "Data",
      sortable: true,
      width: "w-28",
      sortValue: (t) => `${t.occurred_at}|${String(t.entry_seq ?? 999999999).padStart(9, "0")}|${t.created_at ?? ""}`,
      render: (t) => <span className="text-zinc-500">{datePt(t.occurred_at)}</span>,
    },
    {
      key: "entry_seq",
      label: "Seq.",
      sortable: true,
      width: "w-16",
      sortValue: (t) => t.entry_seq ?? 999999999,
      render: (t) => <span className="text-xs tabular-nums text-zinc-400">{t.entry_seq ?? "-"}</span>,
    },
    {
      key: "type",
      label: "Tipo",
      sortable: true,
      width: "w-24",
      render: (t) => <Badge tone={t.type === "income" ? "green" : "red"}>{t.type === "income" ? "Entrada" : "Saida"}</Badge>,
    },
    { key: "category_name", label: "Conta contabil", sortable: true, render: (t) => <span className="text-zinc-500">{t.category_name ?? "-"}</span> },
    { key: "description", label: "Descricao", sortable: true, render: (t) => <span className="text-zinc-500">{t.description ?? "-"}</span> },
    { key: "account_name", label: "Conta", sortable: true, render: (t) => <span className="text-zinc-500">{t.account_name ?? "-"}</span> },
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
      width: "w-24",
      render: (t) => (
        <input
          type="checkbox"
          className="h-4 w-4 rounded border-zinc-300 text-sky-600"
          checked={!!t.reconciled}
          disabled={!canReconcile || busyId === t.id}
          onChange={() => toggle(t)}
          title={t.reconciled ? "Desconciliar" : "Conciliar"}
        />
      ),
    },
  ];

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <div className="relative min-w-[220px] flex-1">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input className="pl-8" placeholder="Buscar descricao, conta, doador..." value={query} onChange={(e) => setQuery(e.target.value)} />
        </div>
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
      </div>
      <p className="mb-2 flex items-center gap-1 text-xs text-zinc-500">
        <CheckCircle2 className="h-3.5 w-3.5" />
        Marque um lançamento individualmente; ele fica travado imediatamente. O encerramento por periodo fica na aba Períodos.
      </p>
      <DataTable
        columns={columns}
        data={filtered}
        keyExtractor={(t) => t.id}
        loading={loading}
        sort={{ column: sortColumn, direction: sortDirection, onSort: handleSort }}
        emptyMessage="Nenhum lancamento no filtro"
        emptyDescription="Ajuste o periodo, o status ou a busca."
        compact
      />
    </div>
  );
}
