"use client";

import { useCallback, useEffect, useState } from "react";
import { CreditCard, RefreshCw, Receipt, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { currency, datePt } from "@/lib/format";
import {
  getTenantBilling,
  subscribeTenant,
  cancelTenantSubscription,
  syncTenantBilling,
  chargeTenant,
  refundInvoice,
  listPlans,
  type TenantBilling,
  type Plan,
  type SubscribeInput,
} from "@/lib/api";

const STATUS_TONE: Record<string, "green" | "red" | "amber" | "zinc" | "sky"> = {
  active: "green",
  paid: "green",
  pending: "amber",
  overdue: "red",
  canceled: "zinc",
  refunded: "sky",
  suspended: "red",
};

const STATUS_LABEL: Record<string, string> = {
  active: "Ativa",
  paid: "Pago",
  pending: "Pendente",
  overdue: "Vencida",
  canceled: "Cancelada",
  refunded: "Estornada",
  suspended: "Suspensa",
  none: "Sem assinatura",
};

type SubForm = {
  plan_key: string;
  billing_type: string;
  cycle: string;
  cpf_cnpj: string;
  email: string;
  phone: string;
  next_due_date: string;
  value: string;
  holder_name: string;
  number: string;
  expiry_month: string;
  expiry_year: string;
  ccv: string;
  holder_cpf: string;
  postal_code: string;
  address_number: string;
};

const EMPTY: SubForm = {
  plan_key: "", billing_type: "PIX", cycle: "MONTHLY", cpf_cnpj: "", email: "", phone: "",
  next_due_date: "", value: "",
  holder_name: "", number: "", expiry_month: "", expiry_year: "", ccv: "",
  holder_cpf: "", postal_code: "", address_number: "",
};

export function TenantBilling({ tenantId }: { tenantId: string }) {
  const { toast } = useToast();
  const [data, setData] = useState<TenantBilling | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<SubForm>({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [tb, p] = await Promise.all([getTenantBilling(tenantId), listPlans()]);
      setData(tb);
      setPlans(p.plans);
      setForm({
        ...EMPTY,
        plan_key: tb.subscription?.plan_key ?? tb.plan,
        billing_type: tb.subscription?.billing_type ?? "PIX",
        cycle: tb.subscription?.cycle ?? "MONTHLY",
        email: tb.customer?.email ?? "",
        phone: tb.customer?.phone ?? "",
        cpf_cnpj: tb.customer?.cpf_cnpj ?? "",
      });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar cobranca da igreja", "error");
      setData(null);
    }
  }, [tenantId, toast]);

  useEffect(() => { load(); }, [load]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: SubscribeInput = {
        plan_key: form.plan_key,
        billing_type: form.billing_type,
        cycle: form.cycle,
        cpf_cnpj: form.cpf_cnpj,
        email: form.email,
        phone: form.phone,
        next_due_date: form.next_due_date || undefined,
      };
      if (form.value) payload.value_cents = Math.round(Number(form.value) * 100);
      if (form.billing_type === "CREDIT_CARD") {
        payload.card = {
          holder_name: form.holder_name,
          number: form.number,
          expiry_month: form.expiry_month,
          expiry_year: form.expiry_year,
          ccv: form.ccv,
          holder_cpf: form.holder_cpf,
          postal_code: form.postal_code,
          address_number: form.address_number,
        };
      }
      const tb = await subscribeTenant(tenantId, payload);
      setData(tb);
      toast("Assinatura atualizada");
      setOpen(false);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao assinar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function cancelSub() {
    if (!confirm("Cancelar a assinatura desta igreja? As cobrancas futuras serao interrompidas.")) return;
    try {
      await cancelTenantSubscription(tenantId);
      toast("Assinatura cancelada");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao cancelar", "error");
    }
  }

  async function sync() {
    try {
      const tb = await syncTenantBilling(tenantId);
      setData(tb);
      toast("Cobranca sincronizada");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao sincronizar", "error");
    }
  }

  async function refund(inv: { id: string }) {
    if (!confirm("Estornar esta fatura no gateway?")) return;
    try {
      await refundInvoice(tenantId, inv.id);
      toast("Fatura estornada");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao estornar", "error");
    }
  }

  if (data === null) {
    return <div className="p-4"><SkeletonRows rows={4} /></div>;
  }

  const sub = data.subscription;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Assinatura</CardTitle>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={sync}><RefreshCw className="h-3.5 w-3.5" /> Sincronizar</Button>
            <Button variant="outline" size="sm" onClick={() => setOpen(true)}>
              <CreditCard className="h-3.5 w-3.5" /> {sub ? "Alterar" : "Assinar"}
            </Button>
            {sub && (
              <Button variant="outline" size="sm" onClick={cancelSub}><XCircle className="h-3.5 w-3.5" /> Cancelar</Button>
            )}
          </div>
        </CardHeader>
        <CardContent className="grid grid-cols-2 gap-4 text-sm md:grid-cols-4">
          <Info label="Situacao" value={<Badge tone={STATUS_TONE[data.billing_status] ?? "zinc"}>{STATUS_LABEL[data.billing_status] ?? data.billing_status}</Badge>} />
          <Info label="Plano" value={data.plan} />
          <Info label="Valor" value={sub ? currency(sub.value_cents / 100) : currency((plans.find((p) => p.key === data.plan)?.price_cents ?? 0) / 100)} />
          <Info label="Forma" value={sub?.billing_type ?? "-"} />
          <Info label="Ciclo" value={sub ? (sub.cycle === "YEARLY" ? "Anual" : "Mensal") : "-"} />
          <Info label="Prox. vencimento" value={sub?.next_due_date ? datePt(sub.next_due_date) : "-"} />
          <Info label="Isenta" value={data.billing_exempt ? "Sim" : "Nao"} />
          <Info label="Inadimplente desde" value={data.past_due_since ? datePt(data.past_due_since) : "-"} />
        </CardContent>
      </Card>

      <Card className="overflow-hidden p-0">
        <CardHeader className="px-4 pt-4"><CardTitle>Faturas</CardTitle></CardHeader>
        {data.invoices.length === 0 ? (
          <EmptyState icon={<Receipt className="h-10 w-10" />} title="Sem faturas" description="As faturas aparecem quando a assinatura gera cobrancas." />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Status</TH><TH className="text-right">Valor</TH><TH>Forma</TH><TH>Vencimento</TH><TH>Pago em</TH><TH></TH></TRow>
            </THead>
            <TBody>
              {data.invoices.map((v) => (
                <TRow key={v.id}>
                  <TD><Badge tone={STATUS_TONE[v.status] ?? "zinc"}>{STATUS_LABEL[v.status] ?? v.status}</Badge></TD>
                  <TD className="text-right tabular-nums">{currency(v.value_cents / 100)}</TD>
                  <TD>{v.billing_type || "-"}</TD>
                  <TD>{v.due_date ? datePt(v.due_date) : "-"}</TD>
                  <TD>{v.paid_at ? datePt(v.paid_at) : "-"}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      {v.invoice_url && <a className="btn-base btn-outline h-7 px-2 text-xs" href={v.invoice_url} target="_blank" rel="noreferrer">Ver</a>}
                      {v.status === "paid" && (
                        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => refund(v)}>Estornar</Button>
                      )}
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title={sub ? "Alterar assinatura" : "Nova assinatura"}>
        <form onSubmit={submit} className="space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <Field label="Plano *">
              <Select className="h-8 text-sm" value={form.plan_key} onChange={(e) => setForm({ ...form, plan_key: e.target.value })}>
                <option value="">Selecione...</option>
                {plans.map((p) => <option key={p.key} value={p.key}>{p.name}</option>)}
              </Select>
            </Field>
            <Field label="Forma de pagamento *">
              <Select className="h-8 text-sm" value={form.billing_type} onChange={(e) => setForm({ ...form, billing_type: e.target.value })}>
                <option value="PIX">PIX</option>
                <option value="BOLETO">Boleto</option>
                <option value="CREDIT_CARD">Cartao de credito</option>
              </Select>
            </Field>
            <Field label="Ciclo">
              <Select className="h-8 text-sm" value={form.cycle} onChange={(e) => setForm({ ...form, cycle: e.target.value })}>
                <option value="MONTHLY">Mensal</option>
                <option value="YEARLY">Anual</option>
              </Select>
            </Field>
            <Field label="Proximo vencimento">
              <Input type="date" className="h-8 text-sm" value={form.next_due_date} onChange={(e) => setForm({ ...form, next_due_date: e.target.value })} />
            </Field>
            <Field label="CPF/CNPJ">
              <Input className="h-8 text-sm" value={form.cpf_cnpj} onChange={(e) => setForm({ ...form, cpf_cnpj: e.target.value })} />
            </Field>
            <Field label="E-mail">
              <Input type="email" className="h-8 text-sm" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
            </Field>
            <Field label="Telefone">
              <Input className="h-8 text-sm" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
            </Field>
            <Field label="Valor (opcional)" hint="Vazio usa o preco do plano">
              <CurrencyInput className="h-8 text-sm" value={form.value} onChange={(v) => setForm({ ...form, value: v })} />
            </Field>
          </div>

          {form.billing_type === "CREDIT_CARD" && (
            <div className="space-y-3 rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Cartao de credito</p>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Titular *" className="col-span-2">
                  <Input className="h-8 text-sm" value={form.holder_name} onChange={(e) => setForm({ ...form, holder_name: e.target.value })} />
                </Field>
                <Field label="Numero *" className="col-span-2">
                  <Input className="h-8 text-sm" value={form.number} onChange={(e) => setForm({ ...form, number: e.target.value })} />
                </Field>
                <Field label="Mes (MM) *">
                  <Input className="h-8 text-sm" value={form.expiry_month} onChange={(e) => setForm({ ...form, expiry_month: e.target.value })} />
                </Field>
                <Field label="Ano (AAAA) *">
                  <Input className="h-8 text-sm" value={form.expiry_year} onChange={(e) => setForm({ ...form, expiry_year: e.target.value })} />
                </Field>
                <Field label="CVV *">
                  <Input className="h-8 text-sm" value={form.ccv} onChange={(e) => setForm({ ...form, ccv: e.target.value })} />
                </Field>
                <Field label="CPF do titular *">
                  <Input className="h-8 text-sm" value={form.holder_cpf} onChange={(e) => setForm({ ...form, holder_cpf: e.target.value })} />
                </Field>
                <Field label="CEP">
                  <Input className="h-8 text-sm" value={form.postal_code} onChange={(e) => setForm({ ...form, postal_code: e.target.value })} />
                </Field>
                <Field label="Numero do endereco">
                  <Input className="h-8 text-sm" value={form.address_number} onChange={(e) => setForm({ ...form, address_number: e.target.value })} />
                </Field>
              </div>
              <p className="text-xs text-zinc-400">Os dados do cartao sao enviados direto ao gateway e nao sao armazenados.</p>
            </div>
          )}

          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving || !form.plan_key}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}

function Info({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <p className="text-xs text-zinc-500">{label}</p>
      <p className="font-medium">{value}</p>
    </div>
  );
}
