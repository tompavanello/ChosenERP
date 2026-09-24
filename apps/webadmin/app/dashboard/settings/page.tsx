"use client";

import { useCallback, useEffect, useState } from "react";
import { Building2, Church, Globe, Pencil, Plus, Trash2 } from "lucide-react";
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
  getBranchChannels, updateBranchChannels, connectBranchWhatsApp,
  getBranchWhatsAppState, disconnectBranchWhatsApp,
  listAllTenants, createTenant,
  type Tenant, type Branch, type AdminTenant,
} from "@/lib/api";

// Estrutura de governo da igreja: Matriz (Sede) > Filial (congregacao) > PAE.
const BRANCH_KINDS = [
  { v: "matriz", l: "Matriz (Sede)" },
  { v: "filial", l: "Filial / Regional" },
  { v: "pae", l: "PAE (Ponto de Atendimento)" },
];

const EMPTY_BRANCH = {
  name: "", slug: "", kind: "filial", cnpj: "", parent_id: "", is_active: "true",
  street: "", number: "", district: "", city: "", state: "", zip_code: "",
};

const EMPTY_CHANNELS = {
  whatsapp_phone: "", smtp_host: "", smtp_port: "587", smtp_user: "",
  smtp_password: "", smtp_from: "", smtp_from_name: "Chosen ERP", smtp_secure: false,
};

type BranchAddress = {
  street?: string; number?: string; district?: string;
  city?: string; state?: string; zip_code?: string;
};

const EMPTY_CHURCH = {
  name: "", slug: "", plan: "starter", admin_name: "", admin_email: "", admin_password: "",
};

// Dominio base do white-label (subdominio da igreja).
const BASE_DOMAIN = process.env.NEXT_PUBLIC_BASE_DOMAIN ?? "erpchosen.com.br";

export default function SettingsPage() {
  const { toast } = useToast();
  const { user, hasPerm } = useAuth();
  const canWrite = hasPerm("settings.write");
  const isSuperAdmin = user?.role === "super_admin";

  const [tab, setTab] = useState("igreja");
  // Onboarding de igrejas (apenas super_admin).
  const [churches, setChurches] = useState<AdminTenant[] | null>(null);
  const [churchDrawer, setChurchDrawer] = useState(false);
  const [churchForm, setChurchForm] = useState({ ...EMPTY_CHURCH });
  const [savingChurch, setSavingChurch] = useState(false);
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [tenantForm, setTenantForm] = useState({
    name: "", slug: "", legal_name: "", cnpj: "", plan: "starter", locale: "pt-BR", timezone: "America/Sao_Paulo",
    logo_url: "", brand_color: "", favicon_url: "", custom_domain: "",
  });
  const [savingTenant, setSavingTenant] = useState(false);

  const [branches, setBranches] = useState<Branch[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Branch }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY_BRANCH });
  const [saving, setSaving] = useState(false);

  // Canais por filial (WhatsApp + SMTP), editados no mesmo drawer.
  const [chForm, setChForm] = useState({ ...EMPTY_CHANNELS });
  const [waStatus, setWaStatus] = useState("disconnected");
  const [waNumber, setWaNumber] = useState("");
  const [qr, setQr] = useState("");
  const [connecting, setConnecting] = useState(false);
  const [savingChannels, setSavingChannels] = useState(false);

  const load = useCallback(async () => {
    try {
      const [t, b] = await Promise.all([getTenant(), listBranches()]);
      setTenant(t);
      setTenantForm({
        name: t.name, slug: t.slug, legal_name: t.legal_name ?? "", cnpj: t.cnpj ?? "",
        plan: t.plan, locale: t.locale, timezone: t.timezone,
        logo_url: t.logo_url ?? "", brand_color: t.brand_color ?? "",
        favicon_url: t.favicon_url ?? "", custom_domain: t.custom_domain ?? "",
      });
      setBranches(b.branches);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar configuracoes", "error");
      setBranches([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const loadChurches = useCallback(async () => {
    try {
      const r = await listAllTenants();
      setChurches(r.tenants);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar igrejas", "error");
      setChurches([]);
    }
  }, [toast]);

  useEffect(() => {
    if (tab === "igrejas" && isSuperAdmin) loadChurches();
  }, [tab, isSuperAdmin, loadChurches]);

  async function submitChurch(e: React.FormEvent) {
    e.preventDefault();
    setSavingChurch(true);
    try {
      const r = await createTenant({
        name: churchForm.name,
        slug: churchForm.slug,
        plan: churchForm.plan,
        admin_name: churchForm.admin_name,
        admin_email: churchForm.admin_email,
        admin_password: churchForm.admin_password,
      });
      toast(`Igreja criada. Acesse ${r.subdomain}`);
      setChurchDrawer(false);
      setChurchForm({ ...EMPTY_CHURCH });
      await loadChurches();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar igreja", "error");
    } finally {
      setSavingChurch(false);
    }
  }

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
    setChForm({ ...EMPTY_CHANNELS });
    setQr("");
    setConnecting(false);
    setWaStatus("disconnected");
    setWaNumber("");
    getBranchChannels(b.id)
      .then((c) => {
        setWaStatus(c.whatsapp_status);
        setWaNumber(c.whatsapp_number || "");
        setChForm({
          whatsapp_phone: c.whatsapp_phone,
          smtp_host: c.smtp_host,
          smtp_port: String(c.smtp_port || 587),
          smtp_user: c.smtp_user,
          smtp_password: "",
          smtp_from: c.smtp_from,
          smtp_from_name: c.smtp_from_name || "Chosen ERP",
          smtp_secure: c.smtp_secure,
        });
      })
      .catch(() => {/* sem canais configurados */});
    setDrawer({ open: true, editing: b });
  }

  async function saveChannels() {
    if (!drawer.editing) return;
    setSavingChannels(true);
    try {
      const payload: Record<string, unknown> = {
        whatsapp_phone: chForm.whatsapp_phone,
        smtp_host: chForm.smtp_host,
        smtp_port: Number(chForm.smtp_port) || 587,
        smtp_user: chForm.smtp_user,
        smtp_from: chForm.smtp_from,
        smtp_from_name: chForm.smtp_from_name,
        smtp_secure: chForm.smtp_secure,
      };
      if (chForm.smtp_password) payload.smtp_password = chForm.smtp_password;
      const c = await updateBranchChannels(drawer.editing.id, payload);
      setWaStatus(c.whatsapp_status);
      setChForm({ ...chForm, smtp_password: "" });
      toast("Canais da filial salvos.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar canais", "error");
    } finally {
      setSavingChannels(false);
    }
  }

  async function connectWhatsApp() {
    if (!drawer.editing) return;
    setConnecting(true);
    try {
      const res = await connectBranchWhatsApp(drawer.editing.id);
      setQr(res.qrcode_base64 || "");
      setWaNumber(res.number || "");
      if (res.status === "connected") {
        setWaStatus("connected");
        setConnecting(false);
        toast("WhatsApp ja conectado.");
      } else {
        setWaStatus("connecting");
        toast("Leia o QR Code no WhatsApp da filial.");
      }
      await load();
      mutate("branches");
    } catch (err) {
      setConnecting(false);
      toast(err instanceof Error ? err.message : "Erro ao conectar WhatsApp", "error");
    }
  }

  async function disconnectWhatsApp() {
    if (!drawer.editing) return;
    try {
      await disconnectBranchWhatsApp(drawer.editing.id);
      setWaStatus("disconnected");
      setWaNumber("");
      setQr("");
      setConnecting(false);
      toast("WhatsApp desconectado.");
      await load();
      mutate("branches");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao desconectar", "error");
    }
  }

  // Enquanto ha QR na tela, consulta o estado ate conectar.
  useEffect(() => {
    if (!connecting || !drawer.editing) return;
    const id = drawer.editing.id;
    const timer = setInterval(async () => {
      try {
        const s = await getBranchWhatsAppState(id);
        setWaStatus(s.status);
        if (s.number) setWaNumber(s.number);
        if (s.connected) {
          setQr("");
          setConnecting(false);
          await load();
          mutate("branches");
        }
      } catch {
        /* mantem a tentativa */
      }
    }, 3000);
    return () => clearInterval(timer);
  }, [connecting, drawer, load]);
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
      toast("Filial excluida.");
      await load();
      mutate("branches");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Configuracoes"
        description="Dados da igreja e filiais associadas ao tenant"
        actions={
          tab === "filiais" && canWrite ? (
            <Button onClick={openCreate}><Plus className="h-4 w-4" /> Nova filial</Button>
          ) : tab === "igrejas" && isSuperAdmin ? (
            <Button onClick={() => { setChurchForm({ ...EMPTY_CHURCH }); setChurchDrawer(true); }}><Plus className="h-4 w-4" /> Nova igreja</Button>
          ) : undefined
        }
      />

      <Tabs
        tabs={[
          { key: "igreja", label: "Dados da igreja", icon: <Church className="h-4 w-4" /> },
          { key: "filiais", label: "Filiais", icon: <Building2 className="h-4 w-4" /> },
          ...(isSuperAdmin
            ? [{ key: "igrejas", label: "Igrejas", icon: <Globe className="h-4 w-4" /> }]
            : []),
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
              <Field label="Razao social">
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
              <Field label="Fuso horario">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.timezone} onChange={(e) => setTenantForm({ ...tenantForm, timezone: e.target.value })} />
              </Field>
              <Field label="Idioma (locale)">
                <Input disabled={!canWrite} className="h-8 text-sm" value={tenantForm.locale} onChange={(e) => setTenantForm({ ...tenantForm, locale: e.target.value })} />
              </Field>
              <Field
                label="Subdominio (slug)"
                className="sm:col-span-2"
                hint={`A igreja responde em ${tenantForm.slug || "..."}.${BASE_DOMAIN}. Alterar muda o endereco: o antigo deixa de funcionar.`}
              >
                <Input
                  disabled={!canWrite}
                  className="h-8 text-sm"
                  value={tenantForm.slug}
                  onChange={(e) => setTenantForm({ ...tenantForm, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })}
                />
              </Field>
              <p className="text-xs text-zinc-400 sm:col-span-2">
                Endereco da igreja: <code className="rounded bg-zinc-100 px-1 dark:bg-zinc-800">{tenantForm.slug || "..."}.{BASE_DOMAIN}</code>
              </p>
              <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-400 sm:col-span-2">White-label (subdominio)</p>
              <Field label="URL do logo" hint="Exibido na tela de login da igreja.">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="https://..." value={tenantForm.logo_url} onChange={(e) => setTenantForm({ ...tenantForm, logo_url: e.target.value })} />
              </Field>
              <Field label="Cor da marca" hint="Hexadecimal, ex.: #0ea5e9.">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="#0ea5e9" value={tenantForm.brand_color} onChange={(e) => setTenantForm({ ...tenantForm, brand_color: e.target.value })} />
              </Field>
              <Field label="URL do favicon">
                <Input disabled={!canWrite} className="h-8 text-sm" placeholder="https://..." value={tenantForm.favicon_url} onChange={(e) => setTenantForm({ ...tenantForm, favicon_url: e.target.value })} />
              </Field>
              <Field label="Dominio proprio" hint="Opcional (ex.: igreja.minhadominio.com).">
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
            <EmptyState icon={<Building2 className="h-10 w-10" />} title="Nenhuma filial" description="Cadastre a Matriz, as filiais (regionais) e os PAEs." />
          ) : (
            <Table>
              <THead><TRow><TH>Nome</TH><TH>Tipo</TH><TH>Identificador</TH><TH>CNPJ</TH><TH>WhatsApp</TH><TH className="text-right">Membros</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
              <TBody>
                {branches.map((b) => (
                  <TRow key={b.id}>
                    <TD className="font-medium">{b.name}</TD>
                    <TD><Badge tone="zinc">{BRANCH_KINDS.find((k) => k.v === b.kind)?.l ?? b.kind}</Badge></TD>
                    <TD className="text-sm text-zinc-500">{b.slug}</TD>
                    <TD className="text-sm text-zinc-500">{b.cnpj ?? "-"}</TD>
                    <TD>
                      {b.whatsapp_status === "connected" ? (
                        <div className="flex flex-col items-start gap-0.5">
                          <Badge tone="green">Conectado</Badge>
                          {(b.whatsapp_number || b.whatsapp_phone) && (
                            <span className="text-xs text-zinc-500">{b.whatsapp_number || b.whatsapp_phone}</span>
                          )}
                        </div>
                      ) : b.whatsapp_status === "connecting" ? (
                        <Badge tone="sky">Aguardando leitura</Badge>
                      ) : (
                        <span className="text-xs text-zinc-400">-</span>
                      )}
                    </TD>
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

      {tab === "igrejas" && isSuperAdmin && (
        <Card className="overflow-hidden p-0">
          {churches === null ? (
            <div className="p-4"><SkeletonRows rows={4} /></div>
          ) : churches.length === 0 ? (
            <EmptyState icon={<Globe className="h-10 w-10" />} title="Nenhuma igreja" description="Crie a primeira igreja; o subdominio passa a funcionar na hora." />
          ) : (
            <Table>
              <THead><TRow><TH>Igreja</TH><TH>Subdominio</TH><TH>Plano</TH><TH className="text-right">Filiais</TH><TH className="text-right">Membros</TH><TH>Situacao</TH></TRow></THead>
              <TBody>
                {churches.map((t) => (
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
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      <Drawer open={churchDrawer} onClose={() => setChurchDrawer(false)} title="Nova igreja">
        <form onSubmit={submitChurch} className="space-y-3">
          <Field label="Nome da igreja *">
            <Input required className="h-8 text-sm" value={churchForm.name} onChange={(e) => setChurchForm({ ...churchForm, name: e.target.value })} />
          </Field>
          <Field label="Subdominio (slug) *" hint={`Minusculas, numeros e hifen. Ex.: matriz -> matriz.${BASE_DOMAIN}`}>
            <Input required className="h-8 text-sm" value={churchForm.slug} onChange={(e) => setChurchForm({ ...churchForm, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, "-") })} />
          </Field>
          <Field label="Plano">
            <Select className="h-8 text-sm" value={churchForm.plan} onChange={(e) => setChurchForm({ ...churchForm, plan: e.target.value })}>
              <option value="starter">Starter</option>
              <option value="pro">Pro</option>
              <option value="enterprise">Enterprise</option>
            </Select>
          </Field>
          <p className="text-[11px] font-semibold uppercase tracking-wide text-zinc-400">Primeiro super_admin</p>
          <Field label="Nome do admin *">
            <Input required className="h-8 text-sm" value={churchForm.admin_name} onChange={(e) => setChurchForm({ ...churchForm, admin_name: e.target.value })} />
          </Field>
          <Field label="E-mail do admin *">
            <Input required type="email" className="h-8 text-sm" value={churchForm.admin_email} onChange={(e) => setChurchForm({ ...churchForm, admin_email: e.target.value })} />
          </Field>
          <Field label="Senha do admin *" hint="Minimo 8 caracteres.">
            <Input required type="password" className="h-8 text-sm" value={churchForm.admin_password} onChange={(e) => setChurchForm({ ...churchForm, admin_password: e.target.value })} />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setChurchDrawer(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={savingChurch}>{savingChurch ? "Criando..." : "Criar igreja"}</Button>
          </div>
        </form>
      </Drawer>

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
            <Field
              label="Unidade superior"
              hint={
                form.kind === "matriz"
                  ? "A Matriz e a raiz: nao tem unidade superior."
                  : form.kind === "pae"
                    ? "Obrigatorio para o PAE: vincule a uma Matriz ou Filial."
                    : "Opcional (a Filial pode reportar a Matriz)."
              }
            >
              <Select
                className="h-8 text-sm"
                disabled={form.kind === "matriz"}
                required={form.kind === "pae"}
                value={form.kind === "matriz" ? "" : form.parent_id}
                onChange={(e) => setForm({ ...form, parent_id: e.target.value })}
              >
                <option value="">-</option>
                {(branches ?? [])
                  .filter((b) => b.id !== drawer.editing?.id)
                  .filter((b) => form.kind !== "pae" || b.kind === "matriz" || b.kind === "filial")
                  .map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
              </Select>
            </Field>
            <Field label="Situacao">
              <Select className="h-8 text-sm" value={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.value })}>
                <option value="true">Ativa</option>
                <option value="false">Inativa</option>
              </Select>
            </Field>
            <Field label="Logradouro" className="sm:col-span-2">
              <Input className="h-8 text-sm" value={form.street} onChange={(e) => setForm({ ...form, street: e.target.value })} />
            </Field>
            <Field label="Numero">
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

        {drawer.editing && (
          <div className="mt-6 space-y-3 border-t border-zinc-100 pt-4 dark:border-zinc-800">
            <h3 className="text-sm font-semibold">Canais de envio</h3>

            {/* WhatsApp proprio da filial (instancia Evolution = id da filial) */}
            <div className="rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
              <div className="mb-2 flex items-center justify-between">
                <span className="text-[13px] font-medium">WhatsApp da filial</span>
                <Badge tone={waStatus === "connected" ? "green" : waStatus === "connecting" ? "sky" : "zinc"}>
                  {waStatus === "connected" ? "Conectado" : waStatus === "connecting" ? "Aguardando QR" : "Desconectado"}
                </Badge>
              </div>
              {waStatus === "connected" && waNumber && (
                <p className="mb-2 text-xs text-zinc-500">Numero conectado: <b>{waNumber}</b></p>
              )}
              <Field label="Telefone (WhatsApp)" hint="Numero exibido/associado a filial.">
                <Input className="h-8 text-sm" placeholder="(11) 90000-0000" value={chForm.whatsapp_phone} onChange={(e) => setChForm({ ...chForm, whatsapp_phone: e.target.value })} />
              </Field>
              <div className="mt-2 flex gap-2">
                <Button type="button" className="h-8 text-sm" disabled={connecting} onClick={connectWhatsApp}>
                  {waStatus === "connected" ? "Reconectar" : "Conectar WhatsApp"}
                </Button>
                {waStatus !== "disconnected" && (
                  <Button type="button" variant="ghost" className="h-8 text-sm" onClick={disconnectWhatsApp}>Desconectar</Button>
                )}
              </div>
              {qr && (
                <div className="mt-3 flex flex-col items-center gap-2">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    alt="QR Code para conectar o WhatsApp"
                    className="h-48 w-48 rounded-md border border-zinc-200 bg-white p-1 dark:border-zinc-700"
                    src={qr.startsWith("data:") ? qr : `data:image/png;base64,${qr}`}
                  />
                  <p className="text-center text-xs text-zinc-500">
                    Abra o WhatsApp da filial {"->"} Aparelhos conectados {"->"} Conectar um aparelho.
                  </p>
                </div>
              )}
            </div>

            {/* SMTP proprio da filial */}
            <div className="rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
              <span className="mb-2 block text-[13px] font-medium">E-mail (SMTP) da filial</span>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <Field label="Servidor SMTP">
                  <Input className="h-8 text-sm" placeholder="smtp.exemplo.com" value={chForm.smtp_host} onChange={(e) => setChForm({ ...chForm, smtp_host: e.target.value })} />
                </Field>
                <Field label="Porta">
                  <Input className="h-8 text-sm" value={chForm.smtp_port} onChange={(e) => setChForm({ ...chForm, smtp_port: e.target.value.replace(/\D/g, "") })} />
                </Field>
                <Field label="Usuario">
                  <Input className="h-8 text-sm" value={chForm.smtp_user} onChange={(e) => setChForm({ ...chForm, smtp_user: e.target.value })} />
                </Field>
                <Field label="Senha" hint="Deixe vazio para manter a atual.">
                  <Input type="password" className="h-8 text-sm" value={chForm.smtp_password} onChange={(e) => setChForm({ ...chForm, smtp_password: e.target.value })} />
                </Field>
                <Field label="Remetente (e-mail)">
                  <Input className="h-8 text-sm" placeholder="contato@igreja.com" value={chForm.smtp_from} onChange={(e) => setChForm({ ...chForm, smtp_from: e.target.value })} />
                </Field>
                <Field label="Nome do remetente">
                  <Input className="h-8 text-sm" value={chForm.smtp_from_name} onChange={(e) => setChForm({ ...chForm, smtp_from_name: e.target.value })} />
                </Field>
              </div>
              <label className="mt-3 flex items-center gap-2 text-sm">
                <input type="checkbox" checked={chForm.smtp_secure} onChange={(e) => setChForm({ ...chForm, smtp_secure: e.target.checked })} />
                Conexao segura (TLS implicito, porta 465)
              </label>
            </div>

            <div className="flex justify-end">
              <Button type="button" className="h-8 text-sm" disabled={savingChannels} onClick={saveChannels}>
                {savingChannels ? "Salvando..." : "Salvar canais"}
              </Button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}
