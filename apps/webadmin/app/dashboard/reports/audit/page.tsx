"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ShieldCheck, Plus, Search, Paperclip, Lock, ArrowLeft, CheckCircle2, Eye, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Input, Field } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { ExportButtons } from "@/components/reports/export-buttons";
import {
  listAudits, createAudit, getAudit, markAudit, closeAudit, deleteAudit, listAttachments,
  type FinancialAudit, type AuditItem, type FinancialAttachment,
} from "@/lib/api";
import { PAYMENT_METHODS } from "@/lib/constants";
import { currency, datePt, dateTimePt, todayISO } from "@/lib/format";

export default function AuditReportPage() {
  const { toast } = useToast();
  const { hasPerm, user } = useAuth();
  const canAudit = hasPerm("finance.audit");
  const isHQ = user?.role === "super_admin" || user?.role === "admin_sede";
  const [audits, setAudits] = useState<FinancialAudit[] | null>(null);
  const [selected, setSelected] = useState<FinancialAudit | null>(null);
  const [items, setItems] = useState<AuditItem[]>([]);
  const [loadingItems, setLoadingItems] = useState(false);
  // Ordenacao/filtro do grid de lancamentos da auditoria.
  const [sortColumn, setSortColumn] = useState("occurred_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [itemSearch, setItemSearch] = useState("");
  const [itemFrom, setItemFrom] = useState("");
  const [itemTo, setItemTo] = useState("");
  // Ordenacao/filtro da lista de auditorias (grid principal da tela).
  const [auditSortColumn, setAuditSortColumn] = useState("period_start");
  const [auditSortDirection, setAuditSortDirection] = useState<"asc" | "desc">("desc");
  const [auditSearch, setAuditSearch] = useState("");
  const [auditFrom, setAuditFrom] = useState("");
  const [auditTo, setAuditTo] = useState("");

  const [showNew, setShowNew] = useState(false);
  const now = new Date();
  const [form, setForm] = useState({
    title: "Auditoria financeira",
    period_start: todayISO(new Date(now.getFullYear(), now.getMonth(), 1)),
    period_end: todayISO(now),
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
    setItemSearch("");
    setItemFrom("");
    setItemTo("");
    setSortColumn("occurred_at");
    setSortDirection("asc");
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
    if (!confirm(`Excluir a auditoria "${a.title}"? Esta acao remove a auditoria e suas marcacoes.`)) return;
    try {
      await deleteAudit(a.id);
      toast("Auditoria excluida.");
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
      toast(audited ? "Todos os lancamentos marcados como auditados." : "Marcacoes removidas.");
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

  // Filtro por descricao/conta/doador e por periodo (a data efetiva e
  // YYYY-MM-DD, entao a comparacao lexicografica equivale a comparacao de data).
  const filteredItems = useMemo(() => {
    const q = itemSearch.trim().toLowerCase();
    return items.filter((it) => {
      if (q) {
        const hay = [it.description, it.category_name, it.account_name, it.supplier_name, it.donor_name, it.receipt_ref]
          .filter(Boolean)
          .join(" ")
          .toLowerCase();
        if (!hay.includes(q)) return false;
      }
      if (itemFrom && it.occurred_at < itemFrom) return false;
      if (itemTo && it.occurred_at > itemTo) return false;
      return true;
    });
  }, [items, itemSearch, itemFrom, itemTo]);

  // Filtro por titulo e por periodo (auditorias que interceptam o intervalo).
  const filteredAudits = useMemo(() => {
    const q = auditSearch.trim().toLowerCase();
    return (audits ?? []).filter((a) => {
      if (q && !a.title.toLowerCase().includes(q) && !(a.signer_name ?? "").toLowerCase().includes(q)) return false;
      if (auditFrom && a.period_end < auditFrom) return false;
      if (auditTo && a.period_start > auditTo) return false;
      return true;
    });
  }, [audits, auditSearch, auditFrom, auditTo]);

  const handleAuditSort = (col: string) => {
    if (auditSortColumn === col) {
      setAuditSortDirection(auditSortDirection === "asc" ? "desc" : "asc");
    } else {
      setAuditSortColumn(col);
      setAuditSortDirection("asc");
    }
  };

  const handleItemSort = (col: string) => {
    if (sortColumn === col) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortColumn(col);
      setSortDirection("asc");
    }
  };

  const itemColumns: Column<AuditItem>[] = [
    {
      key: "audited",
      label: "Auditado",
      width: "w-20",
      render: (it) => (
        <input
          type="checkbox"
          className="h-4 w-4 rounded border-zinc-300 text-sky-600"
          checked={it.audited}
          disabled={!isOpen || !canAudit}
          onChange={() => toggleOne(it)}
        />
      ),
    },
    {
      key: "occurred_at",
      label: "Data",
      sortable: true,
      width: "w-28",
      render: (it) => <span className="text-zinc-500">{datePt(it.occurred_at)}</span>,
    },
    {
      key: "type",
      label: "Tipo",
      sortable: true,
      width: "w-24",
      render: (it) => (
        <Badge tone={it.type === "income" ? "green" : "red"}>
          {it.type === "income" ? "Entrada" : "Saida"}
        </Badge>
      ),
    },
    {
      key: "category_name",
      label: "Conta contabil",
      sortable: true,
      render: (it) => <span className="text-zinc-500">{it.category_name ?? "-"}</span>,
    },
    {
      key: "description",
      label: "Descricao",
      sortable: true,
      render: (it) => <span className="text-zinc-500">{it.description ?? "-"}</span>,
    },
    {
      key: "amount",
      label: "Valor",
      sortable: true,
      align: "right",
      render: (it) => (
        <span className={`font-medium ${it.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
          {currency(it.amount)}
        </span>
      ),
    },
    {
      key: "actions",
      label: "",
      width: "w-16",
      align: "center",
      render: (it) => (
        <Button variant="ghost" className="h-7 px-2 text-xs" title="Detalhes do lancamento" onClick={() => openDetail(it)}>
          <Eye className="h-4 w-4" />
          {it.attachment_count > 0 && <span className="ml-0.5 text-[10px]">{it.attachment_count}</span>}
        </Button>
      ),
    },
  ];

  if (selected) {
    return (
      <div className="page">
        <PageHeader
          title={selected.title}
          description={`Periodo: ${datePt(selected.period_start)} a ${datePt(selected.period_end)}`}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => { setSelected(null); loadAudits(); }}>
                <ArrowLeft className="h-4 w-4" /> Voltar
              </Button>
              {isHQ && (
                <Button variant="ghost" onClick={() => removeAudit(selected)} title="Excluir auditoria">
                  <Trash2 className="h-4 w-4 text-red-500" /> Excluir
                </Button>
              )}
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
            Auditados {auditedCount}/{items.length} - Total {currency(selected.total_amount)} - Auditado {currency(selected.audited_amount)}
          </span>
          {isOpen && canAudit && (
            <div className="ml-auto flex flex-wrap gap-2">
              <Button size="sm" variant="outline" onClick={() => markAll(true)}>
                <CheckCircle2 className="h-4 w-4" /> Marcar todos
              </Button>
              <Button size="sm" variant="ghost" onClick={() => markAll(false)}>Limpar marcacoes</Button>
              <Button size="sm" onClick={() => setCloseOpen(true)}>
                <Lock className="h-4 w-4" /> Fechar e assinar
              </Button>
            </div>
          )}
        </div>

        {!isOpen && selected.signer_name && (
          <Card className="mb-4 text-sm">
            <p className="font-medium">Documento auditado (imutavel)</p>
            <p className="mt-1 text-zinc-500">
              Responsavel: <strong>{selected.signer_name}</strong>
              {selected.signer_role ? ` (${selected.signer_role})` : ""}
              {selected.closed_at ? ` - Fechado em ${dateTimePt(selected.closed_at)}` : ""}
            </p>
            {selected.signature_hash && (
              <p className="mt-1 break-all text-xs text-zinc-400">Hash: {selected.signature_hash}</p>
            )}
          </Card>
        )}

        {items.length > 0 && (
          <div className="mb-3 flex flex-wrap items-end gap-2">
            <div className="relative min-w-[220px] flex-1">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <Input
                className="pl-8"
                placeholder="Buscar por descricao, conta, doador..."
                value={itemSearch}
                onChange={(e) => setItemSearch(e.target.value)}
              />
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-zinc-500">Periodo</span>
              <Input type="date" className="w-40" value={itemFrom} onChange={(e) => setItemFrom(e.target.value)} aria-label="Data inicial" />
              <span className="text-xs text-zinc-400">a</span>
              <Input type="date" className="w-40" value={itemTo} onChange={(e) => setItemTo(e.target.value)} aria-label="Data final" />
              {(itemSearch || itemFrom || itemTo) && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => { setItemSearch(""); setItemFrom(""); setItemTo(""); }}
                >
                  Limpar
                </Button>
              )}
            </div>
          </div>
        )}
        {items.length > 0 && (itemSearch || itemFrom || itemTo) && (
          <p className="mb-2 text-xs text-zinc-500">
            Mostrando {filteredItems.length} de {items.length} lancamento(s).
          </p>
        )}

        <DataTable
          columns={itemColumns}
          data={filteredItems}
          keyExtractor={(it) => it.transaction_id}
          loading={loadingItems}
          sort={{ column: sortColumn, direction: sortDirection, onSort: handleItemSort }}
          emptyIcon={<ShieldCheck className="h-10 w-10" />}
          emptyMessage={items.length === 0 ? "Sem lancamentos" : "Nenhum lancamento no filtro"}
          emptyDescription={items.length === 0 ? "Nenhum lancamento no periodo desta auditoria." : "Ajuste o periodo ou a busca."}
          compact
        />

        <Modal open={closeOpen} onClose={() => setCloseOpen(false)} title="Fechar e assinar auditoria">
          <form onSubmit={doClose} className="space-y-3 text-sm">
            <p className="text-zinc-500">
              Ao fechar, a auditoria vira um documento imutavel: nao sera possivel marcar/desmarcar
              lancamentos nem estornar os lancamentos auditados.
            </p>
            <Field label="Nome do responsavel *">
              <Input required value={signerName} onChange={(e) => setSignerName(e.target.value)} />
            </Field>
            <Field label="Funcao / cargo">
              <Input value={signerRole} onChange={(e) => setSignerRole(e.target.value)} placeholder="Ex.: Tesoureiro(a)" />
            </Field>
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" type="button" onClick={() => setCloseOpen(false)}>Cancelar</Button>
              <Button type="submit" disabled={closing}>{closing ? "Fechando..." : "Fechar e assinar"}</Button>
            </div>
          </form>
        </Modal>

        <Drawer open={!!detailItem} onClose={() => setDetailItem(null)} title="Detalhes do lancamento">
          {detailItem && (
            <div className="space-y-3 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-zinc-500">Valor</span>
                <span className={`text-xl font-semibold ${detailItem.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
                  {detailItem.type === "income" ? "+" : "-"}{currency(detailItem.amount)}
                </span>
              </div>
              <DetRow label="Tipo" value={detailItem.type === "income" ? "Entrada" : "Saida"} />
              <DetRow label="Data" value={datePt(detailItem.occurred_at)} />
              <DetRow label="Conta contabil" value={detailItem.category_name ?? "-"} />
              <DetRow label="Conta bancaria" value={detailItem.account_name ?? "-"} />
              <DetRow label="Fornecedor" value={detailItem.supplier_name ?? "-"} />
              <DetRow label="Forma de pagamento" value={detailItem.payment_method ? (PAYMENT_METHODS[detailItem.payment_method] ?? detailItem.payment_method) : "-"} />
              <DetRow label="Origem" value={detailItem.is_anonymous ? "Anonimo" : (detailItem.donor_name ?? "-")} />
              {detailItem.receipt_ref && <DetRow label="Recibo" value={detailItem.receipt_ref} />}
              <div>
                <span className="text-zinc-500">Descricao</span>
                <p className="mt-0.5 whitespace-pre-wrap">{detailItem.description ?? "-"}</p>
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

  const auditColumns: Column<FinancialAudit>[] = [
    {
      key: "title",
      label: "Titulo",
      sortable: true,
      render: (a) => <span className="font-medium">{a.title}</span>,
    },
    {
      key: "period_start",
      label: "Periodo",
      sortable: true,
      sortValue: (a) => a.period_start,
      render: (a) => <span className="text-zinc-500">{datePt(a.period_start)} a {datePt(a.period_end)}</span>,
    },
    {
      key: "status",
      label: "Situacao",
      sortable: true,
      width: "w-28",
      render: (a) => (
        <Badge tone={a.status === "aberta" ? "amber" : "green"}>
          {a.status === "aberta" ? "Aberta" : "Fechada"}
        </Badge>
      ),
    },
    {
      key: "audited_items",
      label: "Auditados",
      sortable: true,
      align: "right",
      width: "w-28",
      render: (a) => <span className="text-zinc-500">{a.audited_items}/{a.total_items}</span>,
    },
    {
      key: "actions",
      label: "",
      align: "right",
      width: "w-40",
      render: (a) => (
        <div className="flex justify-end gap-1">
          <Button size="sm" variant="outline" onClick={() => openAudit(a)}>
            <Search className="h-4 w-4" /> Abrir
          </Button>
          {isHQ && (
            <Button size="sm" variant="ghost" onClick={() => removeAudit(a)} aria-label="Excluir auditoria" title="Excluir">
              <Trash2 className="h-4 w-4 text-red-500" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="page">
      <PageHeader
        title="Auditoria financeira"
        description="Lista lancamentos do periodo, permite marcar como auditado e fechar com assinatura"
        actions={canAudit ? <Button onClick={() => setShowNew(true)}><Plus className="h-4 w-4" /> Nova auditoria</Button> : undefined}
      />

      {audits !== null && audits.length > 0 && (
        <div className="mb-3 flex flex-wrap items-end gap-2">
          <div className="relative min-w-[220px] flex-1">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
            <Input
              className="pl-8"
              placeholder="Buscar por titulo ou responsavel..."
              value={auditSearch}
              onChange={(e) => setAuditSearch(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-zinc-500">Periodo</span>
            <Input type="date" className="w-40" value={auditFrom} onChange={(e) => setAuditFrom(e.target.value)} aria-label="Periodo inicial" />
            <span className="text-xs text-zinc-400">a</span>
            <Input type="date" className="w-40" value={auditTo} onChange={(e) => setAuditTo(e.target.value)} aria-label="Periodo final" />
            {(auditSearch || auditFrom || auditTo) && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => { setAuditSearch(""); setAuditFrom(""); setAuditTo(""); }}
              >
                Limpar
              </Button>
            )}
          </div>
        </div>
      )}

      <DataTable
        columns={auditColumns}
        data={filteredAudits}
        keyExtractor={(a) => a.id}
        loading={audits === null}
        sort={{ column: auditSortColumn, direction: auditSortDirection, onSort: handleAuditSort }}
        emptyIcon={<ShieldCheck className="h-10 w-10" />}
        emptyMessage={audits && audits.length > 0 ? "Nenhuma auditoria no filtro" : "Nenhuma auditoria"}
        emptyDescription={audits && audits.length > 0 ? "Ajuste o periodo ou a busca." : "Crie uma auditoria para um periodo e marque os lancamentos."}
      />

      <Drawer open={showNew} onClose={() => setShowNew(false)} title="Nova auditoria">
        <form onSubmit={createNew} className="space-y-3">
          <Field label="Titulo"><Input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="Periodo - de *"><Input type="date" required value={form.period_start} onChange={(e) => setForm({ ...form, period_start: e.target.value })} /></Field>
          <Field label="Periodo - ate *"><Input type="date" required value={form.period_end} onChange={(e) => setForm({ ...form, period_end: e.target.value })} /></Field>
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
