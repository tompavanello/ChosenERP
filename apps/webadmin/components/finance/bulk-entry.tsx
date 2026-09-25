"use client";

import { Fragment, useEffect, useMemo, useState } from "react";
import {
  Plus, Trash2, Copy, Check, AlertCircle, Paperclip, X, SlidersHorizontal, ChevronUp,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { Combobox } from "@/components/ui/combobox";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listCategories, listAccounts, listMembers, listBenefactors, listSuppliers, listEvents,
  createTransactionsBatch, uploadAttachment,
  type Category, type BankAccount, type Member, type Benefactor, type Supplier, type ChurchEvent,
  type ImportResult,
} from "@/lib/api";
import { PAYMENT_METHODS } from "@/lib/constants";
import { currency, todayISO } from "@/lib/format";

// Linha editavel do grid. `key` e apenas local (nao vai para o backend).
// O `type` (entrada/saida) e DERIVADO da conta contabil escolhida.
interface BulkRow {
  key: number;
  type: "income" | "expense";
  occurred_at: string;
  amount: string;
  category_id: string;
  description: string;
  account_id: string;
  payment_method: string;
  donor_member_id: string;
  benefactor_id: string;
  supplier_id: string;
  is_anonymous: boolean;
  event_ids: string[];
  file: File | null;
  expanded: boolean;
}

const today = () => todayISO();

let seq = 0;
function emptyRow(): BulkRow {
  seq += 1;
  return {
    key: seq,
    type: "income",
    occurred_at: "",
    amount: "",
    category_id: "",
    description: "",
    account_id: "",
    payment_method: "pix",
    donor_member_id: "",
    benefactor_id: "",
    supplier_id: "",
    is_anonymous: false,
    event_ids: [],
    file: null,
    expanded: false,
  };
}

function SubField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1">
      <span className="block text-[11px] font-medium uppercase tracking-wide text-zinc-500">{label}</span>
      {children}
    </div>
  );
}

/**
 * Grid dinamico de lancamentos: cada linha e um lancamento. Conta corrente e data
 * padrao do lote sao aplicadas a todas as linhas (ajustaveis por linha). O tipo
 * (entrada/saida) vem da conta contabil; a identificacao do doador fica na linha
 * e o restante no complemento. Salva tudo em uma unica chamada
 * (POST /finance/transactions/batch); anexos sobem em seguida.
 */
export function BulkEntry({
  onSaved,
  onClose,
  embedded = false,
}: {
  onSaved: () => void;
  onClose?: () => void;
  /** Embutido na propria tela (sem botao Fechar e sem a introducao). */
  embedded?: boolean;
}) {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const [rows, setRows] = useState<BulkRow[]>(() => [emptyRow(), emptyRow(), emptyRow()]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [accounts, setAccounts] = useState<BankAccount[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [benefactors, setBenefactors] = useState<Benefactor[]>([]);
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [saving, setSaving] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  // Padroes do lote: aplicados a toda linha que nao tiver valor proprio.
  const [defaultAccountId, setDefaultAccountId] = useState("");
  const [defaultDate, setDefaultDate] = useState(today());

  useEffect(() => {
    listCategories()
      .then((r) => setCategories(r.categories.filter((c) => c.is_active)))
      .catch(() => {});
    listAccounts()
      .then((r) => setAccounts(r.accounts.filter((a) => a.is_active)))
      .catch(() => setAccounts([]));
    listMembers()
      .then((r) => setMembers(r.members))
      .catch(() => setMembers([]));
    listBenefactors()
      .then((r) => setBenefactors(r.benefactors))
      .catch(() => setBenefactors([]));
    listSuppliers()
      .then((r) => setSuppliers(r.suppliers))
      .catch(() => setSuppliers([]));
    const y = new Date().getFullYear();
    listEvents({ from: `${y}-01-01`, to: `${y}-12-31` })
      .then((r) => setEvents(r.events))
      .catch(() => setEvents([]));
  }, []);

  const update = (key: number, patch: Partial<BulkRow>) =>
    setRows((rs) => rs.map((r) => (r.key === key ? { ...r, ...patch } : r)));

  // Tipo (entrada/saida) derivado da conta contabil da linha.
  const catType = (id: string): BulkRow["type"] | "" =>
    categories.find((c) => c.id === id)?.type ?? "";

  function changeCategory(r: BulkRow, id: string) {
    const t = catType(id);
    update(r.key, {
      category_id: id,
      type: t || "income",
      ...(t === "expense" ? { donor_member_id: "", benefactor_id: "", is_anonymous: false } : {}),
    });
  }

  function addRow(copyOf?: BulkRow) {
    setRows((rs) => [
      ...rs,
      copyOf
        ? { ...copyOf, key: ++seq, file: null, expanded: false }
        : emptyRow(),
    ]);
  }

  // Linhas "tocadas": tem conta contabil ou valor. Vao para o backend (que
  // valida e devolve erro por linha).
  const touched = useMemo(
    () => rows.filter((r) => r.category_id || r.amount.trim() !== ""),
    [rows],
  );

  const ready = useMemo(
    () => touched.filter((r) => r.category_id && Number(r.amount) > 0),
    [touched],
  );

  const totals = useMemo(() => {
    let income = 0;
    let expense = 0;
    for (const r of ready) {
      const v = Number(r.amount);
      if (r.type === "income") income += v;
      else expense += v;
    }
    return { income, expense };
  }, [ready]);

  function complementCount(r: BulkRow): number {
    return [
      r.account_id || defaultAccountId, r.payment_method,
      r.benefactor_id, r.supplier_id, r.donor_member_id,
    ].filter(Boolean).length + (r.event_ids.length > 0 ? 1 : 0);
  }

  async function submit() {
    if (!hasPerm("finance.write")) return;
    if (touched.length === 0) {
      toast("Preencha ao menos uma linha (conta contabil e valor).", "error");
      return;
    }
    const payloads = touched.map((r) => ({
      type: r.type,
      amount: Number(r.amount),
      category_id: r.category_id || undefined,
      account_id: r.account_id || defaultAccountId || undefined,
      payment_method: r.payment_method || undefined,
      description: r.description || undefined,
      occurred_at: r.occurred_at || defaultDate || undefined,
      donor_member_id: r.donor_member_id || undefined,
      benefactor_id: r.benefactor_id || undefined,
      supplier_id: r.supplier_id || undefined,
      is_anonymous: r.is_anonymous,
      event_allocations: r.event_ids.length
        ? r.event_ids.map((event_id) => ({ event_id }))
        : undefined,
    }));

    setSaving(true);
    setResult(null);
    try {
      const res = await createTransactionsBatch(payloads);
      const ids = res.created_ids ?? [];

      // Anexos por linha: sobe para o lancamento criado correspondente.
      let attFail = 0;
      for (let i = 0; i < touched.length; i++) {
        const f = touched[i].file;
        const id = ids[i];
        if (f && id) {
          try {
            await uploadAttachment(id, f);
          } catch {
            attFail++;
          }
        }
      }

      setResult(res);
      if (res.imported > 0) {
        toast(
          attFail > 0
            ? `${res.imported} lancamento(s) registrado(s); ${attFail} anexo(s) falharam.`
            : `${res.imported} lancamento(s) registrado(s).`,
        );
        onSaved();
      } else {
        toast("Nenhum lancamento foi registrado.", "error");
      }

      if (res.skipped === 0) {
        setRows([emptyRow(), emptyRow(), emptyRow()]);
      } else {
        // Mantem apenas as linhas que falharam, para correcao.
        const failed = touched.filter((_, i) => !ids[i]);
        setRows(failed.length > 0 ? failed : [emptyRow(), emptyRow(), emptyRow()]);
      }
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao lancar em lote", "error");
    } finally {
      setSaving(false);
    }
  }

  const incomeCats = useMemo(() => categories.filter((c) => c.type === "income"), [categories]);
  const expenseCats = useMemo(() => categories.filter((c) => c.type === "expense"), [categories]);

  return (
    <div className="space-y-3">
      {!embedded && (
        <p className="text-sm text-zinc-500">
          Cada linha e um lancamento. O tipo (entrada/saida) vem da conta contabil. Conta corrente e
          data padrao valem para o lote inteiro; use o complemento para ajustar a linha.
        </p>
      )}

      <div className="flex flex-wrap items-end gap-3">
        <div className="w-full sm:w-64">
          <span className="block text-[11px] font-medium uppercase tracking-wide text-zinc-500">Conta corrente padrao do lote</span>
          <Select
            className="mt-1 h-8 text-xs"
            value={defaultAccountId}
            onChange={(e) => setDefaultAccountId(e.target.value)}
          >
            <option value="">Nenhuma (definir por linha)</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>{a.name}</option>
            ))}
          </Select>
        </div>
        <div className="w-full sm:w-44">
          <span className="block text-[11px] font-medium uppercase tracking-wide text-zinc-500">Data padrao do lote</span>
          <Input
            type="date"
            className="mt-1 h-8 text-xs"
            value={defaultDate}
            onChange={(e) => setDefaultDate(e.target.value)}
          />
        </div>
        <p className="pb-1.5 text-xs text-zinc-500">
          Aplicadas a todas as linhas; se algum lancamento for diferente, ajuste na propria linha.
        </p>
      </div>

      <div className="overflow-x-auto rounded-lg border border-zinc-200 dark:border-zinc-800">
        <table className="w-full min-w-[1000px] text-[13px]">
          <thead className="border-b border-zinc-200 bg-zinc-50 text-[10px] uppercase tracking-wider text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900">
            <tr>
              <th className="w-8 px-2 py-2 text-left">#</th>
              <th className="w-32 px-2 py-2 text-left">Valor</th>
              <th className="w-64 px-2 py-2 text-left">Conta contabil</th>
              <th className="px-2 py-2 text-left">Descricao</th>
              <th className="w-64 px-2 py-2 text-left">Identificado</th>
              <th className="w-24 px-2 py-2 text-left">Anexo</th>
              <th className="w-14 px-2 py-2"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800/70">
            {rows.map((r, i) => {
              const eff = catType(r.category_id) || r.type;
              const tone = eff === "income" ? "text-emerald-600" : "text-red-600";
              return (
                <Fragment key={r.key}>
                  <tr className={r.expanded ? "bg-sky-50/40 dark:bg-sky-950/10" : undefined}>
                    <td className="px-2 py-1 text-right text-zinc-400">{i + 1}</td>
                    <td className="px-2 py-1">
                      <CurrencyInput
                        className={`h-8 text-xs ${tone}`}
                        value={r.amount}
                        onChange={(v) => update(r.key, { amount: v })}
                      />
                    </td>
                    <td className="px-2 py-1">
                      <Select
                        className="h-8 text-xs font-medium"
                        style={{ color: eff === "income" ? "#059669" : "#dc2626" }}
                        value={r.category_id}
                        onChange={(e) => changeCategory(r, e.target.value)}
                      >
                        <option value="">Selecione...</option>
                        <optgroup label="Entradas" style={{ color: "#71717a" }}>
                          {incomeCats.map((c) => (
                            <option key={c.id} value={c.id} style={{ color: "#059669" }}>{c.code} - {c.name}</option>
                          ))}
                        </optgroup>
                        <optgroup label="Saidas" style={{ color: "#71717a" }}>
                          {expenseCats.map((c) => (
                            <option key={c.id} value={c.id} style={{ color: "#dc2626" }}>{c.code} - {c.name}</option>
                          ))}
                        </optgroup>
                      </Select>
                    </td>
                    <td className="px-2 py-1">
                      <Input
                        className="h-8 text-xs"
                        placeholder="Descricao"
                        value={r.description}
                        onChange={(e) => update(r.key, { description: e.target.value })}
                      />
                    </td>
                    <td className="px-2 py-1">
                      {eff === "income" ? (
                        <div className="flex items-center gap-2">
                          <label
                            className="flex shrink-0 items-center gap-1 whitespace-nowrap text-[11px] text-zinc-500"
                            title="Marque para identificar o doador (membro)"
                          >
                            <input
                              type="checkbox"
                              className="h-3.5 w-3.5 rounded border-zinc-300 text-sky-600"
                              checked={!r.is_anonymous}
                              onChange={(e) =>
                                update(r.key, {
                                  is_anonymous: !e.target.checked,
                                  donor_member_id: e.target.checked ? r.donor_member_id : "",
                                })
                              }
                            />
                            Identificado
                          </label>
                          {!r.is_anonymous && (
                            <Combobox
                              value={r.donor_member_id}
                              placeholder="Buscar membro..."
                              searchPlaceholder="Buscar membro..."
                              emptyMessage="Nenhum membro encontrado"
                              options={members.map((m) => ({ value: m.id, label: m.full_name }))}
                              onChange={(v) => update(r.key, { donor_member_id: v, benefactor_id: v ? "" : r.benefactor_id })}
                              className="w-44"
                            />
                          )}
                        </div>
                      ) : (
                        <span className="text-xs text-zinc-400">-</span>
                      )}
                    </td>
                    <td className="px-2 py-1">
                      {r.file ? (
                        <div className="flex items-center gap-1 rounded border border-zinc-200 px-1.5 py-1 text-xs dark:border-zinc-700">
                          <Paperclip className="h-3.5 w-3.5 shrink-0 text-zinc-400" />
                          <span className="max-w-[48px] truncate" title={r.file.name}>{r.file.name}</span>
                          <button
                            type="button"
                            title="Remover anexo"
                            onClick={() => update(r.key, { file: null })}
                            className="text-zinc-400 hover:text-red-600"
                          >
                            <X className="h-3 w-3" />
                          </button>
                        </div>
                      ) : (
                        <label
                          className="flex cursor-pointer items-center justify-center rounded border border-dashed border-zinc-300 px-1.5 py-1 text-zinc-400 hover:border-zinc-400 dark:border-zinc-700"
                          title="Anexar comprovante"
                        >
                          <Paperclip className="h-3.5 w-3.5" />
                          <input
                            type="file"
                            accept="image/*,.pdf,.doc,.docx,.xls,.xlsx,.csv"
                            className="hidden"
                            onChange={(e) => {
                              const f = e.target.files?.[0] ?? null;
                              update(r.key, { file: f });
                              e.target.value = "";
                            }}
                          />
                        </label>
                      )}
                    </td>
                    <td className="px-2 py-1">
                      <div className="flex items-center justify-center gap-0.5">
                        <button
                          type="button"
                          title="Complemento do lancamento"
                          onClick={() => update(r.key, { expanded: !r.expanded })}
                          className={`relative rounded p-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 ${r.expanded ? "text-sky-600" : "text-zinc-400"}`}
                        >
                          <SlidersHorizontal className="h-4 w-4" />
                          {complementCount(r) > 0 && (
                            <span className="absolute -right-0.5 -top-0.5 flex h-3.5 w-3.5 items-center justify-center rounded-full bg-sky-600 text-[9px] font-bold text-white">
                              {complementCount(r)}
                            </span>
                          )}
                        </button>
                        <button
                          type="button"
                          title="Duplicar linha"
                          onClick={() => addRow(r)}
                          className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-sky-600 dark:hover:bg-zinc-800"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          title="Remover linha"
                          onClick={() => setRows((rs) => (rs.length > 1 ? rs.filter((x) => x.key !== r.key) : rs))}
                          className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-red-600 dark:hover:bg-zinc-800"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>

                  {r.expanded && (
                    <tr key={`${r.key}-comp`} className="bg-zinc-50/70 dark:bg-zinc-900/40">
                      <td colSpan={7} className="px-3 py-3">
                        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                          <SubField label="Data efetiva">
                            <Input
                              type="date"
                              className="h-8 text-xs"
                              value={r.occurred_at || defaultDate}
                              onChange={(e) => update(r.key, { occurred_at: e.target.value })}
                            />
                          </SubField>
                          <SubField label="Forma de pagamento">
                            <Select
                              className="h-8 text-xs"
                              value={r.payment_method}
                              onChange={(e) => update(r.key, { payment_method: e.target.value })}
                            >
                              <option value="">-</option>
                              {Object.entries(PAYMENT_METHODS).map(([k, v]) => (
                                <option key={k} value={k}>{v}</option>
                              ))}
                            </Select>
                          </SubField>
                          <SubField label="Conta bancaria">
                            <Select
                              className="h-8 text-xs"
                              value={r.account_id || defaultAccountId}
                              onChange={(e) => update(r.key, { account_id: e.target.value })}
                            >
                              <option value="">-</option>
                              {accounts.map((a) => (
                                <option key={a.id} value={a.id}>{a.name}</option>
                              ))}
                            </Select>
                          </SubField>
                          {eff === "income" && (
                            <SubField label="Benfeitor">
                              <Select
                                className="h-8 text-xs"
                                value={r.benefactor_id}
                                onChange={(e) => {
                                  const v = e.target.value;
                                  update(r.key, {
                                    benefactor_id: v,
                                    donor_member_id: v ? "" : r.donor_member_id,
                                  });
                                }}
                              >
                                <option value="">-</option>
                                {benefactors.map((b) => (
                                  <option key={b.id} value={b.id}>{b.name}</option>
                                ))}
                              </Select>
                            </SubField>
                          )}
                          {eff === "expense" && (
                            <SubField label="Fornecedor">
                              <Select
                                className="h-8 text-xs"
                                value={r.supplier_id}
                                onChange={(e) => update(r.key, { supplier_id: e.target.value })}
                              >
                                <option value="">-</option>
                                {suppliers.map((s) => (
                                  <option key={s.id} value={s.id}>{s.name}</option>
                                ))}
                              </Select>
                            </SubField>
                          )}
                          <div className="sm:col-span-2 lg:col-span-3">
                            <SubField label="Eventos (rateio)">
                              {events.length === 0 ? (
                                <p className="text-xs text-zinc-400">Nenhum evento no ano.</p>
                              ) : (
                                <div className="flex max-h-24 flex-wrap gap-x-4 gap-y-1 overflow-y-auto rounded border border-zinc-200 p-2 dark:border-zinc-800">
                                  {events.map((ev) => {
                                    const checked = r.event_ids.includes(ev.id);
                                    return (
                                      <label key={ev.id} className="flex items-center gap-1.5 whitespace-nowrap text-xs text-zinc-600 dark:text-zinc-300">
                                        <input
                                          type="checkbox"
                                          checked={checked}
                                          onChange={(e) => update(r.key, {
                                            event_ids: e.target.checked
                                              ? [...r.event_ids, ev.id]
                                              : r.event_ids.filter((id) => id !== ev.id),
                                          })}
                                          className="h-3.5 w-3.5 rounded border-zinc-300 text-sky-600"
                                        />
                                        {ev.kind_name ? `${ev.kind_name} - ` : ""}{ev.starts_at.slice(0, 10)}
                                      </label>
                                    );
                                  })}
                                </div>
                              )}
                            </SubField>
                          </div>
                        </div>
                        <div className="mt-2 flex justify-end">
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="h-7 text-xs"
                            onClick={() => update(r.key, { expanded: false })}
                          >
                            <ChevronUp className="h-3.5 w-3.5" /> Recolher
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )}
                </Fragment>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" variant="outline" size="sm" onClick={() => addRow()}>
          <Plus className="h-4 w-4" /> Adicionar linha
        </Button>
        <button
          type="button"
          onClick={() => setRows((rs) => [...rs, ...Array.from({ length: 5 }, () => emptyRow())])}
          className="text-xs text-sky-600 hover:underline"
        >
          +5 linhas
        </button>
        <div className="ml-auto flex items-center gap-3 text-xs text-zinc-500">
          <span>{ready.length} linha(s) pronta(s)</span>
          <Badge tone="green">Entradas {currency(totals.income)}</Badge>
          <Badge tone="red">Saidas {currency(totals.expense)}</Badge>
        </div>
      </div>

      {result && (
        <div className="rounded border border-zinc-200 p-3 text-sm dark:border-zinc-700">
          <p className="flex items-center gap-2 font-medium">
            {result.imported > 0 ? <Check className="h-4 w-4 text-emerald-600" /> : <AlertCircle className="h-4 w-4 text-amber-500" />}
            <Badge tone="green">{result.imported} registrado(s)</Badge>
            {result.skipped > 0 && <Badge tone="amber">{result.skipped} com erro</Badge>}
          </p>
          {result.errors.length > 0 && (
            <ul className="mt-2 max-h-32 space-y-1 overflow-y-auto text-xs text-red-600">
              {result.errors.map((e, i) => (
                <li key={i}>Linha {e.line}: {e.error}</li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div className="flex flex-wrap justify-end gap-2 border-t border-zinc-100 pt-3">
        <Button
          type="button"
          variant="ghost"
          onClick={() => { setRows([emptyRow(), emptyRow(), emptyRow()]); setResult(null); }}
          disabled={saving}
        >
          Limpar
        </Button>
        {!embedded && onClose && (
          <Button type="button" variant="ghost" onClick={onClose} disabled={saving}>Fechar</Button>
        )}
        <Button type="button" onClick={submit} disabled={saving || !hasPerm("finance.write")}>
          {saving ? "Lancando..." : `Lancar ${ready.length || ""}`.trim()}
        </Button>
      </div>
    </div>
  );
}
