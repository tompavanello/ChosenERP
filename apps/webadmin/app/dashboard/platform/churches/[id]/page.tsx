"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { ArrowLeft } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Tabs } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Field, Input, Select } from "@/components/ui/input";
import { SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { TenantUsers } from "@/components/platform/tenant-users";
import {
  getAdminTenant, getAdminTenantUsage, listPlans, updateAdminTenant, getFeatures,
  type AdminTenantDetail, type Plan, type TenantUsage, type FeatureInfo,
} from "@/lib/api";

const BASE_DOMAIN = process.env.NEXT_PUBLIC_BASE_DOMAIN ?? "erpchosen.com.br";

export default function ChurchDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { toast } = useToast();

  const [detail, setDetail] = useState<AdminTenantDetail | null>(null);
  const [usage, setUsage] = useState<TenantUsage | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [catalog, setCatalog] = useState<FeatureInfo[]>([]);
  const [featureOverride, setFeatureOverride] = useState<Record<string, boolean | undefined>>({});
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [tab, setTab] = useState("dados");
  const [saving, setSaving] = useState(false);

  const [form, setForm] = useState({
    name: "", slug: "", legal_name: "", cnpj: "", plan: "starter",
    locale: "pt-BR", timezone: "America/Sao_Paulo", custom_domain: "",
    logo_url: "", brand_color: "", favicon_url: "",
    max_members: "", max_branches: "", max_users: "", max_storage_mb: "",
    is_active: true,
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [d, u, p, f] = await Promise.all([
        getAdminTenant(id),
        getAdminTenantUsage(id),
        listPlans(),
        getFeatures(),
      ]);
      setDetail(d);
      setUsage(u);
      setPlans(p.plans);
      setCatalog(f.features);
      setFeatureOverride({ ...(d.features ?? {}) });
      setForm({
        name: d.name, slug: d.slug, legal_name: d.legal_name ?? "", cnpj: d.cnpj ?? "",
        plan: d.plan, locale: d.locale, timezone: d.timezone, custom_domain: d.custom_domain ?? "",
        logo_url: d.logo_url ?? "", brand_color: d.brand_color ?? "", favicon_url: d.favicon_url ?? "",
        max_members: str(d.limits?.max_members), max_branches: str(d.limits?.max_branches),
        max_users: str(d.limits?.max_users), max_storage_mb: str(d.limits?.max_storage_mb),
        is_active: d.is_active,
      });
    } catch {
      setNotFound(true);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const limits: Record<string, number> = {};
      if (form.max_members) limits.max_members = Number(form.max_members);
      if (form.max_branches) limits.max_branches = Number(form.max_branches);
      if (form.max_users) limits.max_users = Number(form.max_users);
      if (form.max_storage_mb) limits.max_storage_mb = Number(form.max_storage_mb);
      // Somente as chaves definidas viram override; o resto herda do plano.
      const features: Record<string, boolean> = {};
      for (const [k, v] of Object.entries(featureOverride)) {
        if (v !== undefined) features[k] = v;
      }
      const updated = await updateAdminTenant(id, {
        name: form.name, slug: form.slug, legal_name: form.legal_name, cnpj: form.cnpj,
        plan: form.plan, locale: form.locale, timezone: form.timezone,
        custom_domain: form.custom_domain, logo_url: form.logo_url,
        brand_color: form.brand_color, favicon_url: form.favicon_url,
        limits, features, is_active: form.is_active,
      });
      setDetail(updated);
      toast("Igreja atualizada");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar igreja", "error");
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="page"><SkeletonRows rows={6} /></div>;
  if (notFound || !detail) {
    return (
      <div className="page space-y-3">
        <Card><p className="text-sm text-zinc-500">Igreja nao encontrada.</p></Card>
        <Link href="/dashboard/platform/churches" className="text-sm text-sky-600">Voltar</Link>
      </div>
    );
  }

  return (
    <div className="page space-y-4">
      <PageHeader
        title={detail.name}
        description={`${detail.slug}.${BASE_DOMAIN}`}
        actions={
          <Link href="/dashboard/platform/churches" className="btn-base btn-outline px-4 py-2 text-sm rounded-lg">
            <ArrowLeft className="h-4 w-4" /> Voltar
          </Link>
        }
      />

      <div className="flex items-center gap-2">
        <Badge tone={detail.is_active ? "green" : "zinc"}>{detail.is_active ? "Ativa" : "Inativa"}</Badge>
        <Badge tone="zinc">Plano {detail.plan}</Badge>
      </div>

      <Tabs
        tabs={[
          { key: "dados", label: "Dados" },
          { key: "plano", label: "Plano & Limites" },
          { key: "usuarios", label: "Usuarios" },
          { key: "branding", label: "Branding" },
          { key: "situacao", label: "Situacao" },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "usuarios" ? (
        <TenantUsers tenantId={detail.id} />
      ) : (
        <form onSubmit={save} className="max-w-2xl space-y-4">
          {tab === "dados" && (
            <Card className="space-y-3">
              <Field label="Nome">
                <Input className="h-8 text-sm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
              </Field>
              <Field label="Subdominio (slug)">
                <Input className="h-8 text-sm" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })} />
              </Field>
              <Field label="Razao social">
                <Input className="h-8 text-sm" value={form.legal_name} onChange={(e) => setForm({ ...form, legal_name: e.target.value })} />
              </Field>
              <Field label="CNPJ">
                <Input className="h-8 text-sm" value={form.cnpj} onChange={(e) => setForm({ ...form, cnpj: e.target.value })} />
              </Field>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Locale">
                  <Input className="h-8 text-sm" value={form.locale} onChange={(e) => setForm({ ...form, locale: e.target.value })} />
                </Field>
                <Field label="Timezone">
                  <Input className="h-8 text-sm" value={form.timezone} onChange={(e) => setForm({ ...form, timezone: e.target.value })} />
                </Field>
              </div>
              <Field label="Dominio proprio (custom domain)">
                <Input className="h-8 text-sm" value={form.custom_domain} onChange={(e) => setForm({ ...form, custom_domain: e.target.value })} />
              </Field>
            </Card>
          )}

          {tab === "plano" && (
            <Card className="space-y-3">
              {usage && (
                <div className="grid grid-cols-3 gap-2 rounded-lg bg-zinc-50 p-3 text-center dark:bg-zinc-900">
                  <Usage label="Membros" value={usage.members} />
                  <Usage label="Filiais" value={usage.branches} />
                  <Usage label="Usuarios" value={usage.users} />
                </div>
              )}
              <Field label="Plano">
                <Select className="h-8 text-sm" value={form.plan} onChange={(e) => setForm({ ...form, plan: e.target.value })}>
                  {plans.map((p) => <option key={p.key} value={p.key}>{p.name}</option>)}
                </Select>
              </Field>
              <p className="text-xs text-zinc-400">Limites por igreja (vazio = usa o plano; 0 = ilimitado).</p>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Max. membros">
                  <Input type="number" min={0} className="h-8 text-sm" value={form.max_members} onChange={(e) => setForm({ ...form, max_members: e.target.value })} />
                </Field>
                <Field label="Max. filiais">
                  <Input type="number" min={0} className="h-8 text-sm" value={form.max_branches} onChange={(e) => setForm({ ...form, max_branches: e.target.value })} />
                </Field>
                <Field label="Max. usuarios">
                  <Input type="number" min={0} className="h-8 text-sm" value={form.max_users} onChange={(e) => setForm({ ...form, max_users: e.target.value })} />
                </Field>
                <Field label="Max. armazenamento (MB)">
                  <Input type="number" min={0} className="h-8 text-sm" value={form.max_storage_mb} onChange={(e) => setForm({ ...form, max_storage_mb: e.target.value })} />
                </Field>
              </div>

              <div className="border-t border-zinc-100 pt-3 dark:border-zinc-800">
                <p className="mb-2 text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Modulos (override desta igreja)</p>
                <div className="space-y-1.5">
                  {catalog.map((f) => {
                    const inherited = plans.find((p) => p.key === form.plan)?.features?.[f.key] ?? true;
                    const val = featureOverride[f.key];
                    return (
                      <div key={f.key} className="flex items-center justify-between gap-3 text-sm">
                        <span>
                          {f.label}{" "}
                          <span className="text-xs text-zinc-400">(plano: {inherited ? "incluido" : "excluido"})</span>
                        </span>
                        <select
                          className="h-8 rounded-md border border-[var(--line)] bg-[var(--card)] px-2 text-xs"
                          value={val === undefined ? "inherit" : val ? "on" : "off"}
                          onChange={(e) => {
                            const v = e.target.value;
                            setFeatureOverride({ ...featureOverride, [f.key]: v === "inherit" ? undefined : v === "on" });
                          }}
                        >
                          <option value="inherit">Herdar do plano</option>
                          <option value="on">Liberar</option>
                          <option value="off">Bloquear</option>
                        </select>
                      </div>
                    );
                  })}
                </div>
              </div>
            </Card>
          )}

          {tab === "branding" && (
            <Card className="space-y-3">
              <Field label="Logo (URL)">
                <Input className="h-8 text-sm" value={form.logo_url} onChange={(e) => setForm({ ...form, logo_url: e.target.value })} />
              </Field>
              <Field label="Cor da marca (hex)" hint="Ex.: #0284c7">
                <Input className="h-8 text-sm" value={form.brand_color} onChange={(e) => setForm({ ...form, brand_color: e.target.value })} />
              </Field>
              <Field label="Favicon (URL)">
                <Input className="h-8 text-sm" value={form.favicon_url} onChange={(e) => setForm({ ...form, favicon_url: e.target.value })} />
              </Field>
            </Card>
          )}

          {tab === "situacao" && (
            <Card>
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" className="h-4 w-4" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
                Igreja ativa (desmarcar suspende o acesso, mantendo os dados)
              </label>
            </Card>
          )}

          <div className="flex justify-end">
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar alteracoes"}</Button>
          </div>
        </form>
      )}
    </div>
  );
}

function Usage({ label, value }: { label: string; value: number }) {
  return (
    <div>
      <div className="text-lg font-semibold tabular-nums">{value}</div>
      <div className="text-[11px] text-zinc-400">{label}</div>
    </div>
  );
}

function str(v: number | undefined | null): string {
  return v === undefined || v === null ? "" : String(v);
}
