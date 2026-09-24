"use client";

import { useEffect, useState } from "react";
import { Plus, ArrowRight, Wallet } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { listBranches, listTransfers, createTransfer, type Transfer, type Branch } from "@/lib/api";
import { currency, dateTimePt } from "@/lib/format";

export default function TransfersPage() {
  const { toast } = useToast();
  const [transfers, setTransfers] = useState<Transfer[] | null>(null);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ from_branch_id: "", to_branch_id: "", amount: "", rule_name: "" });

  useEffect(() => {
    listTransfers().then((r) => setTransfers(r.transfers)).catch((e) => toast(e.message, "error"));
    listBranches().then((r) => setBranches(r.branches)).catch(() => {});
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

  return (
    <div className="page">
      <PageHeader
        title="Repasses"
        description="Transferencia de recursos entre filiais (splits)"
        actions={<Button onClick={() => setOpen(true)}><Plus className="h-4 w-4" /> Novo Repasse</Button>}
      />

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
          <Field label="Valor (R$) *"><Input required type="number" min="0" step="0.01" value={form.amount} onChange={(e) => setForm({ ...form, amount: e.target.value })} /></Field>
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
