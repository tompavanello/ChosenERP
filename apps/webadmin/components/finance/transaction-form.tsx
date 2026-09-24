"use client";

import { useState, useEffect } from "react";
import { Calendar, User, Users, Tag, CreditCard, FileText, Eye, Upload, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Field, Select, Textarea } from "@/components/ui/input";
import { Combobox } from "@/components/ui/combobox";
import { Section } from "@/components/ui/modal";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listCategories, listAccounts, listMembers, listBenefactors, listSuppliers, createTransaction, uploadAttachment, listEvents,
  type Category, type BankAccount, type Member, type Benefactor, type Supplier, type ChurchEvent,
} from "@/lib/api";
import { PAYMENT_METHODS, ACCOUNT_TYPES } from "@/lib/constants";
import { currency, datePt } from "@/lib/format";

export interface TransactionFormState {
  type: "income" | "expense";
  amount: string;
  category_id: string;
  account_id: string;
  payment_method: string;
  description: string;
  occurred_at: string;
  donor_member_id: string;
  benefactor_id: string;
  supplier_id: string;
  is_anonymous: boolean;
}

const EMPTY: TransactionFormState = {
  type: "income",
  amount: "",
  category_id: "",
  account_id: "",
  payment_method: "pix",
  description: "",
  occurred_at: new Date().toISOString().slice(0, 10),
  donor_member_id: "",
  benefactor_id: "",
  supplier_id: "",
  is_anonymous: false,
};

export interface TransactionFormProps {
  open: boolean;
  onClose: () => void;
  initialState?: Partial<TransactionFormState>;
  /** Se informado, o formulário chama onSubmit com os dados e deixa o
   *  componente pai decidir o que fazer (refetch, invalidação, etc.). */
  onSubmit?: (data: Record<string, unknown>) => Promise<void>;
  /** Se não informado, usa createTransaction e dá reload via onSaved. */
  onSaved?: () => void;
  submitLabel?: string;
}

/**
 * Formulário reutilizável para lançamentos financeiros.
 *
 * Extraído de dashboard/finance/page.tsx para permitir reuso em:
 *  - Drawer de "Novo lançamento" (página financeira)
 *  - Modal de "Lançamento rápido" (dashboard overview)
 *  - Página de transferência interna
 *
 * Campos expostos (não presentes no formulário original):
 *  - occurred_at  ? backdatar lançamentos
 *  - donor_member_id  ? associar a um membro
 *  - benefactor_id  ? associar a um benfeitor
 *  - is_anonymous  ? doação anônima
 */
export function TransactionForm({
  open,
  onClose,
  initialState,
  onSubmit,
  onSaved,
  submitLabel = "Lançar",
}: TransactionFormProps) {
  const { toast } = useToast();
  const { hasPerm } = useAuth();

  const [form, setForm] = useState<TransactionFormState>({ ...EMPTY, ...initialState });
  const [saving, setSaving] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [alloc, setAlloc] = useState<{ event_id: string; amount: string }[]>([]);
  const [tab, setTab] = useState<"dados" | "eventos">("dados");
  const [incomeCats, setIncomeCats] = useState<Category[]>([]);
  const [expenseCats, setExpenseCats] = useState<Category[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [benefactors, setBenefactors] = useState<Benefactor[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);

  // Carrega dependências para os selects/comboboxes
  useEffect(() => {
    if (!open) return;
    let cancelled = false;

    listCategories()
      .then((r) => {
        if (cancelled) return;
        const all = r.categories;
        setIncomeCats(all.filter((c) => c.type === "income"));
        setExpenseCats(all.filter((c) => c.type === "expense"));
      })
      .catch(() => { /* silencioso — usa estados vazios */ });

    listAccounts()
      .then((r) => { if (!cancelled) setAccounts(r.accounts.filter((a) => a.is_active)); })
      .catch(() => { if (!cancelled) setAccounts([]); });

    if (form.type === "income" || form.donor_member_id) {
      listMembers()
        .then((r) => { if (!cancelled) setMembers(r.members); })
        .catch(() => { if (!cancelled) setMembers([]); });
      listBenefactors()
        .then((r) => { if (!cancelled) setBenefactors(r.benefactors); })
        .catch(() => { if (!cancelled) setBenefactors([]); });
    }

    const y = new Date().getFullYear();
    listEvents({ from: `${y}-01-01`, to: `${y}-12-31` })
      .then((r) => { if (!cancelled) setEvents(r.events); })
      .catch(() => { if (!cancelled) setEvents([]); });

    listSuppliers()
      .then((r) => { if (!cancelled) setSuppliers(r.suppliers); })
      .catch(() => { if (!cancelled) setSuppliers([]); });

    return () => { cancelled = true; };
  }, [open, form.type, form.donor_member_id]);

  const set = (k: keyof TransactionFormState, v: string | boolean) =>
    setForm((f) => ({ ...f, [k]: v }));

  const categoriesForType = form.type === "income" ? incomeCats : expenseCats;

  // Validação client-side mínima: a conta contábil é obrigatória.
  const canSubmit =
    !!form.category_id && !!form.amount && Number(form.amount) > 0 &&
    (hasPerm("finance.write") || !!onSubmit);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit || saving) return;

    const amount = Number(form.amount);
    if (isNaN(amount) || amount <= 0) {
      toast("Valor deve ser um número positivo.", "error");
      return;
    }
    if (!form.category_id) {
      toast("Selecione a conta contábil.", "error");
      return;
    }

    const payload: Record<string, unknown> = {
      type: form.type,
      amount,
      category_id: form.category_id,
      account_id: form.account_id || undefined,
      payment_method: form.payment_method || undefined,
      description: form.description || undefined,
      occurred_at: form.occurred_at || undefined,
      donor_member_id: form.donor_member_id || undefined,
      benefactor_id: form.benefactor_id || undefined,
      supplier_id: form.supplier_id || undefined,
      is_anonymous: form.is_anonymous,
    };
    if (alloc.length > 0) {
      payload.event_allocations = alloc.map((a) => ({
        event_id: a.event_id,
        amount: a.amount === "" ? undefined : Number(a.amount),
      }));
    }

    setSaving(true);
    try {
      if (onSubmit) {
        await onSubmit(payload);
      } else {
        const res = await createTransaction(payload);
        // Anexo escolhido no próprio lançamento: sobe depois de criar (o
        // endpoint de anexo precisa do id da transação). Falha no anexo não
        // desfaz o lançamento.
        if (file && res.transaction?.id) {
          try {
            await uploadAttachment(res.transaction.id, file);
          } catch {
            toast("Lançamento registrado, mas o anexo falhou.", "error");
          }
        }
      }
      toast("Lançamento registrado com recibo.");
      onSaved?.();
      onClose();
      setForm({ ...EMPTY, ...initialState });
      setFile(null);
      setAlloc([]);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao registrar lançamento.", "error");
    } finally {
      setSaving(false);
    }
  }

  if (!open) return null;

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="mb-1 flex gap-1 border-b border-zinc-200 dark:border-zinc-800">
        <button
          type="button"
          onClick={() => setTab("dados")}
          className={`border-b-2 px-3 py-1.5 text-sm font-medium transition ${tab === "dados" ? "border-sky-600 text-sky-700 dark:text-sky-400" : "border-transparent text-zinc-500 hover:text-zinc-700"}`}
        >
          Dados
        </button>
        <button
          type="button"
          onClick={() => setTab("eventos")}
          className={`border-b-2 px-3 py-1.5 text-sm font-medium transition ${tab === "eventos" ? "border-sky-600 text-sky-700 dark:text-sky-400" : "border-transparent text-zinc-500 hover:text-zinc-700"}`}
        >
          Eventos (rateio){alloc.length > 0 ? ` (${alloc.length})` : ""}
        </button>
      </div>

      {tab === "dados" && (
      <>
      <Section title="Valores">
        <Field label="Tipo" required>
          <Select
            value={form.type}
            onChange={(e) => set("type", e.target.value as "income" | "expense")}
          >
            <option value="income">Dízimo / Oferta / Doação</option>
            <option value="expense">Despesa</option>
          </Select>
        </Field>
        <Field label="Valor (R$)*">
          <Input
            type="number"
            min="0.01"
            step="0.01"
            required
            placeholder="0,00"
            value={form.amount}
            onChange={(e) => set("amount", e.target.value)}
            disabled={saving}
          />
        </Field>
        <Field label="Data">
          <Input
            type="date"
            value={form.occurred_at}
            onChange={(e) => set("occurred_at", e.target.value)}
            disabled={saving}
          />
        </Field>
        <Field label="Forma de pagamento">
          <Select
            value={form.payment_method}
            onChange={(e) => set("payment_method", e.target.value)}
            disabled={saving}
          >
            {Object.entries(PAYMENT_METHODS).map(([k, v]) => (
              <option key={k} value={k}>{v}</option>
            ))}
          </Select>
        </Field>
      </Section>

      <Section title="Plano de contas">
        <Field label="Conta contábil *" required>
          <Select
            value={form.category_id}
            onChange={(e) => set("category_id", e.target.value)}
            disabled={saving || categoriesForType.length === 0}
          >
            {categoriesForType.length === 0 ? (
              <option value="">Carregando contas...</option>
            ) : (
              <>
                <option value="">Selecione...</option>
                {categoriesForType.map((c) => (
                  <option key={c.id} value={c.id}>{c.code} — {c.name}</option>
                ))}
              </>
            )}
          </Select>
        </Field>
        <div className="flex items-end">
          <Badge tone={form.is_anonymous ? "zinc" : "green"} variant="soft" className="text-xs">
            {form.is_anonymous ? "Anônimo" : "Identificado"}
          </Badge>
        </div>
      </Section>

      {form.type === "income" && (
        <Section title="Doação" hint="Associe a um membro ou benfeitor para emissão automática de recibo.">
          <Field label="Membro doador">
            <Combobox
              value={form.donor_member_id}
              placeholder="Buscar membro..."
              searchPlaceholder="Buscar membro..."
              emptyMessage="Nenhum membro encontrado"
              options={members.map((m) => ({ value: m.id, label: m.full_name }))}
              onChange={(v) => {
                set("donor_member_id", v);
                if (v) set("benefactor_id", "");
                if (!v) set("is_anonymous", false);
              }}
              className="w-full"
            />
          </Field>
          {!form.donor_member_id && (
            <Field label="Benfeitor">
              <Combobox
                value={form.benefactor_id}
                placeholder="Buscar benfeitor..."
                searchPlaceholder="Buscar benfeitor..."
                emptyMessage="Nenhum benfeitor encontrado"
                options={benefactors.map((b) => ({ value: b.id, label: b.name }))}
                onChange={(v) => {
                  set("benefactor_id", v);
                  if (v) set("donor_member_id", "");
                }}
                className="w-full"
              />
            </Field>
          )}
          {!form.donor_member_id && !form.benefactor_id && (
            <Field label="Anônimo">
              <div className="flex items-center gap-2 pt-1">
                <input
                  type="checkbox"
                  id="anon-toggle"
                  checked={form.is_anonymous}
                  onChange={(e) => set("is_anonymous", e.target.checked)}
                  disabled={saving}
                  className="h-4 w-4 text-sky-600 focus:ring-sky-500 border-zinc-300 rounded"
                />
                <label htmlFor="anon-toggle" className="text-sm text-zinc-600">
                  Doação sem identificação
                </label>
              </div>
            </Field>
          )}
        </Section>
      )}

      {form.type === "expense" && (
        <Section title="Fornecedor" hint="Opcional — associe a despesa a um fornecedor cadastrado.">
          <Field label="Fornecedor" className="sm:col-span-2">
            <Combobox
              value={form.supplier_id}
              placeholder="Buscar fornecedor..."
              searchPlaceholder="Buscar fornecedor..."
              emptyMessage="Nenhum fornecedor encontrado"
              options={suppliers.map((s) => ({ value: s.id, label: s.name }))}
              onChange={(v) => set("supplier_id", v)}
              className="w-full"
            />
          </Field>
        </Section>
      )}

      <Section title="Conta bancária">
        {accounts.length > 0 ? (
          <Field label="Conta">
            <Select
              value={form.account_id}
              onChange={(e) => set("account_id", e.target.value)}
              disabled={saving}
            >
              <option value="">Nenhuma (não informado)</option>
              {accounts.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name} — saldo inicial {currency(a.initial_balance)}
                </option>
              ))}
            </Select>
          </Field>
        ) : (
          <p className="text-xs text-zinc-400">
            Nenhuma conta bancária ativa cadastrada. Cadastre uma em{" "}
            <span className="text-sky-600">Contas Bancárias</span>.
          </p>
        )}
      </Section>

      <Section title="Observações">
        <Field label="Descrição" className="sm:col-span-2" hint={`${form.description.length}/2000`}>
          <Textarea
            rows={3}
            maxLength={2000}
            value={form.description}
            onChange={(e) => set("description", e.target.value)}
            placeholder="Ex.: Dízimo de dezembro, oferta de missão..."
            disabled={saving}
          />
        </Field>
        <Field label="Documento de referência" className="sm:col-span-2" hint="Opcional — comprovante, nota ou recibo (imagem/PDF).">
          {file ? (
            <div className="flex items-center justify-between rounded border border-zinc-200 px-2 py-1.5 text-sm dark:border-zinc-700">
              <span className="flex items-center gap-2 truncate">
                <FileText className="h-4 w-4 text-zinc-500" /> {file.name}
              </span>
              <button type="button" onClick={() => setFile(null)} aria-label="Remover anexo" className="text-zinc-400 hover:text-red-600">
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <label className="flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-zinc-300 px-3 py-2 text-sm text-zinc-600 hover:border-zinc-400 dark:border-zinc-700">
              <Upload className="h-4 w-4" /> <span>Escolher arquivo</span>
              <input
                type="file"
                accept="image/*,.pdf,.doc,.docx,.xls,.xlsx,.csv"
                className="hidden"
                disabled={saving}
                onChange={(e) => {
                  setFile(e.target.files?.[0] ?? null);
                  e.target.value = "";
                }}
              />
            </label>
          )}
        </Field>
      </Section>
      </>
      )}

      {tab === "eventos" && (
      <Section title="Eventos (rateio)" hint="Opcional — associe o custo a um ou mais eventos.">
        <div className="space-y-1 sm:col-span-2">
          {events.length === 0 ? (
            <p className="text-xs text-zinc-400">Nenhum evento cadastrado no ano.</p>
          ) : (
            events.map((ev) => {
              const sel = alloc.find((a) => a.event_id === ev.id);
              return (
                <div key={ev.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={!!sel}
                    disabled={saving}
                    onChange={(e) => {
                      if (e.target.checked) setAlloc([...alloc, { event_id: ev.id, amount: "" }]);
                      else setAlloc(alloc.filter((a) => a.event_id !== ev.id));
                    }}
                    className="h-4 w-4 rounded border-zinc-300 text-sky-600"
                  />
                  <span className="flex-1 truncate">{datePt(ev.starts_at)} · {ev.kind_name ?? "Evento"}</span>
                  {sel && (
                    <Input
                      type="number" min="0" step="0.01" placeholder="auto"
                      className="h-7 w-24 text-xs"
                      value={sel.amount}
                      disabled={saving}
                      onChange={(e) => setAlloc(alloc.map((a) => (a.event_id === ev.id ? { ...a, amount: e.target.value } : a)))}
                    />
                  )}
                </div>
              );
            })
          )}
          {alloc.length > 1 && (
            <Button type="button" variant="ghost" className="h-7 text-xs" onClick={() => setAlloc(alloc.map((a) => ({ ...a, amount: "" })))}>
              Dividir igualmente
            </Button>
          )}
        </div>
      </Section>
      )}

      <div className="flex justify-end gap-2 border-t border-zinc-100 pt-4">
        <Button
          variant="ghost"
          type="button"
          onClick={onClose}
          disabled={saving}
        >
          Cancelar
        </Button>
        <Button type="submit" disabled={!canSubmit || saving}>{saving ? " registrado..." : submitLabel}</Button>
      </div>
    </form>
  );
}
