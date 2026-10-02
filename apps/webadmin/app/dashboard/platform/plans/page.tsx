"use client";

import { useCallback, useEffect, useState } from "react";
import { Layers, Pencil, Plus } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input } from "@/components/ui/input";
import { CurrencyInput } from "@/components/ui/currency-input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { currency } from "@/lib/format";
import { listPlans, createPlan, updatePlan, getFeatures, type Plan, type FeatureInfo } from "@/lib/api";

type FormState = {
  key: string; name: string; description: string; price: string; currency: string;
  max_members: string; max_branches: string; max_users: string; max_storage_mb: string;
  features: Record<string, boolean>;
  is_active: boolean; sort_order: string;
};

const EMPTY: FormState = {
  key: "", name: "", description: "", price: "", currency: "BRL",
  max_members: "", max_branches: "", max_users: "", max_storage_mb: "",
  features: {},
  is_active: true, sort_order: "0",
};

function toForm(p: Plan): FormState {
  return {
    key: p.key, name: p.name, description: p.description ?? "",
    // price_cents (inteiro) -> reais (string numerica "1234.56") para o input.
    price: ((p.price_cents ?? 0) / 100).toFixed(2), currency: p.currency ?? "BRL",
    max_members: num(p.max_members), max_branches: num(p.max_branches),
    max_users: num(p.max_users), max_storage_mb: num(p.max_storage_mb),
    features: { ...(p.features ?? {}) },
    is_active: p.is_active, sort_order: String(p.sort_order ?? 0),
  };
}

const fmtLimit = (v?: number | null) => (v === null || v === undefined ? "Ilimitado" : String(v));

export default function PlatformPlansPage() {
  const { toast } = useToast();
  const [plans, setPlans] = useState<Plan[] | null>(null);
  const [catalog, setCatalog] = useState<FeatureInfo[]>([]);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<Plan | null>(null);
  const [form, setForm] = useState<FormState>({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [r, f] = await Promise.all([listPlans(), getFeatures()]);
      setPlans(r.plans);
      setCatalog(f.features);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar planos", "error");
      setPlans([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  function openNew() {
    setEditing(null);
    setForm({ ...EMPTY });
    setOpen(true);
  }
  function openEdit(p: Plan) {
    setEditing(p);
    setForm(toForm(p));
    setOpen(true);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = {
        key: form.key,
        name: form.name,
        description: form.description,
        // reais -> centavos (inteiro) na gravacao.
        price_cents: Math.round(Number(form.price || 0) * 100),
        currency: form.currency,
        features: form.features,
        max_members: Number(form.max_members || 0),
        max_branches: Number(form.max_branches || 0),
        max_users: Number(form.max_users || 0),
        max_storage_mb: Number(form.max_storage_mb || 0),
        is_active: form.is_active,
        sort_order: Number(form.sort_order || 0),
      };
      if (editing) {
        await updatePlan(editing.key, payload);
        toast("Plano atualizado");
      } else {
        await createPlan(payload);
        toast("Plano criado");
      }
      setOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar plano", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Planos"
        description="Catalogo de planos e limites. 0 = ilimitado; vazio no limite da igreja usa o plano."
        actions={<Button onClick={openNew}><Plus className="h-4 w-4" /> Novo plano</Button>}
      />

      <Card className="overflow-hidden p-0">
        {plans === null ? (
          <div className="p-4"><SkeletonRows rows={4} /></div>
        ) : plans.length === 0 ? (
          <EmptyState icon={<Layers className="h-10 w-10" />} title="Nenhum plano" description="Crie o primeiro plano do catalogo." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Plano</TH><TH>Chave</TH><TH className="text-right">Preco</TH>
                <TH className="text-right">Membros</TH><TH className="text-right">Filiais</TH>
                <TH className="text-right">Usuarios</TH><TH className="text-right">Storage (MB)</TH>
                <TH>Situacao</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {plans.map((p) => (
                <TRow key={p.key}>
                  <TD className="font-medium">{p.name}</TD>
                  <TD className="font-mono text-xs text-zinc-500">{p.key}</TD>
                  <TD className="text-right tabular-nums">{currency((p.price_cents ?? 0) / 100)}</TD>
                  <TD className="text-right tabular-nums">{fmtLimit(p.max_members)}</TD>
                  <TD className="text-right tabular-nums">{fmtLimit(p.max_branches)}</TD>
                  <TD className="text-right tabular-nums">{fmtLimit(p.max_users)}</TD>
                  <TD className="text-right tabular-nums">{fmtLimit(p.max_storage_mb)}</TD>
                  <TD><Badge tone={p.is_active ? "green" : "zinc"}>{p.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                  <TD className="text-right">
                    <Button variant="outline" size="sm" onClick={() => openEdit(p)}>
                      <Pencil className="h-3.5 w-3.5" /> Editar
                    </Button>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title={editing ? `Editar plano - ${editing.name}` : "Novo plano"}>
        <form onSubmit={submit} className="space-y-3">
          {!editing && (
            <Field label="Chave *" hint="Identificador estavel (minusculas/hifen). Ex.: starter">
              <Input required className="h-8 text-sm" value={form.key} onChange={(e) => setForm({ ...form, key: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })} />
            </Field>
          )}
          <Field label="Nome *">
            <Input required className="h-8 text-sm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </Field>
          <Field label="Descricao">
            <Input className="h-8 text-sm" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Preco (R$)">
              <CurrencyInput className="h-8 text-sm" value={form.price} onChange={(v) => setForm({ ...form, price: v })} />
            </Field>
            <Field label="Moeda">
              <Input className="h-8 text-sm" value={form.currency} onChange={(e) => setForm({ ...form, currency: e.target.value })} />
            </Field>
            <Field label="Max. membros" hint="0 = ilimitado">
              <Input type="number" min={0} className="h-8 text-sm" value={form.max_members} onChange={(e) => setForm({ ...form, max_members: e.target.value })} />
            </Field>
            <Field label="Max. filiais" hint="0 = ilimitado">
              <Input type="number" min={0} className="h-8 text-sm" value={form.max_branches} onChange={(e) => setForm({ ...form, max_branches: e.target.value })} />
            </Field>
            <Field label="Max. usuarios" hint="0 = ilimitado">
              <Input type="number" min={0} className="h-8 text-sm" value={form.max_users} onChange={(e) => setForm({ ...form, max_users: e.target.value })} />
            </Field>
            <Field label="Storage (MB)" hint="0 = ilimitado">
              <Input type="number" min={0} className="h-8 text-sm" value={form.max_storage_mb} onChange={(e) => setForm({ ...form, max_storage_mb: e.target.value })} />
            </Field>
            <Field label="Ordem">
              <Input type="number" className="h-8 text-sm" value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: e.target.value })} />
            </Field>
          </div>
          <div className="border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Modulos incluidos</p>
            <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2">
              {catalog.map((f) => (
                <label key={f.key} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4"
                    checked={form.features[f.key] ?? true}
                    onChange={(e) => setForm({ ...form, features: { ...form.features, [f.key]: e.target.checked } })}
                  />
                  {f.label}
                </label>
              ))}
            </div>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="h-4 w-4" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            Ativo
          </label>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}

function num(v?: number | null): string {
  return v === null || v === undefined ? "" : String(v);
}
