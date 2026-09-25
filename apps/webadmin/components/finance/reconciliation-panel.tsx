"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, Search, Lock, ArrowLeft, Trash2, CheckCircle2 } from "lucide-react";
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
  conciliateReconciliation, deleteReconciliation,
  type FinancialReconciliation, type ReconciliationItem,
} from "@/lib/api";
import { currency, datePt, dateTimePt, todayISO } from "@/lib/format";

export function ReconciliationPanel() {
  const { toast } = useToast();
  const { hasPerm, user } = useAuth();
  const canReconcile = hasPerm("finance.reconcile");
  const isHQ = user?.role === "super_admin" || user?.role === "admin_sede";

  const [list, setList] = useState<FinancialReconciliation[] | null>(null);
  const [selected, setSelected] = useState<FinancialReconciliation | null>(null);
  const [items, setItems] = useState<ReconciliationItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);
  const [query, setQuery] = useState("");

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
          {r.status === "conciliada" ? "Conciliada (travada)" : "Aberta"}
        </Badge>
      ),
    },
    { key: "total_items", label: "Lancamentos", sortable: true, align: "right", width: "w-28", render: (r) => <span className="text-zinc-500">{r.total_items}</span> },
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

  const itemColumns: Column<ReconciliationItem>[] = [
    { key: "occurred_at", label: "Data", sortable: true, width: "w-28", render: (it) => <span className="text-zinc-500">{datePt(it.occurred_at)}</span> },
    { key: "type", label: "Tipo", sortable: true, width: "w-24", render: (it) => <Badge tone={it.type === "income" ? "green" : "red"}>{it.type === "income" ? "Entrada" : "Saida"}</Badge> },
    { key: "category_name", label: "Conta contabil", sortable: true, render: (it) => <span className="text-zinc-500">{it.category_name ?? "-"}</span> },
    { key: "description", label: "Descricao", sortable: true, render: (it) => <span className="text-zinc-500">{it.description ?? "-"}</span> },
    {
      key: "amount",
      label: "Valor",
      sortable: true,
      align: "right",
      render: (it) => <span className={`font-medium ${it.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(it.amount)}</span>,
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
          <Badge tone={locked ? "green" : "amber"}>{locked ? "Conciliada (travada)" : "Aberta"}</Badge>
          <span className="text-sm text-zinc-500">
            {selected.total_items} lancamento(s) - Entradas {currency(selected.total_income)} - Saidas {currency(selected.total_expense)} - Saldo {currency(net)}
          </span>
          {canReconcile && !locked && (
            <Button size="sm" className="ml-auto" onClick={() => setConfirmConciliate(true)}>
              <Lock className="h-4 w-4" /> Conciliar e travar periodo
            </Button>
          )}
        </div>

        {locked && (
          <Card className="mb-4 text-sm">
            <p className="flex items-center gap-2 font-medium text-emerald-700">
              <CheckCircle2 className="h-4 w-4" /> Periodo conciliado e travado
            </p>
            <p className="mt-1 text-zinc-500">
              Nenhum lancamento pode ser criado, alterado, estornado ou excluido neste periodo
              {selected.reconciled_at ? ` (conciliado em ${dateTimePt(selected.reconciled_at)})` : ""}.
            </p>
          </Card>
        )}

        <DataTable
          columns={itemColumns}
          data={items}
          keyExtractor={(it) => it.id}
          loading={loadingItems}
          emptyMessage="Sem lancamentos"
          emptyDescription="Nenhum lancamento no periodo desta conciliacao."
          compact
        />

        <Modal open={confirmConciliate} onClose={() => setConfirmConciliate(false)} title="Conciliar e travar periodo">
          <div className="space-y-3 text-sm">
            <p className="text-zinc-600">
              Ao conciliar, o periodo <strong>{datePt(selected.period_start)} a {datePt(selected.period_end)}</strong> sera
              <strong> travado</strong>: nao sera possivel lancar, editar, estornar ou excluir lancamentos nesse intervalo.
              Apenas a Sede podera excluir esta conciliacao para destravar.
            </p>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setConfirmConciliate(false)}>Cancelar</Button>
              <Button onClick={doConciliate} disabled={conciliating}>
                {conciliating ? "Conciliando..." : "Conciliar e travar"}
              </Button>
            </div>
          </div>
        </Modal>
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
