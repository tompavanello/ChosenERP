"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowLeftRight, ArrowRight, Plus, Trash2, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Input, Field, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  createAccountTransfer, deleteAccountTransfer, getAccountBalances, listAccountTransfers,
  type AccountBalances, type AccountTransfer,
} from "@/lib/api";
import { ACCOUNT_TYPES, TRANSFER_KINDS } from "@/lib/constants";
import { currency, datePt, monthStartISO, todayISO } from "@/lib/format";

type Kind = "deposit" | "redeem" | "transfer";

export default function AccountTransfersPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("finance.write");

  const [balances, setBalances] = useState<AccountBalances | null>(null);
  const [transfers, setTransfers] = useState<AccountTransfer[] | null>(null);
  const [from, setFrom] = useState(monthStartISO());
  const [to, setTo] = useState(todayISO());
  const [loading, setLoading] = useState(true);

  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    kind: "deposit" as Kind,
    from_account_id: "",
    to_account_id: "",
    amount: "",
    occurred_at: todayISO(),
    description: "",
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [bal, list] = await Promise.all([
        getAccountBalances(),
        listAccountTransfers(from, to),
      ]);
      setBalances(bal);
      setTransfers(list.transfers);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar transferencias", "error");
    } finally {
      setLoading(false);
    }
  }, [from, to, toast]);

  useEffect(() => { load(); }, [load]);

  // Atalhos vindos das telas de contas/extrato: ?kind=deposit|redeem&from=...
  useEffect(() => {
    const p = new URLSearchParams(window.location.search);
    const kind = p.get("kind");
    if (kind === "deposit" || kind === "redeem" || kind === "transfer") {
      setForm((f) => ({
        ...f,
        kind,
        from_account_id: p.get("from") ?? "",
        to_account_id: p.get("to") ?? "",
      }));
      setOpen(true);
    }
  }, []);

  const accounts = useMemo(() => (balances?.accounts ?? []).filter((a) => a.is_active), [balances]);
  const fromAccount = accounts.find((a) => a.id === form.from_account_id);
  const toAccount = accounts.find((a) => a.id === form.to_account_id);
  const sameBranch = (a: { branch_id?: string }) =>
    !fromAccount || !fromAccount.branch_id || !a.branch_id || a.branch_id === fromAccount.branch_id;

  const fromOptions = useMemo(() => {
    if (form.kind === "deposit") return accounts.filter((a) => a.account_type !== "investment" && sameBranch(a));
    if (form.kind === "redeem") return accounts.filter((a) => a.account_type === "investment" && sameBranch(a));
    return accounts.filter(sameBranch);
  }, [accounts, form.kind, fromAccount]);
  const toOptions = useMemo(() => {
    let list = accounts;
    if (form.kind === "deposit") list = accounts.filter((a) => a.account_type === "investment");
    else if (form.kind === "redeem") list = accounts.filter((a) => a.account_type !== "investment");
    return list.filter((a) => a.id !== form.from_account_id && sameBranch(a));
  }, [accounts, form.kind, form.from_account_id, fromAccount]);

  const amount = Number(form.amount) || 0;

  function openNew(kind: Kind = "deposit", fromID = "", toID = "") {
    setForm({
      kind,
      from_account_id: fromID,
      to_account_id: toID,
      amount: "",
      occurred_at: todayISO(),
      description: "",
    });
    setOpen(true);
  }

  function changeKind(kind: Kind) {
    setForm((f) => ({ ...f, kind, from_account_id: "", to_account_id: "" }));
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.from_account_id || !form.to_account_id) {
      toast("Selecione as contas de origem e destino.", "error");
      return;
    }
    if (form.from_account_id === form.to_account_id) {
      toast("As contas de origem e destino precisam ser diferentes.", "error");
      return;
    }
    if (amount <= 0) {
      toast("Informe um valor positivo.", "error");
      return;
    }
    setSaving(true);
    try {
      await createAccountTransfer({
        from_account_id: form.from_account_id,
        to_account_id: form.to_account_id,
        kind: form.kind,
        amount,
        occurred_at: form.occurred_at,
        description: form.description.trim() || undefined,
      });
      toast("Transferencia registrada.");
      setOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao transferir", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(t: AccountTransfer) {
    if (!confirm(`Excluir a transferencia de ${currency(t.amount)} (${datePt(t.occurred_at)})? As duas pernas serao removidas.`)) return;
    try {
      await deleteAccountTransfer(t.id);
      toast("Transferencia excluida.");
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao excluir", "error");
    }
  }

  const columns: Column<AccountTransfer>[] = [
    { key: "occurred_at", label: "Data", sortable: true, width: "w-28", render: (t) => <span className="text-zinc-500">{datePt(t.occurred_at)}</span> },
    {
      key: "kind",
      label: "Tipo",
      sortable: true,
      width: "w-28",
      render: (t) => (
        <Badge tone={t.kind === "deposit" ? "green" : t.kind === "redeem" ? "amber" : "sky"}>
          {TRANSFER_KINDS[t.kind] ?? t.kind}
        </Badge>
      ),
    },
    {
      key: "from_account_name",
      label: "Origem",
      sortable: true,
      render: (t) => (
        <span className="inline-flex items-center gap-1.5 text-zinc-600">
          <Wallet className="h-3.5 w-3.5 text-zinc-400" /> {t.from_account_name ?? "-"}
        </span>
      ),
    },
    {
      key: "to_account_name",
      label: "Destino",
      sortable: true,
      render: (t) => (
        <span className="inline-flex items-center gap-1.5 text-zinc-600">
          <ArrowRight className="h-3.5 w-3.5 text-zinc-400" /> {t.to_account_name ?? "-"}
        </span>
      ),
    },
    {
      key: "amount",
      label: "Valor",
      sortable: true,
      align: "right",
      render: (t) => <span className="font-semibold tabular-nums text-sky-700">{currency(t.amount)}</span>,
    },
    { key: "description", label: "Descricao", sortable: true, render: (t) => <span className="text-zinc-500">{t.description ?? "-"}</span> },
    {
      key: "status",
      label: "Conciliacao",
      width: "w-40",
      render: (t) =>
        t.from_reconciled && t.to_reconciled ? (
          <Badge tone="green">Conciliada</Badge>
        ) : t.from_reconciled || t.to_reconciled ? (
          <Badge tone="amber">Parcialmente conciliada</Badge>
        ) : (
          <Badge tone="zinc">Pendente</Badge>
        ),
    },
    {
      key: "actions",
      label: "",
      align: "right",
      width: "w-16",
      render: (t) =>
        canWrite ? (
          <Button variant="ghost" size="sm" onClick={() => remove(t)} title="Excluir transferencia">
            <Trash2 className="h-4 w-4 text-red-500" />
          </Button>
        ) : null,
    },
  ];

  return (
    <div className="page">
      <PageHeader
        title="Transferencias entre contas"
        description="Deposito e resgate entre conta corrente e aplicacao (movimentacao interna)"
        actions={
          canWrite ? (
            <Button onClick={() => openNew()}>
              <Plus className="h-4 w-4" /> Nova transferencia
            </Button>
          ) : undefined
        }
      />

      <div className="mb-4 grid gap-4 md:grid-cols-4">
        {(balances?.by_account_type ?? []).map((b) => (
          <Card key={b.account_type}>
            <p className="text-sm text-zinc-500">{ACCOUNT_TYPES[b.account_type] ?? b.account_type}</p>
            <p className={`mt-1 text-2xl font-semibold ${b.net >= 0 ? "text-sky-700" : "text-red-600"}`}>{currency(b.net)}</p>
            <p className="mt-1 text-xs text-zinc-400">Saldo atual das contas deste tipo</p>
          </Card>
        ))}
        {balances && (
          <Card>
            <p className="text-sm text-zinc-500">Total geral</p>
            <p className={`mt-1 text-2xl font-semibold ${balances.total >= 0 ? "text-emerald-600" : "text-red-600"}`}>{currency(balances.total)}</p>
            <p className="mt-1 text-xs text-zinc-400">Transferencias internas se anulam</p>
          </Card>
        )}
      </div>

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-4">
        <div>
          <label className="label">De</label>
          <Input type="date" className="h-8 text-sm" value={from} onChange={(e) => setFrom(e.target.value)} />
        </div>
        <div>
          <label className="label">Ate</label>
          <Input type="date" className="h-8 text-sm" value={to} onChange={(e) => setTo(e.target.value)} />
        </div>
        <div className="ml-auto flex flex-wrap gap-2">
          {canWrite && accounts.some((a) => a.account_type === "investment") && (
            <>
              <Button variant="outline" size="sm" onClick={() => openNew("deposit")}>
                <ArrowRight className="h-4 w-4" /> Depositar
              </Button>
              <Button variant="outline" size="sm" onClick={() => openNew("redeem")}>
                <ArrowLeftRight className="h-4 w-4" /> Resgatar
              </Button>
            </>
          )}
        </div>
      </Card>

      <DataTable
        columns={columns}
        data={transfers ?? []}
        keyExtractor={(t) => t.id}
        loading={loading}
        emptyIcon={<ArrowLeftRight className="h-10 w-10" />}
        emptyMessage="Nenhuma transferencia"
        emptyDescription="Registre um deposito ou resgate entre contas."
        maxHeight="max(260px, calc(100vh - 380px))"
        compact
      />

      <Modal open={open} onClose={() => { if (!saving) setOpen(false); }} title="Transferencia entre contas">
        <form onSubmit={submit} className="space-y-3 text-sm">
          <Field label="Operacao *">
            <Select value={form.kind} onChange={(e) => changeKind(e.target.value as Kind)}>
              <option value="deposit">Deposito (corrente/caixa para aplicacao)</option>
              <option value="redeem">Resgate (aplicacao para corrente/caixa)</option>
              <option value="transfer">Transferencia entre contas</option>
            </Select>
          </Field>
          <Field label="Conta de origem *">
            <Select required value={form.from_account_id} onChange={(e) => setForm({ ...form, from_account_id: e.target.value })}>
              <option value="">Selecione...</option>
              {fromOptions.map((a) => (
                <option key={a.id} value={a.id}>{a.name} - {currency(a.balance)}</option>
              ))}
            </Select>
          </Field>
          <Field label="Conta de destino *">
            <Select required value={form.to_account_id} onChange={(e) => setForm({ ...form, to_account_id: e.target.value })}>
              <option value="">Selecione...</option>
              {toOptions.map((a) => (
                <option key={a.id} value={a.id}>{a.name} - {currency(a.balance)}</option>
              ))}
            </Select>
            {toOptions.length === 0 && (
              <p className="mt-1 text-xs text-amber-600">
                Nenhuma conta compativel na mesma filial. Selecione a filial na topbar ao cadastrar a conta de aplicacao.
              </p>
            )}
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Valor (R$) *">
              <CurrencyInput required value={form.amount} onChange={(v) => setForm({ ...form, amount: v })} />
            </Field>
            <Field label="Data *">
              <Input type="date" required value={form.occurred_at} onChange={(e) => setForm({ ...form, occurred_at: e.target.value })} />
            </Field>
          </div>
          <Field label="Descricao">
            <Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Opcional" />
          </Field>

          {fromAccount && toAccount && amount > 0 && (
            <div className="rounded border border-zinc-200 px-3 py-2 text-xs text-zinc-600 dark:border-zinc-700 dark:text-zinc-300">
              <p>
                Saldo apos: <b>{fromAccount.name}</b>{" "}
                {currency(fromAccount.balance - amount)} · <b>{toAccount.name}</b>{" "}
                {currency(toAccount.balance + amount)}
              </p>
            </div>
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)} disabled={saving}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Transferindo..." : "Registrar transferencia"}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
