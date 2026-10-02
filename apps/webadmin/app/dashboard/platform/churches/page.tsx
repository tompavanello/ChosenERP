"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { Globe, Pencil, Plus, Search } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listAllTenants, createTenant, listPlans,
  type AdminTenant, type Plan,
} from "@/lib/api";

const BASE_DOMAIN = process.env.NEXT_PUBLIC_BASE_DOMAIN ?? "erpchosen.com.br";

const EMPTY_NEW = {
  name: "", slug: "", plan: "starter", admin_name: "", admin_email: "", admin_password: "",
};

export default function PlatformChurchesPage() {
  const { toast } = useToast();
  const [tenants, setTenants] = useState<AdminTenant[] | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [q, setQ] = useState("");

  const [newOpen, setNewOpen] = useState(false);
  const [newForm, setNewForm] = useState({ ...EMPTY_NEW });
  const [savingNew, setSavingNew] = useState(false);

  const load = useCallback(async () => {
    try {
      const [t, p] = await Promise.all([listAllTenants(), listPlans()]);
      setTenants(t.tenants);
      setPlans(p.plans);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar igrejas", "error");
      setTenants([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const filtered = useMemo(() => {
    const list = tenants ?? [];
    const s = q.trim().toLowerCase();
    if (!s) return list;
    return list.filter((t) => t.name.toLowerCase().includes(s) || t.slug.includes(s));
  }, [tenants, q]);

  async function submitNew(e: React.FormEvent) {
    e.preventDefault();
    setSavingNew(true);
    try {
      const r = await createTenant({ ...newForm });
      toast(`Igreja criada. Acesse ${r.subdomain}`);
      setNewOpen(false);
      setNewForm({ ...EMPTY_NEW });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar igreja", "error");
    } finally {
      setSavingNew(false);
    }
  }

  return (
    <div className="page space-y-4">
      <PageHeader
        title="Igrejas"
        description="Console da plataforma: crie, edite, ajuste plano/limites e suspenda igrejas."
        actions={
          <Button onClick={() => setNewOpen(true)}>
            <Plus className="h-4 w-4" /> Nova igreja
          </Button>
        }
      />

      <Card className="p-3">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
          <Input
            className="pl-9"
            placeholder="Buscar por nome ou subdominio..."
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
        </div>
      </Card>

      <Card className="overflow-hidden p-0">
        {tenants === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : filtered.length === 0 ? (
          <EmptyState icon={<Globe className="h-10 w-10" />} title="Nenhuma igreja" description="Crie a primeira igreja; o subdominio passa a funcionar na hora." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Igreja</TH><TH>Subdominio</TH><TH>Plano</TH>
                <TH className="text-right">Filiais</TH><TH className="text-right">Membros</TH>
                <TH>Situacao</TH><TH></TH>
              </TRow>
            </THead>
            <TBody>
              {filtered.map((t) => (
                <TRow key={t.id}>
                  <TD className="font-medium">{t.name}</TD>
                  <TD className="text-sm">
                    <a className="text-sky-600 hover:underline" href={`https://${t.slug}.${BASE_DOMAIN}`} target="_blank" rel="noreferrer">
                      {t.slug}.{BASE_DOMAIN}
                    </a>
                  </TD>
                  <TD><Badge tone="zinc">{t.plan}</Badge></TD>
                  <TD className="text-right tabular-nums">{t.branch_count}</TD>
                  <TD className="text-right tabular-nums">{t.member_count}</TD>
                  <TD><Badge tone={t.is_active ? "green" : "zinc"}>{t.is_active ? "Ativa" : "Inativa"}</Badge></TD>
                  <TD className="text-right">
                    <Link href={`/dashboard/platform/churches/${t.id}`} className="btn-base btn-outline px-2.5 py-1.5 text-xs rounded-md">
                      <Pencil className="h-3.5 w-3.5" /> Gerenciar
                    </Link>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      {/* Criar igreja */}
      <Drawer open={newOpen} onClose={() => setNewOpen(false)} title="Nova igreja">
        <form onSubmit={submitNew} className="space-y-3">
          <Field label="Nome da igreja *">
            <Input required className="h-8 text-sm" value={newForm.name} onChange={(e) => setNewForm({ ...newForm, name: e.target.value })} />
          </Field>
          <Field label="Subdominio (slug) *" hint={`Ex.: matriz -> matriz.${BASE_DOMAIN}`}>
            <Input required className="h-8 text-sm" value={newForm.slug} onChange={(e) => setNewForm({ ...newForm, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })} />
          </Field>
          <Field label="Plano">
            <Select className="h-8 text-sm" value={newForm.plan} onChange={(e) => setNewForm({ ...newForm, plan: e.target.value })}>
              {plans.map((p) => <option key={p.key} value={p.key}>{p.name}</option>)}
            </Select>
          </Field>
          <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Primeiro super_admin</p>
          <Field label="Nome do admin *">
            <Input required className="h-8 text-sm" value={newForm.admin_name} onChange={(e) => setNewForm({ ...newForm, admin_name: e.target.value })} />
          </Field>
          <Field label="E-mail do admin *">
            <Input required type="email" className="h-8 text-sm" value={newForm.admin_email} onChange={(e) => setNewForm({ ...newForm, admin_email: e.target.value })} />
          </Field>
          <Field label="Senha do admin *" hint="Minimo 8 caracteres.">
            <Input required type="password" className="h-8 text-sm" value={newForm.admin_password} onChange={(e) => setNewForm({ ...newForm, admin_password: e.target.value })} />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setNewOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={savingNew}>{savingNew ? "Criando..." : "Criar igreja"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
