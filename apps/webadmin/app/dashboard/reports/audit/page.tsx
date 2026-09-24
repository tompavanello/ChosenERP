"use client";

import { useCallback, useEffect, useState } from "react";
import { ShieldCheck, Plus, Search, Paperclip, Lock, ArrowLeft, CheckCircle2, Eye, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Input, Field } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { ExportButtons } from "@/components/reports/export-buttons";
import {
  listAudits, createAudit, getAudit, markAudit, closeAudit, deleteAudit, listAttachments,
  type FinancialAudit, type AuditItem, type FinancialAttachment,
} from "@/lib/api";
import { PAYMENT_METHODS } from "@/lib/constants";
import { currency, datePt, dateTimePt } from "@/lib/format";

function isoDate(d: Date) {
  return d.toISOString().slice(0, 10);
}

export default function AuditReportPage() {
  const { toast } = useToast();
  const [audits, setAudits] = useState<FinancialAudit[] | null>(null);
  const [selected, setSelected] = useState<FinancialAudit | null>(null);
  const [items, setItems] = useState<AuditItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);

  const [showNew, setShowNew] = useState(false);
  const now = new Date();
  const [form, setForm] = useState({
    title: "Auditoria financeira",
    period_start: isoDate(new Date(now.getFullYear(), now.getMonth(), 1)),
    period_end: isoDate(now),
  });
  const [saving, setSaving] = useState(false);

  const [closeOpen, setCloseOpen] = useState(false);
  const [signerName, setSignerName] = useState("");
  const [signerRole, setSignerRole] = useState("");
  const [closing, setClosing] = useState(false);

  const [detailItem, setDetailItem] = useState<AuditItem | null>(null);
  const [detailAtts, setDetailAtts] = useState<FinancialAttachment[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);

  const loadAudits = useCallback(async () => {
    try {
      setAudits(await listAudits().then((r) => r.audits));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }, [toast]);

  useEffect(() => { loadAudits(); }, [loadAudits]);

  async function openAudit(a: FinancialAudit) {
    setSelected(a);
    setLoadingItems(true);
    try {
      const res = await getAudit(a.id);
      setSelected(res.audit);
      setItems(res.items);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoadingItems(false);
    }
  }

  async function reloadItems(id: string) {
    const res = await getAudit(id);
    setSelected(res.audit);
    setItems(res.items);
  }

  async function removeAudit(a: FinancialAudit) {
    if (!confirm(`Excluir a auditoria "${a.title}"? Esta ação remove a auditoria e suas marcações.`)) return;
    try {
      await deleteAudit(a.id);
      toast("Auditoria excluída.");
      setSelected(null);
      await loadAudits();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  async function createNew(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const a = await createAudit(form);
      toast("Auditoria criada.");
      setShowNew(false);
      await loadAudits();
      await openAudit(a);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  async function toggleOne(it: AuditItem) {
    if (!selected) return;
    try {
      await markAudit(selected.id, { transaction_ids: [it.transaction_id], audited: !it.audited });
      await reloadItems(selected.id);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function markAll(audited: boolean) {
    if (!selected) return;
    try {
      await markAudit(selected.id, { transaction_ids: [], audited });
      await reloadItems(selected.id);
      toast(audited ? "Todos os lançamentos marcados como auditados." : "Marcações removidas.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function doClose(e: React.FormEvent) {
    e.preventDefault();
    if (!selected) return;
    setClosing(true);
    try {
      const a = await closeAudit(selected.id, { signer_name: signerName, signer_role: signerRole });
      setSelected(a);
      setCloseOpen(false);
      setSignerName("");
      setSignerRole("");
      await loadAudits();
      toast("Auditoria fechada e assinada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setClosing(false);
    }
  }

  async function openDetail(it: AuditItem) {
    setDetailItem(it);
    setDetailLoading(true);
    setDetailAtts([]);
    try {
      const res = await listAttachments(it.transaction_id);
      setDetailAtts(res.attachments);
    } catch {
      setDetailAtts([]);
    } finally {
      setDetailLoading(false);
    }
  }

  const isOpen = selected?.status === "aberta";
  const auditedCount = items.filter((i) => i.audited).length;

  if (selected) {
    return (
      <div className="page">
        <PageHeader
          title={selected.title}
          description={`Período: ${datePt(selected.period_start)} a ${datePt(selected.period_end)}`}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => { setSelected(null); loadAudits(); }}>
                <ArrowLeft className="h-4 w-4" /> Voltar
              </Button>
              <Button variant="ghost" onClick={() => removeAudit(selected)} title="Excluir auditoria">
                <Trash2 className="h-4 w-4 text-red-500" /> Excluir
              </Button>
              <ExportButtons
                path={`/api/v1/finance/audits/${selected.id}/export`}
                filenameBase={`auditoria-${selected.period_start}-${selected.period_end}`}
              />
            </div>
          }
        />

        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Badge tone={isOpen ? "amber" : "green"}>{isOpen ? "Aberta" : "Fechada"}</Badge>
          <span className="text-sm text-zinc-500">
            Auditados {auditedCount}/{items.length} · Total {currency(selected.total_amount)} · Auditado {currency(selected.audited_amount)}
          </span>
          {isOpen && (
            <div className="ml-auto flex flex-wrap gap-2">
              <Button size="sm" variant="outline" onClick={() => markAll(true)}>
                <CheckCircle2 className="h-4 w-4" /> Marcar todos
              </Button>
              <Button size="sm" variant="ghost" onClick={() => markAll(false)}>Limpar marcações</Button>
              <Button size="sm" onClick={() => setCloseOpen(true)}>
                <Lock className="h-4 w-4" /> Fechar e assinar
              </Button>
            </div>
          )}
        </div>

        {!isOpen && selected.signer_name && (
          <Card className="mb-4 text-sm">
            <p className="font-medium">Documento auditado (imutável)</p>
            <p className="mt-1 text-zinc-500">
              Responsável: <strong>{selected.signer_name}</strong>
              {selected.signer_role ? ` (${selected.signer_role})` : ""}
              {selected.closed_at ? ` · Fechado em ${dateTimePt(selected.closed_at)}` : ""}
            </p>
            {selected.signature_hash && (
              <p className="mt-1 break-all text-xs text-zinc-400">Hash: {selected.signature_hash}</p>
            )}
          </Card>
        )}

        <Card className="overflow-hidden p-0">
          {loadingItems ? (
            <SkeletonRows />
          ) : items.length === 0 ? (
            <EmptyState icon={<ShieldCheck className="h-10 w-10" />} title="Sem lançamentos" description="Nenhum lançamento no período desta auditoria." />
          ) : (
            <Table>
              <THead>
                <TRow>
                  <TH className="w-16">Auditado</TH>
                  <TH>Data</TH>
                  <TH>Tipo</TH>
                  <TH>Conta contábil</TH>
                  <TH>Descrição</TH>
                  <TH className="text-right">Valor</TH>
                  <TH className="w-16"></TH>
                </TRow>
              </THead>
              <TBody>
                {items.map((it) => (
                  <TRow key={it.transaction_id}>
                    <TD>
                      <input
                        type="checkbox"
                        className="h-4 w-4 rounded border-zinc-300 text-sky-600"
                        checked={it.audited}
                        disabled={!isOpen}
                        onChange={() => toggleOne(it)}
                      />
                    </TD>
                    <TD className="text-zinc-500">{datePt(it.occurred_at)}</TD>
                    <TD>
                      <Badge tone={it.type === "income" ? "green" : "red"}>
                        {it.type === "income" ? "Entrada" : "Saída"}
                      </Badge>
                    </TD>
                    <TD className="text-zinc-500">{it.category_name ?? "—"}</TD>
                    <TD className="text-zinc-500">{it.description ?? "—"}</TD>
                    <TD className={`text-right font-medium ${it.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(it.amount)}</TD>
                    <TD className="text-center">
                      <Button variant="ghost" className="h-7 px-2 text-xs" title="Detalhes do lançamento" onClick={() => openDetail(it)}>
                        <Eye className="h-4 w-4" />
                        {it.attachment_count > 0 && <span className="ml-0.5 text-[10px]">{it.attachment_count}</span>}
                      </Button>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>

        <Modal open={closeOpen} onClose={() => setCloseOpen(false)} title="Fechar e assinar auditoria">
          <form onSubmit={doClose} className="space-y-3 text-sm">
            <p className="text-zinc-500">
              Ao fechar, a auditoria vira um documento imutável: não será possível marcar/desmarcar
              lançamentos nem estornar os lançamentos auditados.
            </p>
            <Field label="Nome do responsável *">
              <Input required value={signerName} onChange={(e) => setSignerName(e.target.value)} />
            </Field>
            <Field label="Função / cargo">
              <Input value={signerRole} onChange={(e) => setSignerRole(e.target.value)} placeholder="Ex.: Tesoureiro(a)" />
            </Field>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setCloseOpen(false)}>Cancelar</Button>
              <Button type="submit" disabled={closing}>{closing ? "Fechando..." : "Fechar e assinar"}</Button>
            </div>
          </form>
        </Modal>

        <Drawer open={!!detailItem} onClose={() => setDetailItem(null)} title="Detalhes do lançamento">
          {detailItem && (
            <div className="space-y-3 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-zinc-500">Valor</span>
                <span className={`text-xl font-semibold ${detailItem.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                  {detailItem.type === "income" ? "+" : "-"}{currency(detailItem.amount)}
                </span>
              </div>
              <DetRow label="Tipo" value={detailItem.type === "income" ? "Entrada" : "Saída"} />
              <DetRow label="Data" value={dateTimePt(detailItem.occurred_at)} />
              <DetRow label="Conta contábil" value={detailItem.category_name ?? "—"} />
              <DetRow label="Conta bancária" value={detailItem.account_name ?? "—"} />
              <DetRow label="Fornecedor" value={detailItem.supplier_name ?? "—"} />
              <DetRow label="Forma de pagamento" value={detailItem.payment_method ? (PAYMENT_METHODS[detailItem.payment_method] ?? detailItem.payment_method) : "—"} />
              <DetRow label="Origem" value={detailItem.is_anonymous ? "Anônimo" : (detailItem.donor_name ?? "—")} />
              {detailItem.receipt_ref && <DetRow label="Recibo" value={detailItem.receipt_ref} />}
              <div>
                <span className="text-zinc-500">Descrição</span>
                <p className="mt-0.5 whitespace-pre-wrap">{detailItem.description ?? "—"}</p>
              </div>
              {detailItem.allocations.length > 0 && (
                <div>
                  <span className="text-zinc-500">Rateio por eventos</span>
                  <ul className="mt-1 divide-y divide-zinc-100 dark:divide-zinc-800">
                    {detailItem.allocations.map((al) => (
                      <li key={al.event_id} className="flex items-center justify-between py-1">
                        <span className="truncate">{al.event_name}</span>
                        <span className="tabular-nums text-zinc-500">{currency(al.amount)}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              <div>
                <span className="text-zinc-500">Anexos</span>
                {detailLoading ? (
                  <p className="mt-1 text-xs text-zinc-400">Carregando...</p>
                ) : detailAtts.length === 0 ? (
                  <p className="mt-1 text-xs text-zinc-400">Nenhum anexo.</p>
                ) : (
                  <div className="mt-1 space-y-2">
                    {detailAtts.map((att) => (
                      <a key={att.id} href={att.file_url} target="_blank" rel="noopener noreferrer"
                         className="flex items-center justify-between rounded border border-zinc-200 p-2 text-sm hover:bg-zinc-50 dark:border-zinc-700 dark:hover:bg-zinc-800">
                        <span className="flex items-center gap-2 truncate"><Paperclip className="h-4 w-4 text-zinc-500" /> {att.file_name}</span>
                        <span className="text-xs text-zinc-400">{(att.file_size / 1024).toFixed(1)} KB</span>
                      </a>
                    ))}
                  </div>
                )}
              </div>
              {detailItem.audited && (
                <p className="text-xs text-emerald-600">
                  Auditado{detailItem.audited_at ? ` em ${dateTimePt(detailItem.audited_at)}` : ""}.
                </p>
              )}
            </div>
          )}
        </Drawer>
      </div>
    );
  }

  return (
    <div className="page">
      <PageHeader
        title="Auditoria financeira"
        description="Lista lançamentos do período, permite marcar como auditado e fechar com assinatura"
        actions={<Button onClick={() => setShowNew(true)}><Plus className="h-4 w-4" /> Nova auditoria</Button>}
      />

      <Card className="overflow-hidden p-0">
        {audits === null ? (
          <SkeletonRows />
        ) : audits.length === 0 ? (
          <EmptyState icon={<ShieldCheck className="h-10 w-10" />} title="Nenhuma auditoria" description="Crie uma auditoria para um período e marque os lançamentos." />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Título</TH><TH>Período</TH><TH>Situação</TH><TH className="text-right">Auditados</TH><TH></TH></TRow>
            </THead>
            <TBody>
              {audits.map((a) => (
                <TRow key={a.id}>
                  <TD className="font-medium">{a.title}</TD>
                  <TD className="text-zinc-500">{datePt(a.period_start)} a {datePt(a.period_end)}</TD>
                  <TD><Badge tone={a.status === "aberta" ? "amber" : "green"}>{a.status === "aberta" ? "Aberta" : "Fechada"}</Badge></TD>
                  <TD className="text-right text-zinc-500">{a.audited_items}/{a.total_items}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="outline" onClick={() => openAudit(a)}>
                        <Search className="h-4 w-4" /> Abrir
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => removeAudit(a)} aria-label="Excluir auditoria" title="Excluir">
                        <Trash2 className="h-4 w-4 text-red-500" />
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={showNew} onClose={() => setShowNew(false)} title="Nova auditoria">
        <form onSubmit={createNew} className="space-y-3">
          <Field label="Título"><Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="Período — de *"><Input type="date" required value={form.period_start} onChange={(e) => setForm({ ...form, period_start: e.target.value })} /></Field>
          <Field label="Período — até *"><Input type="date" required value={form.period_end} onChange={(e) => setForm({ ...form, period_end: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowNew(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Criando..." : "Criar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}

function DetRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-zinc-500">{label}</span>
      <span className="text-right">{value}</span>
    </div>
  );
}
