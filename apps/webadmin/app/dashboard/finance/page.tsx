"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowDownRight, ArrowUpRight, Wallet, Plus, Eye, Send, Filter, Tags, Repeat, Banknote, Upload, FileText, Paperclip, Pencil, Trash2, FolderTree } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { StatCard } from "@/components/ui/stat-card";
import { Drawer, Modal } from "@/components/ui/modal";
import { Tabs } from "@/components/ui/tabs";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input, Field, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTable, Column } from "@/components/ui/data-table";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
   listCategories, listTransactions, getBalance,
   getReceiptHTML, sendReceipt, createCategory, updateCategory, deleteCategory,
   listCategoryGroups, createCategoryGroup, updateCategoryGroup, deleteCategoryGroup,
   listAccounts, createAccount, updateAccount, deleteAccount,
   uploadAttachment, listAttachments, listDeliveries, createTransaction, deleteTransaction,
   listTransactionEvents, previewTransactions, importTransactionsFile,
   type Category, type CategoryGroup, type Transaction, type Balance, type BankAccount,
   type FinancialAttachment, type Delivery, type ImportResult, type EventAllocation,
 } from "@/lib/api";
import { PAYMENT_METHODS, ACCOUNT_TYPES } from "@/lib/constants";
import { currency, datePt, dateTimePt } from "@/lib/format";
import { RecurringPanel } from "@/components/finance/recurring-panel";
import { BulkEntry } from "@/components/finance/bulk-entry";
import { TransactionForm, type TransactionFormState } from "@/components/finance/transaction-form";

const PER_PAGE = 12;

const IMPORT_FIELDS: { key: string; label: string; required?: boolean }[] = [
  { key: "data", label: "Data", required: true },
  { key: "tipo", label: "Tipo (entrada/saida)", required: true },
  { key: "conta", label: "Conta contabil", required: true },
  { key: "valor", label: "Valor", required: true },
  { key: "forma_pagamento", label: "Forma de pagamento" },
  { key: "descricao", label: "Descricao" },
  { key: "anonimo", label: "Anonimo" },
  { key: "conta_bancaria", label: "Conta bancaria" },
];

function colLetter(i: number): string {
  let s = "";
  let n = i;
  do { s = String.fromCharCode(65 + (n % 26)) + s; n = Math.floor(n / 26) - 1; } while (n >= 0);
  return s;
}

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

function guessMapping(rows: string[][]): Record<string, number> {
  if (rows.length === 0) return {};
  const header = rows[0].map((h) => (h ?? "").toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, ""));
  const find = (aliases: string[]) => header.findIndex((h) => aliases.some((a) => h.includes(a)));
  const m: Record<string, number> = {};
  const set = (k: string, i: number) => { if (i >= 0) m[k] = i; };
  set("data", find(["data", "dia", "date"]));
  set("tipo", find(["tipo", "type", "entrada"]));
  set("conta", find(["conta", "categoria", "codigo", "rubrica"]));
  set("valor", find(["valor", "amount"]));
  set("forma_pagamento", find(["forma", "pagamento"]));
  set("descricao", find(["descri", "histor", "observa"]));
  set("anonimo", find(["anonim"]));
  set("conta_bancaria", find(["banco", "bancar"]));
  return m;
}

export default function FinancePage() {
  const { hasPerm } = useAuth();
  const { toast } = useToast();
  const [tab, setTab] = useState("lancamentos");
  const [categories, setCategories] = useState<Category[]>([]);
  const [groups, setGroups] = useState<CategoryGroup[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [txns, setTxns] = useState<Transaction[]>([]);
  const [balance, setBalance] = useState<Balance | null>(null);
  const [loading, setLoading] = useState(true);
  const [filterType, setFilterType] = useState("");
  const [filterCat, setFilterCat] = useState("");
  const [filterAcct, setFilterAcct] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(PER_PAGE);
  const [sortColumn, setSortColumn] = useState("occurred_at");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [detail, setDetail] = useState<Transaction | null>(null);
  const [attachments, setAttachments] = useState<FinancialAttachment[]>([]);
  const [allocations, setAllocations] = useState<EventAllocation[]>([]);
  const [showNew, setShowNew] = useState(false);
  const [editing, setEditing] = useState<Transaction | null>(null);
  const [showCatForm, setShowCatForm] = useState(false);
  const [showAcctForm, setShowAcctForm] = useState(false);
  const [showImport, setShowImport] = useState(false);
  const [importFile, setImportFile] = useState<{ data: string; filename: string } | null>(null);
  const [importRows, setImportRows] = useState<string[][]>([]);
  const [importStartRow, setImportStartRow] = useState(1);
  const [importMap, setImportMap] = useState<Record<string, number>>({});
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [sendDrawer, setSendDrawer] = useState<{ open: boolean; txn: Transaction | null; channel: "email" | "whatsapp" }>({ open: false, txn: null, channel: "email" });
  const [catForm, setCatForm] = useState({ type: "income", code: "", name: "", group_id: "" });
  const [showGroupForm, setShowGroupForm] = useState(false);
  const [groupForm, setGroupForm] = useState({ name: "", sort_order: "" });
  const [editingGroup, setEditingGroup] = useState<CategoryGroup | null>(null);
  const [acctForm, setAcctForm] = useState({ name: "", bank: "", bank_code: "", agency: "", account_number: "", account_type: "checking", initial_balance: "" });
  const [editingCat, setEditingCat] = useState<Category | null>(null);
  const [editingAcct, setEditingAcct] = useState<BankAccount | null>(null);
  const [deliveries, setDeliveries] = useState<Delivery[]>([]);
  const [sending, setSending] = useState(false);
  const [recipient, setRecipient] = useState("");

  const load = useCallback(async () => {
    // Cada API e carregada independentemente: uma falha nao deve bloquear o
    // carregamento do restante. Mas a lista de lancamentos AVISA quando falha
    // (antes o erro era engolido e a tela so ficava "vazia").
    const cat = await listCategories().catch(() => ({ categories: [] }));
    const grp = await listCategoryGroups().catch(() => ({ groups: [] }));
    const accts = await listAccounts().catch(() => ({ accounts: [] }));
    try {
      const tx = await listTransactions();
      setTxns(Array.isArray(tx.transactions) ? tx.transactions : []);
    } catch (e) {
      setTxns([]);
      toast(e instanceof Error ? e.message : "Falha ao carregar lancamentos", "error");
    }
    const bal = await getBalance().catch(() => null);
    setCategories(cat.categories);
    setGroups(grp.groups);
    setAccounts(accts.accounts);
    setBalance(bal);
    setLoading(false);
  }, [toast]);

  useEffect(() => {
    load().catch((e) => toast(e.message, "error"));
  }, [load, toast]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const list = Array.isArray(txns) ? txns : [];
    return list.filter((t) => {
      return (
        (!filterType || t.type === filterType) &&
         (!filterCat || t.category_id === filterCat) &&
         (!filterAcct || t.account_id === filterAcct) &&
        (!q || t.description?.toLowerCase().includes(q))
      );
    });
  }, [txns, filterType, filterCat, filterAcct, query]);

  useEffect(() => setPage(1), [filterType, filterCat, query]);

  const handleSort = (col: string) => {
    if (sortColumn === col) {
      setSortDirection(sortDirection === "asc" ? "desc" : "asc");
    } else {
      setSortColumn(col);
      setSortDirection("desc");
    }
  };

  const columns: Column<Transaction>[] = useMemo(() => [
    {
      key: "occurred_at",
      label: "Data efetiva",
      sortable: true,
      width: "w-28",
      // Ordena pela data da ocorrencia e, dentro dela, pela sequencia do
      // lancamento (created_at), igual a conciliacao/auditoria.
      sortValue: (t) => `${t.occurred_at}T${t.created_at ?? ""}`,
      render: (t) => <span className="text-zinc-500">{datePt(t.occurred_at)}</span>,
    },
    {
      key: "description",
      label: "Descricao",
      render: (t) => (
        <div className="flex items-center gap-2">
          <span className={t.voided_at ? "text-zinc-400 line-through" : ""}>{t.description ?? "-"}</span>
          {t.voided_at && <Badge tone="red" className="text-[10px]">Estornado</Badge>}
          {!!t.attachment_count && t.attachment_count > 0 && (
            t.attachment_count === 1 ? (
              <a
                href={t.attachment_url}
                target="_blank"
                rel="noopener noreferrer"
                title="Ver anexo"
                className="inline-flex items-center text-sky-600 hover:text-sky-700"
              >
                <Paperclip className="h-3.5 w-3.5" />
              </a>
            ) : (
              <button
                type="button"
                onClick={() => openDetail(t)}
                title={`Ver os ${t.attachment_count} anexos`}
                className="inline-flex items-center gap-0.5 text-sky-600 hover:text-sky-700"
              >
                <Paperclip className="h-3.5 w-3.5" />
                <span className="text-[10px] font-medium">{t.attachment_count}</span>
              </button>
            )
          )}
        </div>
      ),
    },
    {
      key: "category_name",
      label: "Conta contabil",
      sortable: true,
      render: (t) => (
        <div>
          <p className="font-medium">{t.category_name ?? "-"}</p>
          {t.payment_method && <p className="text-xs text-zinc-400">{PAYMENT_METHODS[t.payment_method] ?? t.payment_method}</p>}
        </div>
      ),
    },
    {
      key: "account_name",
      label: "Conta",
      sortable: true,
      width: "w-36",
      render: (t) => <span className="text-zinc-500">{t.account_name ?? "-"}</span>,
    },
    {
      key: "amount",
      label: "Valor",
      sortable: false,
      align: "right" as const,
      width: "w-28",
      render: (t) => (
        <span className={`font-semibold ${t.type === "income" ? "text-emerald-600" : "text-red-600"}`}>
          {t.type === "income" ? "+" : "-"}{currency(t.amount)}
        </span>
      ),
    },
    {
      key: "id",
      label: "",
      sortable: false,
      width: "w-40",
      align: "right" as const,
      render: (t) => (
        <div className="flex items-center justify-end gap-0.5">
          {t.receipt_id && (
            <Button variant="ghost" size="sm" onClick={() => viewReceipt(t)} aria-label="Ver recibo" title="Ver recibo"><Eye className="h-3.5 w-3.5" /> Ver</Button>
          )}
          <Button variant="ghost" size="sm" onClick={() => openDetail(t)} aria-label="Ver detalhes" title="Detalhes"><Eye className="h-3.5 w-3.5" /></Button>
          {!t.voided_at && hasPerm("finance.write") && (
            <>
              <Button variant="ghost" size="sm" onClick={() => openEdit(t)} aria-label="Editar" title="Editar"><Pencil className="h-3.5 w-3.5" /></Button>
              <Button variant="ghost" size="sm" onClick={() => removeTxn(t)} aria-label="Excluir" title="Excluir"><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>
            </>
          )}
        </div>
      ),
    },
  ], [hasPerm]);

  async function handleTxnSaved() {
    // Invalidar apenas as listas de transacoes e saldo, nao todas as APIs.
    const [tx, bal] = await Promise.all([
      listTransactions().catch(() => ({ transactions: [] })),
      getBalance().catch(() => null),
    ]);
    setTxns(tx.transactions);
    setBalance(bal);
  }

  async function viewReceipt(t: Transaction) {
    if (!t.receipt_id) return;
    // Abre a janela ANTES do fetch: abrir depois do await e bloqueado pelo
    // bloqueador de pop-up (deixaria de ser um gesto do usuario).
    const w = window.open("", "_blank", "width=540,height=760");
    try {
      const html = await getReceiptHTML(t.receipt_id);
      if (w) {
        w.document.write(html);
        w.document.close();
      }
    } catch (err) {
      w?.close();
      toast(err instanceof Error ? err.message : "Erro ao abrir o recibo", "error");
    }
  }

  async function openEdit(t: Transaction) {
    setEditing(t);
    setShowNew(true);
  }

  // "Alterar" = excluir o original e lancar o novo (o backend recalcula a cadeia).
  async function amendTxn(data: Record<string, unknown>) {
    if (!editing) return;
    await deleteTransaction(editing.id);
    await createTransaction(data);
    toast("Lancamento corrigido.");
    setEditing(null);
    await handleTxnSaved();
  }

  async function removeTxn(t: Transaction) {
    if (!confirm("Excluir definitivamente este lancamento? Esta acao nao pode ser desfeita.")) return;
    try {
      await deleteTransaction(t.id);
      toast("Lancamento excluido.");
      await handleTxnSaved();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function openSendDrawer(t: Transaction, channel: "email" | "whatsapp") {
    setSendDrawer({ open: true, txn: t, channel });
  }

  async function submitSend(recipient: string) {
    const t = sendDrawer.txn;
    if (!t || !t.receipt_id) return;
    setSending(true);
    try {
      await sendReceipt(t.receipt_id, sendDrawer.channel, recipient);
      toast(`Enviado por ${sendDrawer.channel}.`);
      setSendDrawer({ open: false, txn: null, channel: "email" });
      if (t.receipt_id) {
        setRecipient("");
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSending(false);
    }
  }

  async function openDetail(t: Transaction) {
    setDetail(t);
    try {
      const [res, alloc] = await Promise.all([
        listAttachments(t.id),
        listTransactionEvents(t.id).catch(() => ({ allocations: [] })),
      ]);
      setAttachments(res.attachments);
      setAllocations(alloc.allocations);
      if (t.receipt_id) {
        const d = await listDeliveries(t.receipt_id).catch(() => ({ deliveries: [] }));
        setDeliveries(d.deliveries);
      }
    } catch {
      setAttachments([]);
      setDeliveries([]);
      setAllocations([]);
    }
  }

   async function attachFile(t: Transaction, file: File) {
    try {
      await uploadAttachment(t.id, file);
      toast("Anexo enviado.");
      const res = await listAttachments(t.id);
      setAttachments(res.attachments);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  function closeCatForm() {
    setShowCatForm(false);
    setEditingCat(null);
    setCatForm({ type: "income", code: "", name: "", group_id: "" });
  }

  function openCatEdit(c: Category) {
    setEditingCat(c);
    setCatForm({ type: c.type, code: c.code, name: c.name, group_id: c.group_id ?? "" });
    setShowCatForm(true);
  }

  /** Recarrega plano de contas + grupos (usado apos qualquer alteracao). */
  async function reloadPlan() {
    const [cat, grp] = await Promise.all([
      listCategories().catch(() => ({ categories: [] })),
      listCategoryGroups().catch(() => ({ groups: [] })),
    ]);
    setCategories(cat.categories);
    setGroups(grp.groups);
  }

  async function submitCat(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (editingCat) {
        await updateCategory(editingCat.id, catForm);
        toast("Conta contabil atualizada.");
      } else {
        await createCategory(catForm);
        toast("Conta contabil criada.");
      }
      closeCatForm();
      await reloadPlan();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function toggleCategoryActive(c: Category) {
    try {
      await updateCategory(c.id, { is_active: !c.is_active });
      await reloadPlan();
      toast(c.is_active ? "Conta contabil desativada." : "Conta contabil reativada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function deleteCat(c: Category) {
    if (!confirm(`Excluir a conta contabil "${c.name}"?`)) return;
    try {
      await deleteCategory(c.id);
      toast("Conta contabil excluida.");
      await reloadPlan();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  function closeGroupForm() {
    setShowGroupForm(false);
    setEditingGroup(null);
    setGroupForm({ name: "", sort_order: "" });
  }

  function openGroupEdit(g: CategoryGroup) {
    setEditingGroup(g);
    setGroupForm({ name: g.name, sort_order: String(g.sort_order) });
    setShowGroupForm(true);
  }

  async function submitGroup(e: React.FormEvent) {
    e.preventDefault();
    const payload = {
      name: groupForm.name,
      sort_order: groupForm.sort_order === "" ? undefined : Number(groupForm.sort_order),
    };
    try {
      if (editingGroup) {
        await updateCategoryGroup(editingGroup.id, payload);
        toast("Grupo atualizado.");
      } else {
        await createCategoryGroup(payload);
        toast("Grupo criado.");
      }
      closeGroupForm();
      await reloadPlan();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function toggleGroupActive(g: CategoryGroup) {
    try {
      await updateCategoryGroup(g.id, { is_active: !g.is_active });
      await reloadPlan();
      toast(g.is_active ? "Grupo desativado." : "Grupo reativado.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function deleteGroup(g: CategoryGroup) {
    if (!confirm(`Excluir o grupo "${g.name}"? As contas associadas ficam sem grupo.`)) return;
    try {
      await deleteCategoryGroup(g.id);
      toast("Grupo excluido.");
      await reloadPlan();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function pickImportFile(f: File) {
    try {
      const data = await fileToBase64(f);
      setImportFile({ data, filename: f.name });
      setImportResult(null);
      const { rows } = await previewTransactions(data, f.name);
      setImportRows(rows);
      setImportStartRow(rows.length > 1 ? 2 : 1);
      setImportMap(guessMapping(rows));
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao ler a planilha", "error");
    }
  }

  async function doImport() {
    if (!importFile) return;
    setImporting(true);
    try {
      const res = await importTransactionsFile(importFile.data, importFile.filename, importStartRow - 1, importMap);
      setImportResult(res);
      toast(`${res.imported} lancamento(s) importado(s).`);
      await handleTxnSaved();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao importar", "error");
    } finally {
      setImporting(false);
    }
  }

  async function submitAcct(e: React.FormEvent) {
    e.preventDefault();
    const balance = acctForm.initial_balance === "" ? 0 : Number(acctForm.initial_balance);
    if (isNaN(balance) || balance < 0) {
      toast("Saldo inicial invalido.", "error");
      return;
    }
    try {
      if (editingAcct) {
        await updateAccount(editingAcct.id, { ...acctForm, initial_balance: balance });
        toast("Conta atualizada.");
      } else {
        await createAccount({ ...acctForm, initial_balance: balance });
        toast("Conta criada.");
      }
      closeAcctForm();
      setAccounts(await listAccounts().then((r) => r.accounts));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  function closeAcctForm() {
    setShowAcctForm(false);
    setEditingAcct(null);
    setAcctForm({ name: "", bank: "", bank_code: "", agency: "", account_number: "", account_type: "checking", initial_balance: "" });
  }

  function openAcctEdit(a: BankAccount) {
    setEditingAcct(a);
    setAcctForm({
      name: a.name, bank: a.bank ?? "", bank_code: a.bank_code ?? "",
      agency: a.agency ?? "", account_number: a.account_number ?? "",
      account_type: a.account_type, initial_balance: String(a.initial_balance),
    });
    setShowAcctForm(true);
  }

  async function toggleAccountActive(a: BankAccount) {
    try {
      await updateAccount(a.id, { is_active: !a.is_active });
      setAccounts(await listAccounts().then((r) => r.accounts));
      toast(a.is_active ? "Conta desativada." : "Conta reativada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function deleteAcct(a: BankAccount) {
    if (!confirm(`Excluir a conta "${a.name}"?`)) return;
    try {
      await deleteAccount(a.id);
      toast("Conta excluida.");
      setAccounts(await listAccounts().then((r) => r.accounts));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  const incomeCats = categories.filter((c) => c.type === "income");
  const expenseCats = categories.filter((c) => c.type === "expense");

  return (
    <div className="page">
      <PageHeader
        title="Financeiro"
        description="Dizimos, ofertas, despesas e plano de contas"
        actions={hasPerm("finance.write") ? (
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => { setImportResult(null); setShowImport(true); }}>
              <Upload className="h-4 w-4" /> Importar
            </Button>
            <Button onClick={() => { setEditing(null); setShowNew(true); }}><Plus className="h-4 w-4" /> Lancamento</Button>
          </div>
        ) : undefined}
      />

      <Tabs
        tabs={[
          { key: "lancamentos", label: "Lancamentos", icon: <Wallet className="h-4 w-4" /> },
          { key: "recorrentes", label: "Recorrencias", icon: <Repeat className="h-4 w-4" /> },
          { key: "contas", label: "Contas Bancarias", icon: <Banknote className="h-4 w-4" /> },
          { key: "categorias", label: "Plano de Contas", icon: <Tags className="h-4 w-4" /> },
          { key: "grupos", label: "Grupos", icon: <FolderTree className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "lancamentos" && (
        <>
          {loading ? (
            <div className="grid gap-4 md:grid-cols-3">{[0, 1, 2].map((i) => <Skeleton key={i} className="h-24" />)}</div>
          ) : (
            <div className="mb-6 grid gap-4 md:grid-cols-3">
              <StatCard label="Entradas" value={balance ? currency(balance.income) : "..."} icon={ArrowUpRight} tone="green" />
              <StatCard label="Saidas" value={balance ? currency(balance.expense) : "..."} icon={ArrowDownRight} tone="red" />
              <StatCard label="Saldo" value={balance ? currency(balance.net) : "..."} icon={Wallet} tone="sky" />
            </div>
          )}

          {hasPerm("finance.write") && (
            <Card className="mb-6 p-4">
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Wallet className="h-4 w-4 text-sky-600" />
                <h3 className="text-sm font-semibold text-zinc-700">Lancamento rapido (em lote)</h3>
                <span className="text-xs text-zinc-400">Digite varias linhas e lance tudo de uma vez</span>
              </div>
              <BulkEntry embedded onSaved={handleTxnSaved} />
            </Card>
          )}

          <div className="mb-3 flex flex-wrap items-center gap-2">
            <div className="relative min-w-48 flex-1">
              <Filter className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
              <Input className="pl-9" placeholder="Buscar por descricao" value={query} onChange={(e) => setQuery(e.target.value)} />
            </div>
            <Select className="w-36" value={filterType} onChange={(e) => setFilterType(e.target.value)}>
              <option value="">Todos</option><option value="income">Entradas</option><option value="expense">Saidas</option>
            </Select>
            <Select className="w-44" value={filterCat} onChange={(e) => setFilterCat(e.target.value)}>
              <option value="">Todas as contas</option>
              {categories.map((c) => (<option key={c.id} value={c.id}>{c.name}</option>))}
            </Select>
            <Select className="w-44" value={filterAcct} onChange={(e) => setFilterAcct(e.target.value)}>
              <option value="">Todas contas</option>
              {accounts.map((a) => (<option key={a.id} value={a.id}>{a.name}</option>))}
            </Select>
          </div>

          <DataTable
            columns={columns}
            data={filtered}
            keyExtractor={(t) => t.id}
            loading={loading}
            emptyMessage="Sem lancamentos"
            emptyIcon={<Wallet className="h-10 w-10" />}
            pagination={{
              page,
              pageSize,
              total: filtered.length,
              onPageChange: setPage,
              onPageSizeChange: setPageSize,
              pageSizeOptions: [12, 25, 50],
            }}
            sort={{
              column: sortColumn,
              direction: sortDirection,
              onSort: handleSort,
            }}
            hoverable
            compact
          />
        </>
      )}

      {tab === "recorrentes" && <RecurringPanel />}

      {tab === "contas" && (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-zinc-100 p-4">
            <h3 className="text-sm font-semibold text-zinc-700">Contas bancarias</h3>
            {hasPerm("finance.write") && <Button size="sm" onClick={() => setShowAcctForm(true)}><Plus className="h-4 w-4" /> Conta</Button>}
          </div>
          <Table>
            <THead><TRow><TH>Nome</TH><TH>Banco</TH><TH>Agencia</TH><TH>Conta</TH><TH>Tipo</TH><TH className="text-right">Saldo Inicial</TH><TH>Ativa</TH><TH></TH></TRow></THead>
            <TBody>
              {accounts.map((a) => (
                <TRow key={a.id}>
                  <TD className="font-medium">{a.name}</TD>
                  <TD className="text-zinc-500">{a.bank ?? "-"}</TD>
                  <TD className="text-zinc-500">{a.agency ?? "-"}</TD>
                  <TD className="text-zinc-500">{a.account_number ?? "-"}</TD>
                  <TD className="text-zinc-500">{ACCOUNT_TYPES[a.account_type] ?? a.account_type}</TD>
                  <TD className="text-right text-zinc-500">{currency(a.initial_balance)}</TD>
                  <TD><Badge tone={a.is_active ? "green" : "zinc"}>{a.is_active ? "Sim" : "Nao"}</Badge></TD>
                  <TD className="text-center">
                    {hasPerm("finance.write") && (
                      <div className="flex justify-center gap-1">
                        <Button
                          variant="ghost" size="sm"
                          onClick={() => toggleAccountActive(a)}
                        >
                          {a.is_active ? "Desativar" : "Ativar"}
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => openAcctEdit(a)} aria-label="Editar conta" title="Editar">
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => deleteAcct(a)} aria-label="Excluir conta" title="Excluir">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
              {accounts.length === 0 && <TRow><TD colSpan={8} className="py-8 text-center text-zinc-400">Nenhuma conta cadastrada.</TD></TRow>}
            </TBody>
          </Table>
        </Card>
      )}

      {tab === "categorias" && (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-zinc-100 p-4">
            <h3 className="text-sm font-semibold text-zinc-700">Plano de contas</h3>
            {hasPerm("finance.write") && <Button size="sm" onClick={() => setShowCatForm(true)}><Plus className="h-4 w-4" /> Conta contabil</Button>}
          </div>
          <Table>
            <THead><TRow><TH>Nome</TH><TH>Codigo</TH><TH>Grupo</TH><TH>Tipo</TH><TH>Ativa</TH><TH></TH></TRow></THead>
            <TBody>
              {categories.map((c) => (
                <TRow key={c.id}>
                  <TD className="font-medium">{c.name}</TD>
                  <TD className="text-zinc-500">{c.code}</TD>
                  <TD className="text-zinc-500">{c.group_name ?? "-"}</TD>
                  <TD><Badge tone={c.type === "income" ? "green" : "red"}>{c.type === "income" ? "Entrada" : "Saida"}</Badge></TD>
                  <TD><Badge tone={c.is_active ? "green" : "zinc"}>{c.is_active ? "Sim" : "Nao"}</Badge></TD>
                  <TD className="text-center">
                    {hasPerm("finance.write") && (
                      <div className="flex justify-center gap-1">
                        <Button variant="ghost" size="sm" onClick={() => toggleCategoryActive(c)}>
                          {c.is_active ? "Desativar" : "Ativar"}
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => openCatEdit(c)} aria-label="Editar conta contabil" title="Editar">
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => deleteCat(c)} aria-label="Excluir conta contabil" title="Excluir">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
              {categories.length === 0 && <TRow><TD colSpan={6} className="py-8 text-center text-zinc-400">Nenhuma categoria.</TD></TRow>}
            </TBody>
          </Table>
        </Card>
      )}

      {tab === "grupos" && (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-zinc-100 p-4">
            <h3 className="text-sm font-semibold text-zinc-700">Grupos de contas</h3>
            {hasPerm("finance.write") && <Button size="sm" onClick={() => setShowGroupForm(true)}><Plus className="h-4 w-4" /> Grupo</Button>}
          </div>
          <Table>
            <THead><TRow><TH>Nome</TH><TH>Ordem</TH><TH>Ativo</TH><TH></TH></TRow></THead>
            <TBody>
              {groups.map((g) => (
                <TRow key={g.id}>
                  <TD className="font-medium">{g.name}</TD>
                  <TD className="text-zinc-500">{g.sort_order}</TD>
                  <TD><Badge tone={g.is_active ? "green" : "zinc"}>{g.is_active ? "Sim" : "Nao"}</Badge></TD>
                  <TD className="text-center">
                    {hasPerm("finance.write") && (
                      <div className="flex justify-center gap-1">
                        <Button variant="ghost" size="sm" onClick={() => toggleGroupActive(g)}>
                          {g.is_active ? "Desativar" : "Ativar"}
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => openGroupEdit(g)} aria-label="Editar grupo" title="Editar">
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-7 w-7 p-0" onClick={() => deleteGroup(g)} aria-label="Excluir grupo" title="Excluir">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
              {groups.length === 0 && <TRow><TD colSpan={4} className="py-8 text-center text-zinc-400">Nenhum grupo cadastrado.</TD></TRow>}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={!!detail} onClose={() => { setDetail(null); setAttachments([]); setDeliveries([]); setAllocations([]); }} title="Detalhe do lancamento">
        {detail && (
          <div className="space-y-4 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-zinc-500">Valor</span>
              <span className={`text-2xl font-semibold ${detail.type === "income" ? "text-emerald-600" : "text-red-600"}`}>{detail.type === "income" ? "+" : "-"}{currency(detail.amount)}</span>
            </div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Conta contabil</span><span>{detail.category_name ?? "-"}</span></div>
            {detail.account_name && <div className="flex items-center justify-between"><span className="text-zinc-500"> Conta</span><span>{detail.account_name}</span></div>}
            {detail.supplier_name && <div className="flex items-center justify-between"><span className="text-zinc-500">Fornecedor</span><span>{detail.supplier_name}</span></div>}
            <div className="flex items-center justify-between"><span className="text-zinc-500">Forma de pagamento</span><span>{detail.payment_method ? PAYMENT_METHODS[detail.payment_method] ?? detail.payment_method : "-"}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Data efetiva</span><span>{datePt(detail.occurred_at)}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Registrado em</span><span>{dateTimePt(detail.created_at)}</span></div>
            <div className="flex items-center justify-between"><span className="text-zinc-500">Recibo</span><span>{detail.receipt_ref ?? "-"}</span></div>
            <div className="flex items-start justify-between gap-4"><span className="text-zinc-500">Hash de integridade</span><span className="break-all text-xs text-zinc-400">{detail.hash}</span></div>

            {/* Anexos de comprovacao */}
            <div>
              <label className="text-sm font-semibold text-zinc-700">Anexos</label>
              <div className="mt-2 space-y-2">
                {attachments.length === 0 ? (
                  <p className="text-xs text-zinc-400">Nenhum anexo.</p>
                ) : (
                  attachments.map((att) => (
                    <div key={att.id} className="flex items-center justify-between rounded border border-zinc-200 p-2">
                      <div className="flex items-center gap-2">
                        <FileText className="h-4 w-4 text-zinc-500" />
                        <div>
                          <span className="text-sm">{att.file_name}</span>
                          <p className="text-xs text-zinc-400">{att.content_type} - {(att.file_size / 1024).toFixed(1)} KB</p>
                        </div>
                      </div>
                      <a href={att.file_url} target="_blank" rel="noopener noreferrer" aria-label={`Ver anexo ${att.file_name}`} title="Ver anexo" className="inline-flex items-center justify-center rounded px-2 py-1 text-xs hover:bg-zinc-100">
                        <Eye className="h-3.5 w-3.5" />
                      </a>
                    </div>
                  ))
                )}
               </div>
              {hasPerm("finance.write") && (
                <div className="mt-3">
                  <label className="flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-zinc-300 px-3 py-2 text-sm text-zinc-600 hover:border-zinc-400">
                    <Upload className="h-4 w-4" />
                    <span>Anexar comprovante</span>
                    <input
                      type="file"
                      accept="image/*,.pdf,.doc,.docx"
                      className="hidden"
                      onChange={(e) => {
                        const f = e.target.files?.[0];
                        if (f) attachFile(detail, f);
                        e.target.value = "";
                      }}
                    />
                  </label>
                </div>
              )}
            </div>

            {allocations.length > 0 && (
              <div>
                <label className="text-sm font-semibold text-zinc-700">Eventos (rateio)</label>
                <ul className="mt-2 space-y-1">
                  {allocations.map((a) => (
                    <li key={a.id} className="flex items-center justify-between rounded border border-zinc-200 px-2 py-1 text-sm dark:border-zinc-700">
                      <span className="truncate">{a.event_name}</span>
                      <span className="font-medium tabular-nums">{currency(a.amount)}</span>
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {detail.receipt_id && deliveries.length > 0 && (
              <div>
                <label className="text-sm font-semibold text-zinc-700">Envios do recibo</label>
                <div className="mt-2 space-y-1">
                  {deliveries.map((d) => (
                    <div key={d.id} className="flex items-center justify-between rounded border border-zinc-200 px-2 py-1.5">
                      <div className="flex items-center gap-2">
                        <Badge tone={d.status === "sent" ? "green" : d.status === "failed" ? "red" : "amber"} variant="soft">
                          {d.status === "sent" ? "Enviado" : d.status === "failed" ? "Falhou" : "Pendente"}
                        </Badge>
                        <span className="text-xs text-zinc-500">{d.channel === "email" ? "E-mail" : "WhatsApp"}</span>
                        <span className="text-xs text-zinc-400">{d.recipient}</span>
                      </div>
                      <span className="text-xs text-zinc-500">{d.sent_at ? dateTimePt(d.sent_at) : "-"}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="pt-2">
              {detail.receipt_id && (
                <div className="flex gap-2">
                  <Button variant="outline" className="flex-1" onClick={() => viewReceipt(detail)}><Eye className="h-4 w-4" /> Ver recibo</Button>
                  <Button onClick={() => openSendDrawer(detail, "whatsapp")}><Send className="h-4 w-4" /> Enviar</Button>
                </div>
              )}
            </div>
          </div>
        )}
      </Drawer>

      <Drawer open={showNew} onClose={() => { setShowNew(false); setEditing(null); }} size="xl" title={editing ? "Editar lancamento" : "Novo lancamento"}>
        <TransactionForm
          open={showNew}
          onClose={() => { setShowNew(false); setEditing(null); }}
          onSaved={handleTxnSaved}
          onSubmit={editing ? amendTxn : undefined}
          initialState={editing ? {
            type: editing.type,
            amount: String(editing.amount),
            category_id: editing.category_id ?? "",
            account_id: editing.account_id ?? "",
            payment_method: editing.payment_method ?? "",
            description: editing.description ?? "",
            occurred_at: editing.occurred_at.slice(0, 10),
            donor_member_id: editing.donor_member_id ?? "",
            benefactor_id: editing.benefactor_id ?? "",
            supplier_id: editing.supplier_id ?? "",
            is_anonymous: editing.is_anonymous,
          } : undefined}
          submitLabel={editing ? "Salvar alteracoes" : "Lancar"}
        />
      </Drawer>

      <Drawer open={showCatForm} onClose={closeCatForm} title={editingCat ? "Editar conta contabil" : "Nova conta contabil"}>
        <form onSubmit={submitCat} className="space-y-3">
          <Field label="Tipo">
            <Select value={catForm.type} onChange={(e) => setCatForm({ ...catForm, type: e.target.value })}>
              <option value="income">Entrada</option><option value="expense">Saida</option>
            </Select>
          </Field>
          <Field label="Codigo *"><Input required placeholder="ex.: 101" value={catForm.code} onChange={(e) => setCatForm({ ...catForm, code: e.target.value })} /></Field>
          <Field label="Nome *"><Input required value={catForm.name} onChange={(e) => setCatForm({ ...catForm, name: e.target.value })} /></Field>
          <Field label="Grupo">
            <Select value={catForm.group_id} onChange={(e) => setCatForm({ ...catForm, group_id: e.target.value })}>
              <option value="">Sem grupo</option>
              {groups.filter((g) => g.is_active).map((g) => (
                <option key={g.id} value={g.id}>{g.name}</option>
              ))}
            </Select>
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={closeCatForm}>Cancelar</Button>
            <Button type="submit">{editingCat ? "Salvar" : "Criar"}</Button>
          </div>
        </form>
      </Drawer>

      <Drawer open={showGroupForm} onClose={closeGroupForm} title={editingGroup ? "Editar grupo" : "Novo grupo de contas"}>
        <form onSubmit={submitGroup} className="space-y-3">
          <Field label="Nome *"><Input required placeholder="ex.: Receitas" value={groupForm.name} onChange={(e) => setGroupForm({ ...groupForm, name: e.target.value })} /></Field>
          <Field label="Ordem"><Input type="number" value={groupForm.sort_order} onChange={(e) => setGroupForm({ ...groupForm, sort_order: e.target.value })} placeholder="0" /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={closeGroupForm}>Cancelar</Button>
            <Button type="submit">{editingGroup ? "Salvar" : "Criar"}</Button>
          </div>
        </form>
      </Drawer>

      <Drawer open={showAcctForm} onClose={closeAcctForm} title={editingAcct ? "Editar conta bancaria" : "Nova conta bancaria"}>
        <form onSubmit={submitAcct} className="space-y-3">
          <Field label="Nome *"><Input required value={acctForm.name} onChange={(e) => setAcctForm({ ...acctForm, name: e.target.value })} /></Field>
          <Field label="Banco"><Input value={acctForm.bank} onChange={(e) => setAcctForm({ ...acctForm, bank: e.target.value })} placeholder="ex.: Banco do Brasil" /></Field>
          <Field label="Codigo do banco"><Input value={acctForm.bank_code} onChange={(e) => setAcctForm({ ...acctForm, bank_code: e.target.value })} placeholder="ex.: 1" /></Field>
          <Field label="Agencia"><Input value={acctForm.agency} onChange={(e) => setAcctForm({ ...acctForm, agency: e.target.value })} /></Field>
          <Field label="Numero da conta"><Input value={acctForm.account_number} onChange={(e) => setAcctForm({ ...acctForm, account_number: e.target.value })} /></Field>
          <Field label="Tipo">
            <Select value={acctForm.account_type} onChange={(e) => setAcctForm({ ...acctForm, account_type: e.target.value })}>
              {Object.entries(ACCOUNT_TYPES).map(([k, v]) => (<option key={k} value={k}>{v}</option>))}
            </Select>
          </Field>
          <Field label="Saldo inicial"><CurrencyInput value={acctForm.initial_balance} onChange={(v) => setAcctForm({ ...acctForm, initial_balance: v })} /></Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={closeAcctForm}>Cancelar</Button>
            <Button type="submit">{editingAcct ? "Salvar" : "Criar"}</Button>
          </div>
        </form>
      </Drawer>

      <Modal
        open={sendDrawer.open}
        onClose={() => setSendDrawer({ open: false, txn: null, channel: "email" })}
        title="Enviar recibo"
      >
        <div className="space-y-3 text-sm">
          <p>
            Enviar recibo <strong>{sendDrawer.channel === "email" ? "por e-mail" : "por WhatsApp"}</strong> para:
          </p>
          <Input
            type={sendDrawer.channel === "email" ? "email" : "tel"}
            placeholder={sendDrawer.channel === "email" ? "nome@exemplo.com" : "(11) 9 9999-9999"}
            value={recipient}
            onChange={(e) => setRecipient(e.target.value)}
            disabled={sending}
          />
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" size="sm" onClick={() => setSendDrawer({ open: false, txn: null, channel: "email" })}>Cancelar</Button>
            <Button size="sm" onClick={() => submitSend(recipient)} disabled={sending || !recipient.trim()}>{sending ? "Enviando..." : "Enviar"}</Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={showImport}
        onClose={() => setShowImport(false)}
        title="Importar lancamentos"
        size="xl"
      >
        <div className="space-y-3 text-sm">
          <p className="text-zinc-500">
            Envie uma planilha <b>.xlsx</b> ou <b>.csv</b>. Depois de mapear as colunas e escolher a
            <b> linha inicial</b> dos dados, clique em Importar.
          </p>
          <input
            type="file"
            accept=".csv,text/csv,.xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
            className="block w-full text-sm"
            onChange={async (e) => {
              const f = e.target.files?.[0];
              if (f) await pickImportFile(f);
              e.target.value = "";
            }}
          />

          {importRows.length > 0 && (
            <>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
                {IMPORT_FIELDS.map((f) => {
                  const maxCols = Math.max(...importRows.map((r) => r.length), 0);
                  return (
                    <div key={f.key}>
                      <label className="label">{f.label}{f.required && " *"}</label>
                      <Select
                        className="h-8 text-sm"
                        value={importMap[f.key] ?? -1}
                        onChange={(e) => setImportMap({ ...importMap, [f.key]: Number(e.target.value) })}
                      >
                        <option value={-1}>- nao usar -</option>
                        {Array.from({ length: maxCols }, (_, i) => (
                          <option key={i} value={i}>
                            {colLetter(i)}{importRows[0]?.[i] ? ` (${importRows[0][i]})` : ""}
                          </option>
                        ))}
                      </Select>
                    </div>
                  );
                })}
                <div>
                  <label className="label">Linha inicial (dados)</label>
                  <Input
                    type="number" min="1" className="h-8 text-sm"
                    value={importStartRow}
                    onChange={(e) => setImportStartRow(Math.max(1, Number(e.target.value) || 1))}
                  />
                </div>
              </div>

              <div className="max-h-64 overflow-auto rounded border border-zinc-200 dark:border-zinc-700">
                <table className="w-full text-xs">
                  <tbody>
                    {importRows.slice(0, 30).map((row, ri) => (
                      <tr key={ri} className={ri + 1 === importStartRow ? "bg-sky-50 dark:bg-sky-950/30" : ""}>
                        <td className="border-b border-zinc-100 px-2 py-1 text-right text-zinc-400 dark:border-zinc-800">{ri + 1}</td>
                        {row.map((cell, ci) => (
                          <td key={ci} className="whitespace-nowrap border-b border-zinc-100 px-2 py-1 dark:border-zinc-800">{cell}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}

          {importResult && (
            <div className="rounded border border-zinc-200 p-3 dark:border-zinc-700">
              <p className="font-medium">
                <Badge tone="green">{importResult.imported} importado(s)</Badge>{" "}
                {importResult.skipped > 0 && <Badge tone="amber">{importResult.skipped} ignorado(s)</Badge>}
              </p>
              {importResult.errors.length > 0 && (
                <ul className="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs text-red-600">
                  {importResult.errors.map((e, i) => (
                    <li key={i}>Linha {e.line}: {e.error}</li>
                  ))}
                </ul>
              )}
            </div>
          )}
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setShowImport(false)}>Fechar</Button>
            <Button type="button" onClick={doImport} disabled={importing || !importFile}>
              {importing ? "Importando..." : "Importar"}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
