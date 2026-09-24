"use client";

import { useCallback, useEffect, useState } from "react";
import { Building2, Church, Pencil, Plus, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { Tabs } from "@/components/ui/tabs";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { mutate } from "@/lib/swr-hooks";
import {
  getTenant, updateTenant, listBranches, createBranch, updateBranch, deleteBranch,
  type Tenant, type Branch,
} from "@/lib/api";

const BRANCH_KINDS = [
  { v: "branch", l: "Filial" },
  { v: "congregation", l: "Congregação" },
  { v: "sub_congregation", l: "Sub-congregação" },
];

const EMPTY_BRANCH = {
  name: "", slug: "", kind: "branch", cnpj: "", parent_id: "", is_active: "true",
  street: "", number: "", district: "", city: "", state: "", zip_code: "",
};

type BranchAddress = {
  street?: string; number?: string; district?: string;
  city?: string; state?: string; zip_code?: string;
};

export default function SettingsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("settings.write");

  const [tab, setTab] = useState("igreja");
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [tenantForm, setTenantForm] = useState({
    name: "", legal_name: "", cnpj: "", plan: "starter", locale: "pt-BR", timezone: "America/Sao_Paulo",
    logo_url: "", brand_color: "", favicon_url: "", custom_domain: "",
  });
  const [savingTenant, setSavingTenant] = useState(false);

  const [branches, setBranches] = useState<Branch[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Branch }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY_BRANCH });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const [t, b] = await Promise.all([getTenant(), listBranches()]);
      setTenant(t);
      setTenantForm({
        name: t.name, legal_name: t.legal_name ?? "", cnpj: t.cnpj ?? "",
        plan: t.plan, locale: t.locale, timezone: t.timezone,
        logo_url: t.logo_url ?? "", brand_color: t.brand_color ?? "",
        favicon_url: t.favicon_url ?? "", custom_domain: t.custom_domain ?? "",
      });
      setBranches(b.branches);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar configurações", "error");
      setBranches([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  async function saveTenant(e: React.FormEvent) {
    e.preventDefault();
    setSavingTenant(true);
    try {
      await updateTenant(tenantForm);
      toast("Dados da igreja atualizados.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSavingTenant(false);
    }
  }

  function openCreate() {
    setForm({ ...EMPTY_BRANCH });
    setDrawer({ open: true });
  }
  function openEdit(b: Branch) {
    const a = (b.address ?? {}) as BranchAddress;
    setForm({
      name: b.name, slug: b.slug, kind: b.kind, cnpj: b.cnpj ?? "", parent_id: b.parent_id ?? "",
      is_active: (b.is_active ?? true) ? "true" : "false",
      street: a.street ?? "", number: a.number ?? "", district: a.district ?? "",
      city: a.city ?? "", state: a.state ?? "", zip_code: a.zip_code ?? "",
    });
    setDrawer({ open: true, editing: b });
  }
  async function saveBranch(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        name: form.name,
        slug: form.slug || undefined,
        kind: form.kind,
        cnpj: form.cnpj || undefined,
        parent_id: form.parent_id || undefined,
        is_active: form.is_active === "true",
        address: {
          street: form.street, number: form.number, district: form.district,
          city: form.city, state: form.state, zip_code: form.zip_code,
        },
      };
      if (drawer.editing) await updateBranch(drawer.editing.id, payload);
      else await createBranch(payload);
      toast("Filial salva.");
      setDrawer({ open: false });
      await load();
      mutate("branches");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar filial", "error");
    } finally {
      setSaving(false);
    }
  }
  async function removeBranch(b: Branch) {
    if (!confirm(`Excluir a filial "${b.name}"?`)) return;
    try {
      await deleteBranch(b.id);
      toast("Filial excluída.");
      await load();
      mutate("branches");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Configurações"
        description="Dados da igreja e filiais associadas ao tenant"
        actions={canWrite && tab === "filiais" ? <Button onClick={openCreate}><Plus className="h-4 w-4" /> Nova filial</Button> : undefined}
      />

      <Tabs
        tabs={[
          { key: "igreja", label: "Dados da igreja", icon: <Church className="h-4 w-4" /> },
          { key: "filiais", label: "Filiais", icon: <Building2 className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "igreja" && (
        tenant === null ? <Card className="p-4"><SkeletonRows rows={5} /></Card> : (
          <Card className="max-w-2xl p-4">
            <form onSubmit={saveTenant} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <Field label="Nome da igreja" required className="sm:col-span-2">
                <Input required disabled={!canWrite} className="h-8 text-sm" value={tenantForm.name} onChange={(e) => setTenantForm({ ...tenantForm, name: e.target.value })} />
              </Field>
              <Field label="Razão social">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.legal_name} onChange={(e) => setTenantForm({ ...tenantForm, legal_name: e.target.value })} />
              </Field>
              <Field label="CNPJ">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.cnpj} onChange={(e) => setTenantForm({ ...tenantForm, cnpj: e.target.value })} />
              </Field>
              <Field label="Plano">
                <Select disabled={!canWrite} className="h-8 text-sm" value={tenantForm.plan} onChange={(e) => setTenantForm({ ...tenantForm, plan: e.target.value })}>
                  <option value="starter">Starter</option>
                  <option value="pro">Pro</option>
                  <option value="enterprise">Enterprise</option>
                </Select>
              </Field>
              <Field label="Fuso horário">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.timezone} onChange={(e) => setTenantForm({ ...tenantForm, timezone: e.target.value })} />
              </Field>
              <Field label="Idioma (locale)">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.locale} onChange={(e) => setTenantForm({ ...tenantForm, locale: e.target.value })} />
              </Field>
              <p className="text-xs text-zinc-400 sm:col-span-2">Identificador do tenant: <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">{tenant.slug}</code></p>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-400 sm:col-span-2">White-label (subdomínio)</p>
              <Field label="URL do logo" hint="Exibido na tela de login da igreja.">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="https://..." value={tenantForm.logo_url} onChange={(e) => setTenantForm({ ...tenantForm, logo_url: e.target.value })} />
              </Field>
              <Field label="Cor da marca" hint="Hexadecimal, ex.: #0ea5e9.">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="#0ea5e9" value={tenantForm.brand_color} onChange={(e) => setTenantForm({ ...tenantForm, brand_color: e.target.value })} />
              </Field>
              <Field label="URL do favicon">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="https://..." value={tenantForm.favicon_url} onChange={(e) => setTenantForm({ ...tenantForm, favicon_url: e.target.value })} />
              </Field>
              <Field label="Domínio próprio" hint="Opcional (ex.: igreja.minhadominio.com).">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.custom_domain} onChange={(e) => setTenantForm({ ...tenantForm, custom_domain: e.target.value })} />
              </Field>
              {canWrite && (
                <div className="flex justify-end sm:col-span-2">
                  <Button type="submit" className="h-8 text-sm" disabled={savingTenant}>{savingTenant ? "Salvando..." : "Salvar"}</Button>
                </div>
              )}
            </form>
          </Card>
        )
      )}

      {tab === "filiais" && (
        <Card className="overflow-hidden p-0">
          {branches === null ? (
            <div className="p-4"><SkeletonRows rows={5} /></div>
          ) : branches.length === 0 ? (
            <EmptyState icon={<Building2 className="h-10 w-10" />} title="Nenhuma filial" description="Cadastre as congregações e filiais da igreja." />
          ) : (
            <Table>
              <THead><TRow><TH>Nome</TH><TH>Tipo</TH><TH>Identificador</TH><TH>CNPJ</TH><TH className="text-right">Membros</TH><TH>Situação</TH><TH className="text-right">Ações</TH></TRow></THead>
              <TBody>
                {branches.map((b) => (
                  <TRow key={b.id}>
                    <TD className="font-medium">{b.name}</TD>
                    <TD><Badge tone="zinc">{BRANCH_KINDS.find((k) => k.v === b.kind)?.l ?? b.kind}</Badge></TD>
                    <TD className="text-sm text-zinc-500">{b.slug}</TD>
                    <TD className="text-sm text-zinc-500">{b.cnpj ?? "—"}</TD>
                    <TD className="text-right tabular-nums">{b.member_count ?? 0}</TD>
                    <TD><Badge tone={(b.is_active ?? true) ? "green" : "zinc"}>{(b.is_active ?? true) ? "Ativa" : "Inativa"}</Badge></TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        {canWrite && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(b)}><Pencil className="h-4 w-4" /></Button>}
                        {canWrite && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeBranch(b)}><Trash2 className="h-4 w-4" /></Button>}
                      </div>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} title={drawer.editing ? "Editar filial" : "Nova filial"}>
        <form onSubmit={saveBranch} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Nome" required className="sm:col-span-2">
              <Input required className="h-8 text-sm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </Field>
            <Field label="Identificador (slug)" hint="Deixe vazio para gerar a partir do nome.">
              <Input className="h-8 text-sm" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })} />
            </Field>
            <Field label="CNPJ">
              <Input className="h-8 text-sm" placeholder="00.000.000/0000-00" value={form.cnpj} onChange={(e) => setForm({ ...form, cnpj: e.target.value })} />
            </Field>
            <Field label="Tipo">
              <Select className="h-8 text-sm" value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
                {BRANCH_KINDS.map((k) => <option key={k.v} value={k.v}>{k.l}</option>)}
              </Select>
            </Field>
            <Field label="Filial superior" hint="Para hierarquia Sede > Congregação.">
              <Select className="h-8 text-sm" value={form.parent_id} onChange={(e) => setForm({ ...form, parent_id: e.target.value })}>
                <option value="">—</option>
                {(branches ?? []).filter((b) => b.id !== drawer.editing?.id).map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
              </Select>
            </Field>
            <Field label="Situação">
              <Select className="h-8 text-sm" value={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.value })}>
                <option value="true">Ativa</option>
                <option value="false">Inativa</option>
              </Select>
            </Field>
            <Field label="Logradouro" className="sm:col-span-2">
              <Input className="h-8 text-sm" value={form.street} onChange={(e) => setForm({ ...form, street: e.target.value })} />
            </Field>
            <Field label="Número">
              <Input className="h-8 text-sm" value={form.number} onChange={(e) => setForm({ ...form, number: e.target.value })} />
            </Field>
            <Field label="Bairro">
              <Input className="h-8 text-sm" value={form.district} onChange={(e) => setForm({ ...form, district: e.target.value })} />
            </Field>
            <Field label="Cidade">
              <Input className="h-8 text-sm" value={form.city} onChange={(e) => setForm({ ...form, city: e.target.value })} />
            </Field>
            <Field label="UF">
              <Input maxLength={2} className="h-8 text-sm" value={form.state} onChange={(e) => setForm({ ...form, state: e.target.value.toUpperCase() })} />
            </Field>
            <Field label="CEP">
              <Input className="h-8 text-sm" value={form.zip_code} onChange={(e) => setForm({ ...form, zip_code: e.target.value })} />
            </Field>
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
