"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Landmark, Upload, ArrowLeft, Trash2, Link2, AlertTriangle, CheckCircle2, X } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { DataTable, type Column } from "@/components/ui/data-table";
import { Input, Field, Select } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listBankImports, importBankStatement, getBankImport, ignoreBankEntry,
  generateBankEntry, deleteBankImport, listAccounts, listCategories,
  type BankImport, type BankEntry, type BankDivergence, type BankAccount, type Category,
} from "@/lib/api";
import { currency, datePt } from "@/lib/format";

const ENTRY_STATUS: Record<BankEntry["status"], { label: string; tone: "green" | "amber" | "zinc" | "sky" }> = {
  conciliado: { label: "Conciliado", tone: "green" },
  pendente: { label: "Sem correspondencia", tone: "amber" },
  ignorado: { label: "Ignorado", tone: "zinc" },
  lancamento_gerado: { label: "Lancamento gerado", tone: "sky" },
};

function fileToBase64(f: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => {
      const s = String(r.result);
      const i = s.indexOf(",");
      resolve(i >= 0 ? s.slice(i + 1) : s);
    };
    r.onerror = () => reject(new Error("Falha ao ler o arquivo"));
    r.readAsDataURL(f);
  });
}

export function BankReconciliationPanel() {
  const { toast } = useToast();
  const { hasPerm, user } = useAuth();
  const canReconcile = hasPerm("finance.reconcile");
  const canWrite = hasPerm("finance.write");
  const isHQ = user?.role === "super_admin" || user?.role === "admin_sede";

  const [imports, setImports] = useState<BankImport[] | null>(null);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);

  const [showImport, setShowImport] = useState(false);
  const [impForm, setImpForm] = useState({ account_id: "" });
  const [file, setFile] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const [selected, setSelected] = useState<BankImport | null>(null);
  const [entries, setEntries] = useState<BankEntry[]>([]);
  const [divergences, setDivergences] = useState<BankDivergence[]>([]);
  const [loadingDetail, setLoadingDetail] = useState(false);

  const [genEntry, setGenEntry] = useState<BankEntry | null>(null);
  const [genForm, setGenForm] = useState({ category_id: "", description: "" });
  const [generating, setGenerating] = useState(false);

  const load = useCallback(async () => {
    try {
      setImports((await listBankImports()).imports);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar importacoes", "error");
    }
  }, [toast]);

  useEffect(() => {
    load();
    listAccounts().then((r) => setAccounts(r.accounts.filter((a) => a.is_active))).catch(() => {});
    listCategories().then((r) => setCategories(r.categories)).catch(() => {});
  }, [load]);

  async function openImportDetail(imp: BankImport) {
    setSelected(imp);
    setLoadingDetail(true);
    try {
      const res = await getBankImport(imp.id);
      setSelected(res.import);
      setEntries(res.entries);
      setDivergences(res.divergences);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    } finally {
      setLoadingDetail(false);
    }
  }

  async function submitImport(e: React.FormEvent) {
    e.preventDefault();
    if (!impForm.account_id) {
      toast("Selecione a conta bancaria.", "error");
      return;
    }
    if (!file) {
      toast("Selecione o arquivo do extrato (OFX ou CSV).", "error");
      return;
    }
    setSaving(true);
    try {
      const data = await fileToBase64(file);
      const imp = await importBankStatement({ account_id: impForm.account_id, filename: file.name, data });
      toast("Extrato importado.");
      setShowImport(false);
      setFile(null);
      if (fileRef.current) fileRef.current.value = "";
      await load();
      await openImportDetail(imp);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao importar extrato", "error");
    } finally {
      setSaving(false);
    }
  }

  async function ignore(entry: BankEntry) {
    if (!selected) return;
    try {
      await ignoreBankEntry(selected.id, entry.id);
      await openImportDetail(selected);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  function openGenerate(entry: BankEntry) {
    setGenEntry(entry);
    setGenForm({ category_id: "", description: entry.memo ?? "" });
  }

  async function submitGenerate(e: React.FormEvent) {
    e.preventDefault();
    if (!selected || !genEntry) return;
    if (!genForm.category_id) {
      toast("Selecione a conta contabil.", "error");
      return;
    }
    setGenerating(true);
    try {
      await generateBankEntry(selected.id, genEntry.id, {
        category_id: genForm.category_id,
        description: genForm.description || undefined,
      });
      toast("Lancamento gerado e vinculado ao extrato.");
      setGenEntry(null);
      await openImportDetail(selected);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao gerar lancamento", "error");
    } finally {
      setGenerating(false);
    }
  }

  async function remove(imp: BankImport) {
    if (!confirm(`Excluir a importacao "${imp.filename}"? As entradas do extrato serao removidas.`)) return;
    try {
      await deleteBankImport(imp.id);
      toast("Importacao excluida.");
      setSelected(null);
      await load();
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
    }
  }

  const catsFor = useMemo(() => {
    if (!genEntry) return [] as Category[];
    return categories.filter((c) => c.type === genEntry.direction);
  }, [categories, genEntry]);

  const importColumns: Column<BankImport>[] = [
    { key: "filename", label: "Arquivo", sortable: true, render: (b) => <span className="font-medium">{b.filename || "-"}</span> },
    { key: "account_name", label: "Conta", sortable: true, render: (b) => <span className="text-zinc-500">{b.account_name ?? "-"}</span> },
    {
      key: "period_start",
      label: "Periodo",
      sortable: true,
      render: (b) => <span className="text-zinc-500">{b.period_start ? `${datePt(b.period_start)} a ${datePt(b.period_end)}` : "-"}</span>,
    },
    { key: "format", label: "Formato", width: "w-24", render: (b) => <Badge tone="zinc">{b.format.toUpperCase()}</Badge> },
    { key: "matched_entries", label: "Conciliados", sortable: true, align: "right", render: (b) => <span className="text-emerald-600">{b.matched_entries}/{b.total_entries}</span> },
    {
      key: "missing_entries",
      label: "Faltantes",
      sortable: true,
      align: "right",
      width: "w-24",
      render: (b) => <span className={b.missing_entries > 0 ? "text-amber-600" : "text-zinc-400"}>{b.missing_entries}</span>,
    },
    {
      key: "actions",
      label: "",
      align: "right",
      width: "w-40",
      render: (b) => (
        <div className="flex justify-end gap-1">
          <Button size="sm" variant="outline" onClick={() => openImportDetail(b)}><Link2 className="h-4 w-4" /> Abrir</Button>
          {isHQ && (
            <Button size="sm" variant="ghost" onClick={() => remove(b)} aria-label="Excluir importacao" title="Excluir">
              <Trash2 className="h-4 w-4 text-red-500" />
            </Button>
          )}
        </div>
      ),
    },
  ];

  const entryColumns: Column<BankEntry>[] = [
    { key: "posted_at", label: "Data", sortable: true, width: "w-28", render: (e) => <span className="text-zinc-500">{datePt(e.posted_at)}</span> },
    { key: "memo", label: "Historico", sortable: true, render: (e) => <span className="text-zinc-500">{e.memo ?? "-"}</span> },
    { key: "direction", label: "Tipo", sortable: true, width: "w-24", render: (e) => <Badge tone={e.direction === "income" ? "green" : "red"}>{e.direction === "income" ? "Credito" : "Debito"}</Badge> },
    { key: "amount", label: "Valor", sortable: true, align: "right", render: (e) => <span className="font-medium tabular-nums">{currency(e.amount)}</span> },
    { key: "status", label: "Situacao", sortable: true, render: (e) => <Badge tone={ENTRY_STATUS[e.status].tone}>{ENTRY_STATUS[e.status].label}</Badge> },
    {
      key: "actions",
      label: "",
      align: "right",
      width: "w-56",
      render: (e) => {
        if (e.status !== "pendente") {
          return <span className="text-xs text-zinc-400">{e.transaction_description ?? ""}</span>;
        }
        return (
          <div className="flex justify-end gap-1">
            {canReconcile && (
              <Button size="sm" variant="ghost" onClick={() => ignore(e)} title="Ignorar esta linha do extrato">
                <X className="h-4 w-4" /> Ignorar
              </Button>
            )}
            {canWrite && (
              <Button size="sm" variant="outline" onClick={() => openGenerate(e)} title="Gerar lancamento a partir do extrato">
                <Upload className="h-4 w-4" /> Gerar lancamento
              </Button>
            )}
          </div>
        );
      },
    },
  ];

  const divergenceColumns: Column<BankDivergence>[] = [
    { key: "occurred_at", label: "Data", sortable: true, width: "w-28", render: (d) => <span className="text-zinc-500">{datePt(d.occurred_at)}</span> },
    { key: "description", label: "Descricao", sortable: true, render: (d) => <span className="text-zinc-500">{d.description ?? "-"}</span> },
    { key: "category_name", label: "Conta contabil", sortable: true, render: (d) => <span className="text-zinc-500">{d.category_name ?? "-"}</span> },
    {
      key: "amount",
      label: "Valor",
      sortable: true,
      align: "right",
      render: (d) => <span className={`font-medium ${d.direction === "income" ? "text-emerald-600" : "text-red-600"}`}>{currency(d.amount)}</span>,
    },
  ];

  if (selected) {
    return (
      <div>
        <PageHeader
          title={selected.filename || "Extrato"}
          description={`${selected.account_name ?? "conta"}${selected.period_start ? ` - ${datePt(selected.period_start)} a ${datePt(selected.period_end)}` : ""}`}
          actions={
            <div className="flex items-center gap-2">
              <Button variant="ghost" onClick={() => { setSelected(null); load(); }}><ArrowLeft className="h-4 w-4" /> Voltar</Button>
              {isHQ && <Button variant="ghost" onClick={() => remove(selected)}><Trash2 className="h-4 w-4 text-red-500" /> Excluir</Button>}
            </div>
          }
        />

        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Badge tone="green"><CheckCircle2 className="mr-1 h-3.5 w-3.5" /> {selected.matched_entries} conciliado(s)</Badge>
          <Badge tone={selected.missing_entries > 0 ? "amber" : "zinc"}><AlertTriangle className="mr-1 h-3.5 w-3.5" /> {selected.missing_entries} sem correspondencia</Badge>
          <Badge tone="zinc">{selected.total_entries} linha(s) no extrato</Badge>
        </div>

        <h3 className="mb-2 text-sm font-semibold text-zinc-700">Linhas do extrato</h3>
        <DataTable
          columns={entryColumns}
          data={entries}
          keyExtractor={(e) => e.id}
          loading={loadingDetail}
          emptyIcon={<Landmark className="h-10 w-10" />}
          emptyMessage="Extrato vazio"
          emptyDescription="Nenhuma linha importada."
          compact
        />

        <h3 className="mb-2 mt-6 text-sm font-semibold text-zinc-700">Divergencias - no ERP sem extrato</h3>
        <DataTable
          columns={divergenceColumns}
          data={divergences}
          keyExtractor={(d) => d.id}
          loading={loadingDetail}
          emptyIcon={<CheckCircle2 className="h-10 w-10" />}
          emptyMessage="Sem divergencias"
          emptyDescription="Todos os lancamentos do periodo tem correspondencia no extrato."
          compact
        />

        <Modal open={!!genEntry} onClose={() => setGenEntry(null)} title="Gerar lancamento do extrato">
          {genEntry && (
            <form onSubmit={submitGenerate} className="space-y-3 text-sm">
              <p className="text-zinc-500">
                {genEntry.direction === "income" ? "Credito" : "Debito"} de <strong>{currency(genEntry.amount)}</strong> em {datePt(genEntry.posted_at)}
                {genEntry.memo ? ` - ${genEntry.memo}` : ""}
              </p>
              <Field label="Conta contabil *">
                <Select value={genForm.category_id} onChange={(e) => setGenForm({ ...genForm, category_id: e.target.value })}>
                  <option value="">Selecione...</option>
                  {catsFor.map((c) => <option key={c.id} value={c.id}>{c.code} - {c.name}</option>)}
                </Select>
              </Field>
              <Field label="Descricao">
                <Input value={genForm.description} onChange={(e) => setGenForm({ ...genForm, description: e.target.value })} />
              </Field>
              <div className="flex justify-end gap-2 pt-2">
                <Button variant="ghost" type="button" onClick={() => setGenEntry(null)}>Cancelar</Button>
                <Button type="submit" disabled={generating}>{generating ? "Gerando..." : "Gerar lancamento"}</Button>
              </div>
            </form>
          )}
        </Modal>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <p className="flex-1 text-sm text-zinc-500">
          Importe o extrato do banco (OFX ou CSV), case com os lancamentos e gere os faltantes.
        </p>
        {canReconcile && (
          <Button onClick={() => setShowImport(true)}>
            <Upload className="h-4 w-4" /> Importar extrato
          </Button>
        )}
      </div>

      <DataTable
        columns={importColumns}
        data={imports ?? []}
        keyExtractor={(b) => b.id}
        loading={imports === null}
        emptyIcon={<Landmark className="h-10 w-10" />}
        emptyMessage="Nenhum extrato importado"
        emptyDescription="Importe um arquivo OFX ou CSV do seu banco."
      />

      <Drawer open={showImport} onClose={() => setShowImport(false)} title="Importar extrato bancario">
        <form onSubmit={submitImport} className="space-y-3">
          <Field label="Conta bancaria *">
            <Select value={impForm.account_id} onChange={(e) => setImpForm({ account_id: e.target.value })}>
              <option value="">Selecione...</option>
              {accounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            </Select>
          </Field>
          <Field label="Arquivo (OFX ou CSV) *">
            <label className="flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-zinc-300 px-3 py-3 text-sm text-zinc-600 hover:border-zinc-400 dark:border-zinc-700">
              <Upload className="h-4 w-4" />
              <span>{file ? file.name : "Escolher arquivo"}</span>
              <input
                ref={fileRef}
                type="file"
                accept=".ofx,.csv,.txt,text/csv"
                className="hidden"
                onChange={(e) => setFile(e.target.files?.[0] ?? null)}
              />
            </label>
          </Field>
          <p className="text-xs text-zinc-400">
            O casamento e automatico por valor, tipo (credito/debito) e data (tolerancia de 3 dias).
          </p>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowImport(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Importando..." : "Importar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
