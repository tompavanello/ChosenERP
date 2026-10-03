"use client";

import { useEffect, useState } from "react";
import { Plus, ArrowRight, Wallet, Percent, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listBranches, listTransfers, createTransfer, getSplit, updateSplit,
  type Transfer, type Branch,
} from "@/lib/api";
import { currency, dateTimePt } from "@/lib/format";

type RuleForm = { destination_branch_id: string; name: string; percent: string; is_active: boolean };

export default function TransfersPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("finance.write");

  const [transfers, setTransfers] = useState<Transfer[] | null>(null);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ from_branch_id: "", to_branch_id: "", amount: "", rule_name: "" });

  const [splitEnabled, setSplitEnabled] = useState(false);
  const [splitRules, setSplitRules] = useState<RuleForm[]>([]);
  const [splitSaving, setSplitSaving] = useState(false);

  useEffect(() => {
    listTransfers().then((r) => setTransfers(r.transfers)).catch((e) => toast(e.message, "error"));
    listBranches().then((r) => setBranches(r.branches)).catch(() => {});
    getSplit()
      .then((r) => {
        setSplitEnabled(r.enabled);
        setSplitRules(r.rules.map((x) => ({
          destination_branch_id: x.destination_branch_id,
          name: x.name ?? "",
          percent: String(x.percent),
          is_active: x.is_active,
        })));
      })
      .catch(() => {});
  }, [toast]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createTransfer({ ...form, amount: Number(form.amount) });
      toast("Repasse registrado.");
      setOpen(false);
      setForm({ from_branch_id: "", to_branch_id: "", amount: "", rule_name: "" });
      setTransfers(await listTransfers().then((r) => r.transfers));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  function addRule() {
    setSplitRules([...splitRules, { destination_branch_id: "", name: "", percent: "", is_active: true }]);
  }
  function updRule(i: number, patch: Partial<RuleForm>) {
    setSplitRules(splitRules.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }
  function removeRule(i: number) {
    setSplitRules(splitRules.filter((_, idx) => idx !== i));
  }
  async function saveSplit() {
    setSplitSaving(true);
    try {
      await updateSplit(
        splitEnabled,
        splitRules
          .filter((r) => r.destination_branch_id)
          .map((r) => ({
            destination_branch_id: r.destination_branch_id,
            name: r.name || undefined,
            percent: Number(r.percent) || 0,
            is_active: r.is_active,
          })),
      );
      toast("Split salvo.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSplitSaving(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Repasses"
        description="Transferencia de recursos entre filiais (splits)"
        actions={<Button onClick={() => setOpen(true)}><Plus className="h-4 w-4" /> Novo Repasse</Button>}
      />

      <Card className="mb-4 p-4">
        <div className="mb-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Percent className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Split automático de repasses</h3>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={splitEnabled}
              onChange={(e) => setSplitEnabled(e.target.checked)}
              disabled={!canWrite}
              className="h-4 w-4 rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
            />
            Ativado
          </label>
        </div>
        <p className="mb-3 text-xs text-zinc-400">
          Ao lançar uma <strong>entrada</strong>, o percentual de cada regra ativa é transferido automaticamente para a filial de destino.
        </p>

        <div className="space-y-2">
          {splitRules.map((r, i) => (
            <div key={i} className="flex flex-wrap items-center gap-2">
              <Select
                className="h-8 flex-1 text-sm"
                value={r.destination_branch_id}
                onChange={(e) => updRule(i, { destination_branch_id: e.target.value })}
                disabled={!canWrite}
              >
                <option value="">Filial de destino...</option>
                {branches.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
              </Select>
              <div className="flex items-center gap-1">
                <Input
                  type="number"
                  min={0}
                  max={100}
                  step="0.01"
                  className="h-8 w-24 text-sm"
                  value={r.percent}
                  onChange={(e) => updRule(i, { percent: e.target.value })}
                  disabled={!canWrite}
                />
                <span className="text-sm text-zinc-500">%</span>
              </div>
              <Input
                className="h-8 w-40 text-sm"
                placeholder="Nome (opcional)"
                value={r.name}
                onChange={(e) => updRule(i, { name: e.target.value })}
                disabled={!canWrite}
              />
              <label className="flex items-center gap-1 text-xs text-zinc-500">
                <input
                  type="checkbox"
                  checked={r.is_active}
                  onChange={(e) => updRule(i, { is_active: e.target.checked })}
                  disabled={!canWrite}
                  className="h-4 w-4 rounded border-zinc-300 text-sky-600 focus:ring-sky-500"
                />
                ativo
              </label>
              {canWrite && (
                <Button variant="ghost" size="sm" onClick={() => removeRule(i)} title="Remover">
                  <Trash2 className="h-3.5 w-3.5 text-red-500" />
                </Button>
              )}
            </div>
          ))}
          {splitRules.length === 0 && (
            <p className="text-xs text-zinc-400">Nenhuma regra. Adicione uma filial de destino e o percentual.</p>
          )}
        </div>

        {canWrite && (
          <div className="mt-3 flex justify-end gap-2">
            <Button variant="outline" className="h-8 text-sm" onClick={addRule}><Plus className="h-3.5 w-3.5" /> Adicionar regra</Button>
            <Button className="h-8 text-sm" onClick={saveSplit} disabled={splitSaving}>{splitSaving ? "Salvando..." : "Salvar split"}</Button>
          </div>
        )}
      </Card>

      <Card className="overflow-hidden p-0">
        {transfers === null ? (
          <SkeletonRows />
        ) : transfers.length === 0 ? (
          <EmptyState icon={<Wallet className="h-10 w-10" />} title="Nenhum repasse" description="Registre a transferencia de recursos entre filiais." />
        ) : (
          <Table>
            <THead><TRow><TH>Origem</TH><TH>Destino</TH><TH>Regra</TH><TH className="text-right">Valor</TH><TH className="text-right">Data</TH></TRow></THead>
            <TBody>
              {transfers.map((t) => (
                <TRow key={t.id}>
                  <TD className="font-medium">{t.from_branch}</TD>
                  <TD><span className="inline-flex items-center gap-1 text-zinc-500"><ArrowRight className="h-3.5 w-3.5" /> {t.to_branch}</span></TD>
                  <TD><Badge tone="sky">{t.rule_name || "Repasse"}</Badge></TD>
                  <TD className="text-right font-semibold text-sky-700">{currency(t.amount)}</TD>
                  <TD className="text-right text-zinc-500">{dateTimePt(t.executed_at)}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Modal open={open} onClose={() => setOpen(false)} title="Novo Repasse">
        <form onSubmit={submit} className="space-y-3">
          <Field label="Filial de origem *">
            <Select required value={form.from_branch_id} onChange={(e) => setForm({ ...form, from_branch_id: e.target.value })}>
              <option value="">Selecione...</option>
              {branches.map((b) => (<option key={b.id} value={b.id}>{b.name}</option>))}
            </Select>
          </Field>
          <Field label="Filial de destino *">
            <Select required value={form.to_branch_id} onChange={(e) => setForm({ ...form, to_branch_id: e.target.value })}>
              <option value="">Selecione...</option>
              {branches.map((b) => (<option key={b.id} value={b.id}>{b.name}</option>))}
            </Select>
          </Field>
          <Field label="Valor (R$) *"><CurrencyInput required value={form.amount} onChange={(v) => setForm({ ...form, amount: v })} /></Field>
          <Field label="Regra (opcional)"><Input value={form.rule_name} placeholder="ex.: 10% sede" onChange={(e) => setForm({ ...form, rule_name: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Registrar</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
