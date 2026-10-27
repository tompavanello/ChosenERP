"use client";

import { useEffect, useMemo, useState } from "react";
import { Layers, Link2, Lock, RotateCcw, Trash2, Unlink } from "lucide-react";
import { Drawer, Section } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Field, Input } from "@/components/ui/input";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { useToast } from "@/components/ui/toast";
import {
  createTransactionGroup,
  deleteTransactionGroup,
  getTransactionGroup,
  reconcileTransactionGroup,
  removeTransactionGroupMember,
  updateTransactionGroup,
  type Transaction,
  type TransactionGroup,
} from "@/lib/api";
import { currency, datePt, todayISO } from "@/lib/format";

type SelectionDrawerProps = {
  open: boolean;
  onClose: () => void;
  transactions: Transaction[];
  onSaved: () => void;
};

/** Cria um grupo a partir dos lancamentos selecionados na lista. */
export function GroupSelectionDrawer({ open, onClose, transactions, onSaved }: SelectionDrawerProps) {
  const { toast } = useToast();
  const [occurredAt, setOccurredAt] = useState(todayISO());
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    setOccurredAt(transactions[0]?.occurred_at?.slice(0, 10) ?? todayISO());
    setDescription("");
  }, [open, transactions]);

  const total = useMemo(
    () => transactions.reduce((sum, t) => sum + Number(t.amount || 0), 0),
    [transactions],
  );
  const mixed = useMemo(() => {
    if (transactions.length < 2) return false;
    const first = transactions[0];
    return transactions.some(
      (t) => t.type !== first.type || t.account_id !== first.account_id || t.branch_id !== first.branch_id,
    );
  }, [transactions]);
  const hasReconciled = transactions.some((t) => t.reconciled);
  const hasGrouped = transactions.some((t) => t.group_id);

  async function save() {
    if (transactions.length < 2) {
      toast("Selecione pelo menos 2 lancamentos para agrupar.", "error");
      return;
    }
    if (mixed) {
      toast("Os lancamentos precisam ter a mesma conta bancaria e o mesmo tipo.", "error");
      return;
    }
    if (hasReconciled) {
      toast("Desconcilie os lancamentos antes de agrupar.", "error");
      return;
    }
    if (hasGrouped) {
      toast("Um dos lancamentos ja pertence a outro grupo.", "error");
      return;
    }
    setSaving(true);
    try {
      await createTransactionGroup({
        transaction_ids: transactions.map((t) => t.id),
        occurred_at: occurredAt,
        description: description.trim() || undefined,
      });
      toast("Lancamentos agrupados.", "success");
      onSaved();
      onClose();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao agrupar lancamentos", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Drawer open={open} onClose={onClose} title="Agrupar lancamentos" size="lg">
      <div className="space-y-4">
        <div className="rounded-lg border border-sky-100 bg-sky-50 p-3 text-sm text-sky-800">
          <div className="flex items-center gap-2 font-medium">
            <Layers className="h-4 w-4" /> Uma linha no extrato, varios detalhes contabeis
          </div>
          <p className="mt-1 text-xs text-sky-700">
            Os lancamentos continuam individuais (categoria, doador e recibo). O grupo representa
            apenas a movimentacao bancaria unica.
          </p>
        </div>

        <Section title="Dados do grupo" hint="A data e a descricao aparecem na visao banco do extrato.">
          <Field label="Data do deposito">
            <Input type="date" value={occurredAt} onChange={(e) => setOccurredAt(e.target.value)} />
          </Field>
          <Field label="Descricao">
            <Input
              placeholder="Ex.: Deposito referente a dizimos e ofertas"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>
        </Section>

        <div className="flex items-center justify-between rounded-lg border border-zinc-200 px-3 py-2 text-sm">
          <span className="text-zinc-500">{transactions.length} lancamento(s)</span>
          <span className="font-semibold text-emerald-600">{currency(total)}</span>
        </div>

        {mixed && (
          <p className="text-xs text-red-600">
            Ha lancamentos com contas bancarias ou tipos diferentes. Ajuste a selecao.
          </p>
        )}
        {hasReconciled && (
          <p className="text-xs text-amber-600">Ha lancamento conciliado na selecao. Desconcilie antes de agrupar.</p>
        )}
        {hasGrouped && <p className="text-xs text-amber-600">Ha lancamento ja agrupado na selecao.</p>}

        <div className="max-h-64 overflow-y-auto rounded-lg border border-zinc-200">
          <Table>
            <THead>
              <TRow>
                <TH>Data</TH>
                <TH>Conta contabil</TH>
                <TH>Doador</TH>
                <TH className="text-right">Valor</TH>
              </TRow>
            </THead>
            <TBody>
              {transactions.map((t) => (
                <TRow key={t.id}>
                  <TD className="text-zinc-500">{datePt(t.occurred_at)}</TD>
                  <TD>{t.category_name ?? "-"}</TD>
                  <TD className="text-zinc-500">{t.is_anonymous ? "Anonimo" : (t.donor_name ?? "-")}</TD>
                  <TD className="text-right font-medium">{currency(t.amount)}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose} disabled={saving}>Cancelar</Button>
          <Button onClick={save} disabled={saving}>
            <Link2 className="h-4 w-4" /> {saving ? "Agrupando..." : "Agrupar"}
          </Button>
        </div>
      </div>
    </Drawer>
  );
}

type DetailDrawerProps = {
  open: boolean;
  groupId?: string;
  onClose: () => void;
  onChanged: () => void;
  canWrite: boolean;
  canReconcile: boolean;
};

/** Detalhe do grupo: linha bancaria unica + membros e acoes. */
export function GroupDetailDrawer({ open, groupId, onClose, onChanged, canWrite, canReconcile }: DetailDrawerProps) {
  const { toast } = useToast();
  const [group, setGroup] = useState<TransactionGroup | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [occurredAt, setOccurredAt] = useState("");
  const [description, setDescription] = useState("");

  async function load() {
    if (!groupId) return;
    setLoading(true);
    try {
      const g = await getTransactionGroup(groupId);
      setGroup(g);
      setOccurredAt(g.occurred_at?.slice(0, 10) ?? "");
      setDescription(g.description ?? "");
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao carregar grupo", "error");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (open && groupId) load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, groupId]);

  async function saveHeader() {
    if (!group) return;
    setSaving(true);
    try {
      const g = await updateTransactionGroup(group.id, { occurred_at: occurredAt, description });
      setGroup(g);
      toast("Grupo atualizado.", "success");
      onChanged();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao atualizar grupo", "error");
    } finally {
      setSaving(false);
    }
  }

  async function toggleReconciled() {
    if (!group) return;
    setSaving(true);
    try {
      const g = await reconcileTransactionGroup(group.id, !group.reconciled);
      setGroup(g);
      toast(g.reconciled ? "Grupo conciliado." : "Grupo desconcliado.", "success");
      onChanged();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao conciliar grupo", "error");
    } finally {
      setSaving(false);
    }
  }

  async function ungroup() {
    if (!group) return;
    if (!window.confirm("Desagrupar todos os lancamentos deste grupo?")) return;
    setSaving(true);
    try {
      await deleteTransactionGroup(group.id);
      toast("Lancamentos desagrupados.", "success");
      onChanged();
      onClose();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao desagrupar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function removeMember(transactionId: string) {
    if (!group) return;
    setSaving(true);
    try {
      const g = await removeTransactionGroupMember(group.id, transactionId);
      setGroup(g);
      toast("Lancamento removido do grupo.", "success");
      onChanged();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Falha ao remover lancamento", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Drawer open={open} onClose={onClose} title="Grupo de lancamentos" size="xl">
      {loading && <p className="text-sm text-zinc-500">Carregando...</p>}
      {!loading && !group && <p className="text-sm text-zinc-500">Grupo nao encontrado.</p>}
      {group && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-zinc-200 p-3">
            <div>
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4 text-sky-600" />
                <span className="text-sm font-semibold text-zinc-700">
                  {group.member_count} lancamento(s) em uma linha bancaria
                </span>
                {group.reconciled ? (
                  <Badge tone="green" className="gap-1"><Lock className="h-3 w-3" /> Conciliado</Badge>
                ) : (
                  <Badge tone="indigo">Agrupado</Badge>
                )}
              </div>
              <p className="mt-1 text-xs text-zinc-500">
                {group.account_name ?? "Conta bancaria"} | {datePt(group.occurred_at)}
              </p>
            </div>
            <div className="text-right">
              <p className="text-xs text-zinc-400">Total do grupo</p>
              <p className={`text-xl font-semibold ${group.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                {currency(group.total_amount)}
              </p>
            </div>
          </div>

          <Section title="Dados do grupo" hint="A data e a descricao valem para a linha do extrato bancario.">
            <Field label="Data do deposito">
              <Input type="date" value={occurredAt} onChange={(e) => setOccurredAt(e.target.value)} disabled={!canWrite || group.reconciled} />
            </Field>
            <Field label="Descricao">
              <Input value={description} onChange={(e) => setDescription(e.target.value)} disabled={!canWrite || group.reconciled} />
            </Field>
          </Section>

          <div className="max-h-80 overflow-y-auto rounded-lg border border-zinc-200">
            <Table>
              <THead>
                <TRow>
                  <TH>Data</TH>
                  <TH>Conta contabil</TH>
                  <TH>Doador</TH>
                  <TH>Descricao</TH>
                  <TH className="text-right">Valor</TH>
                  <TH></TH>
                </TRow>
              </THead>
              <TBody>
                {(group.members ?? []).map((m) => (
                  <TRow key={m.id}>
                    <TD className="whitespace-nowrap text-zinc-500">{datePt(m.occurred_at)}</TD>
                    <TD>{m.category_name ?? "-"}</TD>
                    <TD className="text-zinc-500">{m.donor_name ?? "-"}</TD>
                    <TD className="text-zinc-500">{m.description ?? "-"}</TD>
                    <TD className="text-right font-medium">{currency(m.amount)}</TD>
                    <TD className="text-right">
                      {canWrite && !group.reconciled && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => removeMember(m.id)}
                          disabled={saving}
                          title="Remover do grupo"
                          aria-label="Remover do grupo"
                        >
                          <Unlink className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          </div>

          <div className="flex flex-wrap justify-end gap-2">
            {canWrite && !group.reconciled && (
              <Button variant="outline" onClick={saveHeader} disabled={saving}>Salvar dados</Button>
            )}
            {canReconcile && (
              <Button variant={group.reconciled ? "outline" : "primary"} onClick={toggleReconciled} disabled={saving}>
                {group.reconciled ? <RotateCcw className="h-4 w-4" /> : <Lock className="h-4 w-4" />}
                {group.reconciled ? "Desconciliar" : "Conciliar grupo"}
              </Button>
            )}
            {canWrite && !group.reconciled && (
              <Button variant="outline" className="text-red-600" onClick={ungroup} disabled={saving}>
                <Trash2 className="h-4 w-4" /> Desagrupar
              </Button>
            )}
          </div>
        </div>
      )}
    </Drawer>
  );
}
