"use client";

import { useEffect, useState } from "react";
import { Plus, Repeat, Pause, Play, Wallet } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Input, Field, Select } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listRecurring, createRecurring, updateRecurring, listMembers, listCategories, listAccounts,
  type RecurringDonation, type Member, type Category, type BankAccount,
} from "@/lib/api";
import { PAYMENT_METHODS } from "@/lib/constants";
import { currency, datePt } from "@/lib/format";

const FREQUENCY: Record<string, string> = { weekly: "Semanal", monthly: "Mensal", yearly: "Anual" };
const SUBTYPE: Record<string, string> = { dizimo: "Dizimo", oferta: "Oferta", doacao: "Doacao" };

export function RecurringPanel() {
  const { hasPerm } = useAuth();
  const { toast } = useToast();
  const [items, setItems] = useState<RecurringDonation[] | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [cats, setCats] = useState<Category[]>([]);
  const [accts, setAccts] = useState<BankAccount[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    subtype: "dizimo", amount: "", frequency: "monthly", category_id: "",
    account_id: "", member_id: "", payment_method: "pix", description: "", period_start: "",
  });

  const load = async () => setItems((await listRecurring()).recurring);

  useEffect(() => {
    load().catch((e) => toast(e.message, "error"));
    listMembers().then((r) => setMembers(r.members)).catch(() => {});
    listCategories().then((r) => setCats(r.categories)).catch(() => {});
    listAccounts().then((r) => setAccts(r.accounts.filter((a) => a.is_active))).catch(() => {});
  }, [toast]);

  const incomeCats = cats.filter((c) => c.type === "income");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    try {
      await createRecurring({
        subtype: form.subtype,
        amount: Number(form.amount),
        frequency: form.frequency,
        category_id: form.category_id || undefined,
        account_id: form.account_id || undefined,
        member_id: form.member_id || undefined,
        payment_method: form.payment_method,
        description: form.description || undefined,
        period_start: form.period_start || undefined,
      });
      toast("Doacao recorrente agendada.");
      setOpen(false);
      setForm({ subtype: "dizimo", amount: "", frequency: "monthly", category_id: "", account_id: "", member_id: "", payment_method: "pix", description: "", period_start: "" });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function toggle(item: RecurringDonation) {
    try {
      await updateRecurring(item.id, { is_active: !item.is_active });
      toast(item.is_active ? "Pausado." : "Reativado.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div>
      <div className="mb-4 flex justify-end">
        {hasPerm("finance.write") && (
          <Button onClick={() => setOpen(true)}><Plus className="h-4 w-4" /> Agendar Doacao</Button>
        )}
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows />
        ) : items.length === 0 ? (
          <EmptyState icon={<Repeat className="h-10 w-10" />} title="Nenhuma doacao recorrente" description="Agende dizimos e ofertas para gerar lancamentos + recibos automaticos." />
        ) : (
          <Table>
            <THead><TRow><TH>Doador</TH><TH>Tipo</TH><TH>Valor</TH><TH>Frequencia</TH><TH>Proxima</TH><TH>Status</TH><TH className="text-right">Acoes</TH></TRow></THead>
            <TBody>
              {items.map((it) => (
                <TRow key={it.id}>
                  <TD>
                    <p className="font-medium">{it.member_name ?? "Anonimo"}</p>
                    {it.payment_method && <p className="text-xs text-zinc-400">{PAYMENT_METHODS[it.payment_method] ?? it.payment_method}</p>}
                  </TD>
                  <TD><Badge tone="sky">{SUBTYPE[it.subtype] ?? it.subtype}</Badge></TD>
                  <TD className="font-semibold text-emerald-600">{currency(it.amount)}</TD>
                  <TD className="text-zinc-500">{FREQUENCY[it.frequency] ?? it.frequency}</TD>
                  <TD className="text-zinc-500">{datePt(it.next_run_at)}</TD>
                  <TD><Badge tone={it.is_active ? "green" : "zinc"}>{it.is_active ? "Ativa" : "Pausada"}</Badge></TD>
                  <TD className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => toggle(it)}>
                      {it.is_active ? <><Pause className="h-3.5 w-3.5" /> Pausar</> : <><Play className="h-3.5 w-3.5" /> Reativar</>}
                    </Button>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title="Agendar doacao recorrente">
        <form onSubmit={submit} className="space-y-3">
          <Field label="Tipo">
            <Select value={form.subtype} onChange={(e) => setForm({ ...form, subtype: e.target.value })}>
              <option value="dizimo">Dizimo</option>
              <option value="oferta">Oferta</option>
              <option value="doacao">Doacao</option>
            </Select>
          </Field>
          <Field label="Valor (R$)*">
            <CurrencyInput required value={form.amount} onChange={(v) => setForm({ ...form, amount: v })} />
          </Field>
          <Field label="Frequencia">
            <Select value={form.frequency} onChange={(e) => setForm({ ...form, frequency: e.target.value })}>
              <option value="weekly">Semanal</option>
              <option value="monthly">Mensal</option>
              <option value="yearly">Anual</option>
            </Select>
          </Field>
          <Field label="Categoria de receita">
            <Select value={form.category_id} onChange={(e) => setForm({ ...form, category_id: e.target.value })}>
              <option value="">Automatica</option>
              {incomeCats.map((c) => (<option key={c.id} value={c.id}>{c.name}</option>))}
            </Select>
          </Field>
          {accts.length > 0 && (
            <Field label="Conta bancaria">
              <Select value={form.account_id} onChange={(e) => setForm({ ...form, account_id: e.target.value })}>
                <option value="">Nenhuma</option>
                {accts.map((a) => (<option key={a.id} value={a.id}>{a.name}</option>))}
              </Select>
            </Field>
          )}
          <Field label="Doador (membro)">
            <Select value={form.member_id} onChange={(e) => setForm({ ...form, member_id: e.target.value })}>
              <option value="">Anonimo</option>
              {members.map((m) => (<option key={m.id} value={m.id}>{m.full_name}</option>))}
            </Select>
          </Field>
          <Field label="Forma de pagamento">
            <Select value={form.payment_method} onChange={(e) => setForm({ ...form, payment_method: e.target.value })}>
              <option value="pix">PIX</option>
              <option value="card">Cartao</option>
              <option value="boleto">Boleto</option>
              <option value="transfer">Transferencia</option>
            </Select>
          </Field>
          <Field label="Inicio (data)">
            <Input type="date" value={form.period_start} onChange={(e) => setForm({ ...form, period_start: e.target.value })} />
          </Field>
          <Field label="Descricao">
            <Input value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="Ex.: Dizimo mensal" />
          </Field>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">{form.subtype === "dizimo" ? "Agendar Dizimo" : "Agendar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
