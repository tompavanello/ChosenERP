"use client";

import { useEffect, useState } from "react";
import {
  Bell, Plus, Send, Search, Loader2, CheckCircle, XCircle, Clock, Zap, Users,
  Pencil, CalendarClock, History, Trash2,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";
import { Modal, Drawer } from "@/components/ui/modal";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { useBranches } from "@/lib/swr-hooks";
import {
  listAnnouncements, createAnnouncement, updateAnnouncement, deleteAnnouncement,
  sendAnnouncement, listAnnouncementDeliveries, previewAudience, listMinistries,
  listGroups, listEvents, getNotificationSettings, updateNotificationSettings,
  runNotifications, listNotificationRuns,
  type Announcement, type AnnouncementInput, type SendInput, type AudienceFilter,
  type AnnouncementDelivery, type DeliveryStats, type Ministry, type SmallGroup,
  type ChurchEvent, type NotificationSettings, type AnnouncementRun,
} from "@/lib/api";
import { datePt } from "@/lib/format";
import { GENDER, MARITAL_STATUS, MEMBERSHIP_STATUS } from "@/lib/constants";

const AUDIENCE_OPTIONS = [
  { value: "everyone", label: "Todos (membros + visitantes)" },
  { value: "members", label: "Membros" },
  { value: "visitors", label: "Visitantes" },
  { value: "leaders", label: "Líderes/Ministros" },
  { value: "groups", label: "Grupos específicos" },
  { value: "ministries", label: "Ministérios específicos" },
] as const;

const EMPTY_FILTER: AudienceFilter = {
  group_ids: [], ministry_ids: [], branch_ids: [],
  genders: [], marital_statuses: [], membership_statuses: [],
};

type ScheduleType = "manual" | "once" | "daily" | "event";

type AnnForm = {
  id: string | null;
  title: string;
  body: string;
  audience: string;
  channel: string;
  audience_filter: AudienceFilter;
  schedule_type: ScheduleType;
  schedule_at: string;
  schedule_time: string;
  schedule_event_id: string;
  schedule_offset_minutes: number;
};

type AutomationForm = {
  birthdays_enabled: boolean;
  birthday_template: string;
  roster_reminders_enabled: boolean;
  roster_reminder_hours: number;
  roster_template: string;
  visitor_welcome_enabled: boolean;
  visitor_welcome_delay_hours: number;
  visitor_welcome_template: string;
};

function emptyForm(): AnnForm {
  return {
    id: null, title: "", body: "", audience: "everyone", channel: "whatsapp",
    audience_filter: { ...EMPTY_FILTER },
    schedule_type: "manual", schedule_at: "", schedule_time: "08:00",
    schedule_event_id: "", schedule_offset_minutes: -30,
  };
}

function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function formFromAnnouncement(a: Announcement): AnnForm {
  return {
    id: a.id,
    title: a.title,
    body: a.body,
    audience: a.audience,
    channel: a.channel || "whatsapp",
    audience_filter: { ...EMPTY_FILTER, ...(a.audience_filter ?? {}) },
    schedule_type: (a.schedule_type as ScheduleType) ?? "manual",
    schedule_at: toLocalInput(a.schedule_at),
    schedule_time: a.schedule_time ?? "08:00",
    schedule_event_id: a.schedule_event_id ?? "",
    schedule_offset_minutes: a.schedule_offset_minutes ?? -30,
  };
}

function scheduleLabel(a: Announcement): string {
  switch (a.schedule_type) {
    case "once":
      return a.schedule_at ? `Uma vez · ${new Date(a.schedule_at).toLocaleString("pt-BR")}` : "Uma vez";
    case "daily":
      return a.schedule_time ? `Diário · ${a.schedule_time}` : "Diário";
    case "event": {
      const off = a.schedule_offset_minutes ?? 0;
      if (off < 0) return `Evento · ${Math.abs(off)}min antes`;
      if (off > 0) return `Evento · ${off}min depois`;
      return "Evento · no horário";
    }
    default:
      return "Manual";
  }
}

function runTypeLabel(t: string): string {
  switch (t) {
    case "once": return "Uma vez";
    case "daily": return "Diário";
    case "event": return "Evento";
    default: return t;
  }
}

export default function AnnouncementsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const { data: branchData } = useBranches();
  const [items, setItems] = useState<Announcement[] | null>(null);
  const [query, setQuery] = useState("");

  // Form drawer (criar/editar)
  const [formOpen, setFormOpen] = useState(false);
  const [form, setForm] = useState<AnnForm>(emptyForm());
  const [saving, setSaving] = useState(false);

  // Send modal
  const [sendOpen, setSendOpen] = useState(false);
  const [activeAnn, setActiveAnn] = useState<Announcement | null>(null);
  const [sendForm, setSendForm] = useState<SendInput>({ audience: "everyone", audience_filter: { ...EMPTY_FILTER } });
  const [sending, setSending] = useState(false);
  const [previewing, setPreviewing] = useState(false);
  const [previewCount, setPreviewCount] = useState<number | null>(null);
  const [ministries, setMinistries] = useState<Ministry[]>([]);
  const [groups, setGroups] = useState<SmallGroup[]>([]);

  // Automações
  const [automationOpen, setAutomationOpen] = useState(false);
  const [automation, setAutomation] = useState<AutomationForm | null>(null);
  const [savingAutomation, setSavingAutomation] = useState(false);
  const [runningAutomation, setRunningAutomation] = useState(false);

  // Entregas
  const [detailOpen, setDetailOpen] = useState(false);
  const [deliveries, setDeliveries] = useState<AnnouncementDelivery[] | null>(null);
  const [stats, setStats] = useState<DeliveryStats | null>(null);

  // Execuções agendadas
  const [runsOpen, setRunsOpen] = useState(false);
  const [runs, setRuns] = useState<AnnouncementRun[] | null>(null);

  const canAutomate = hasPerm("settings.write");

  useEffect(() => {
    loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function loadAll() {
    try {
      const [annRes, minRes, grpRes] = await Promise.all([
        listAnnouncements(),
        listMinistries().catch(() => ({ ministries: [] })),
        listGroups().catch(() => ({ groups: [] })),
      ]);
      setItems(annRes.announcements);
      setMinistries(minRes.ministries);
      setGroups(grpRes.groups);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro ao carregar", "error");
    }
  }

  function openCreate() {
    setForm(emptyForm());
    setFormOpen(true);
  }

  function openEdit(a: Announcement) {
    setForm(formFromAnnouncement(a));
    setFormOpen(true);
  }

  async function saveAnn(e: React.FormEvent) {
    e.preventDefault();
    if (!form.title.trim()) {
      toast("Informe o título.", "error");
      return;
    }
    if (form.schedule_type === "once" && !form.schedule_at) {
      toast("Informe a data/hora do agendamento.", "error");
      return;
    }
    if (form.schedule_type === "daily" && !form.schedule_time) {
      toast("Informe o horário do agendamento diário.", "error");
      return;
    }
    if (form.schedule_type === "event" && !form.schedule_event_id) {
      toast("Escolha o evento do agendamento.", "error");
      return;
    }
    const payload = buildPayload(form);
    setSaving(true);
    try {
      if (form.id) {
        await updateAnnouncement(form.id, payload);
        toast("Comunicado atualizado.");
      } else {
        await createAnnouncement(payload);
        toast("Comunicado criado.");
      }
      setFormOpen(false);
      loadAll();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  function openSend(a: Announcement) {
    setActiveAnn(a);
    setSendForm({
      channel: a.channel || "whatsapp",
      audience: a.audience || "everyone",
      audience_filter: { ...EMPTY_FILTER, ...(a.audience_filter ?? {}) },
    });
    setPreviewCount(null);
    setSendOpen(true);
  }

  async function doSend(e: React.FormEvent) {
    e.preventDefault();
    setSending(true);
    try {
      const res = await sendAnnouncement(activeAnn!.id, sendForm);
      toast(`Comunicado enviado para ${res.recipient_count} destinatário(s).`);
      setSendOpen(false);
      loadAll();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro no envio", "error");
    } finally {
      setSending(false);
    }
  }

  async function doPreview() {
    setPreviewing(true);
    try {
      const res = await previewAudience(sendForm);
      setPreviewCount(res.count);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao pré-visualizar", "error");
    } finally {
      setPreviewing(false);
    }
  }

  async function openAutomation() {
    setAutomationOpen(true);
    if (automation) return;
    try {
      const s = await getNotificationSettings();
      setAutomation(toAutomationForm(s));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar automações", "error");
    }
  }

  async function saveAutomation() {
    if (!automation) return;
    setSavingAutomation(true);
    try {
      const s = await updateNotificationSettings(automation);
      setAutomation(toAutomationForm(s));
      toast("Automações salvas.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSavingAutomation(false);
    }
  }

  async function runAutomationNow() {
    setRunningAutomation(true);
    try {
      const res = await runNotifications();
      toast(`${res.queued} mensagem(ns) enfileirada(s).`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao executar", "error");
    } finally {
      setRunningAutomation(false);
    }
  }

  async function viewDeliveries(a: Announcement) {
    setActiveAnn(a);
    const res = await listAnnouncementDeliveries(a.id);
    setDeliveries(res.deliveries);
    setStats(res.stats);
    setDetailOpen(true);
  }

  async function removeAnn(a: Announcement) {
    if (!confirm(`Excluir o comunicado "${a.title}"? As entregas e o histórico de execuções serão removidos.`)) return;
    try {
      await deleteAnnouncement(a.id);
      toast("Comunicado excluído.");
      loadAll();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  async function openRuns() {
    setRunsOpen(true);
    setRuns(null);
    try {
      const res = await listNotificationRuns(100);
      setRuns(res.runs);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar execuções", "error");
      setRuns([]);
    }
  }

  const filtered = (items ?? []).filter((a) => {
    const q = query.trim().toLowerCase();
    return !q || a.title.toLowerCase().includes(q) || a.body.toLowerCase().includes(q);
  });

  return (
    <div className="page space-y-6">
      <PageHeader
        title="Comunicados"
        description="Avisos, disparo em massa segmentado e agendamento por WhatsApp ou e-mail"
        actions={
          <>
            <Button variant="outline" size="sm" onClick={openRuns}>
              <History className="h-4 w-4" /> Execuções
            </Button>
            {canAutomate && (
              <Button variant="outline" size="sm" onClick={openAutomation}>
                <Zap className="h-4 w-4" /> Automações
              </Button>
            )}
            <Button onClick={openCreate}><Plus className="h-4 w-4" /> Novo Comunicado</Button>
          </>
        }
      />

      <div className="relative max-w-sm">
        <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-zinc-400" />
        <Input className="pl-9" placeholder="Buscar por título..." value={query} onChange={(e) => setQuery(e.target.value)} />
      </div>

      {items === null ? (
        <SkeletonRows />
      ) : filtered.length === 0 ? (
        <EmptyState icon={<Bell className="h-10 w-10" />} title="Nenhum comunicado" description="Crie comunicados e dispare por WhatsApp." />
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead>
              <TRow>
                <TH>Título</TH>
                <TH>Público</TH>
                <TH>Agendamento</TH>
                <TH>Status</TH>
                <TH className="text-right">Ações</TH>
              </TRow>
            </THead>
            <TBody>
              {filtered.map((a) => (
                <TRow key={a.id}>
                  <TD className="font-medium">
                    <div>
                      <p>{a.title}</p>
                      <p className="text-xs text-zinc-500 line-clamp-1 max-w-xs">{a.body}</p>
                    </div>
                  </TD>
                  <TD>
                    <Badge tone="sky" variant="outline">
                      {AUDIENCE_OPTIONS.find((o) => o.value === a.audience)?.label ?? a.audience}
                    </Badge>
                  </TD>
                  <TD className="text-zinc-500">
                    <span className="flex items-center gap-1.5 text-xs">
                      <CalendarClock className="h-3.5 w-3.5" /> {scheduleLabel(a)}
                    </span>
                    {a.run_count > 0 && <span className="text-[11px] text-zinc-400">{a.run_count} disparo(s)</span>}
                  </TD>
                  <TD>
                    {a.is_active ? <Badge tone="green">Ativo</Badge> : <Badge tone="zinc">Inativo</Badge>}
                  </TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="ghost" onClick={() => openEdit(a)}>
                        <Pencil className="h-3 w-3" /> Editar
                      </Button>
                      <Button size="sm" variant="outline" onClick={() => viewDeliveries(a)}>
                        <Search className="h-3 w-3" /> Status
                      </Button>
                      <Button size="sm" onClick={() => openSend(a)}>
                        <Send className="h-3 w-3" /> Enviar
                      </Button>
                      <Button size="sm" variant="ghost" onClick={() => removeAnn(a)} title="Excluir">
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      {/* Form Drawer (criar/editar) */}
      <Drawer open={formOpen} onClose={() => setFormOpen(false)} title={form.id ? "Editar Comunicado" : "Novo Comunicado"} size="xl">
        <form onSubmit={saveAnn} className="space-y-4">
          <Field label="Título *">
            <Input required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
          </Field>
          <Field label="Corpo da mensagem">
            <Textarea rows={3} value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} placeholder="Digite a mensagem do comunicado..." />
          </Field>

          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Público-alvo">
              <select
                value={form.audience}
                onChange={(e) => setForm({ ...form, audience: e.target.value })}
                className="input w-full"
              >
                {AUDIENCE_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </Field>
            <Field label="Canal">
              <select
                value={form.channel}
                onChange={(e) => setForm({ ...form, channel: e.target.value })}
                className="input w-full"
              >
                <option value="whatsapp">WhatsApp</option>
                <option value="email">E-mail</option>
              </select>
            </Field>
          </div>

          {form.audience !== "visitors" && (
            <SegmentationFields
              filter={form.audience_filter}
              branches={branchData?.branches ?? []}
              groups={groups}
              ministries={ministries}
              showMembers={form.audience === "groups"}
              showMinistries={form.audience === "ministries"}
              onChange={(f) => setForm({ ...form, audience_filter: f })}
            />
          )}

          <ScheduleFields form={form} setForm={setForm} />

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setFormOpen(false)} disabled={saving}>Cancelar</Button>
            <Button type="submit" disabled={saving}>
              {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {form.id ? "Salvar" : "Criar"}
            </Button>
          </div>
        </form>
      </Drawer>

      {/* Send Modal */}
      <Modal open={sendOpen} onClose={() => setSendOpen(false)} title="Disparar Comunicado" size="xl">
        {activeAnn && (
          <form onSubmit={doSend} className="space-y-4">
            <div className="rounded-lg bg-zinc-50 p-3 dark:bg-zinc-900">
              <p className="font-medium">{activeAnn.title}</p>
              <p className="text-sm text-zinc-600 line-clamp-2 dark:text-zinc-400">{activeAnn.body}</p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Canal">
                <select
                  value={sendForm.channel ?? "whatsapp"}
                  onChange={(e) => setSendForm({ ...sendForm, channel: e.target.value })}
                  className="input w-full"
                >
                  <option value="whatsapp">WhatsApp</option>
                  <option value="email">E-mail</option>
                </select>
              </Field>

              <Field label="Público-alvo">
                <select
                  value={sendForm.audience}
                  onChange={(e) => setSendForm({ ...sendForm, audience: e.target.value as SendInput["audience"], audience_filter: { ...EMPTY_FILTER } })}
                  className="input w-full"
                >
                  {AUDIENCE_OPTIONS.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
                </select>
              </Field>
            </div>

            {sendForm.audience !== "visitors" && (
              <SegmentationFields
                filter={sendForm.audience_filter ?? EMPTY_FILTER}
                branches={branchData?.branches ?? []}
                groups={groups}
                ministries={ministries}
                showMembers={sendForm.audience === "groups"}
                showMinistries={sendForm.audience === "ministries"}
                onChange={(f) => { setSendForm({ ...sendForm, audience_filter: f }); setPreviewCount(null); }}
              />
            )}

            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Button type="button" variant="outline" size="sm" onClick={doPreview} disabled={previewing}>
                  {previewing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Users className="h-4 w-4" />}
                  Pré-visualizar
                </Button>
                {previewCount !== null && (
                  <Badge tone={previewCount > 0 ? "sky" : "amber"}>{previewCount} destinatário(s)</Badge>
                )}
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" type="button" onClick={() => setSendOpen(false)} disabled={sending}>Cancelar</Button>
                <Button type="submit" disabled={sending}>
                  {sending ? (<><Loader2 className="h-4 w-4 animate-spin" /> Enviando...</>) : (<><Send className="h-4 w-4" /> Disparar</>)}
                </Button>
              </div>
            </div>
          </form>
        )}
      </Modal>

      {/* Automation Drawer */}
      <Drawer open={automationOpen} onClose={() => setAutomationOpen(false)} title="Automações de WhatsApp">
        {!automation ? (
          <SkeletonRows />
        ) : (
          <div className="space-y-5">
            <p className="text-sm text-zinc-500">
              Mensagens automáticas enviadas uma única vez por destinatário (com deduplicação).
              Use <code>{"{primeiro_nome}"}</code>, <code>{"{nome}"}</code>, <code>{"{igreja}"}</code>
              {" "}— e nas escalas também <code>{"{titulo}"}</code> e <code>{"{data}"}</code>.
            </p>

            <AutomationBlock
              title="Aniversários"
              enabled={automation.birthdays_enabled}
              onToggle={(v) => setAutomation({ ...automation, birthdays_enabled: v })}
            >
              <Textarea rows={2} value={automation.birthday_template} onChange={(e) => setAutomation({ ...automation, birthday_template: e.target.value })} />
            </AutomationBlock>

            <AutomationBlock
              title="Lembretes de escala"
              enabled={automation.roster_reminders_enabled}
              onToggle={(v) => setAutomation({ ...automation, roster_reminders_enabled: v })}
            >
              <Field label="Antecedência (horas)">
                <Input type="number" min={1} value={automation.roster_reminder_hours} onChange={(e) => setAutomation({ ...automation, roster_reminder_hours: Number(e.target.value) })} />
              </Field>
              <Textarea rows={2} value={automation.roster_template} onChange={(e) => setAutomation({ ...automation, roster_template: e.target.value })} />
            </AutomationBlock>

            <AutomationBlock
              title="Boas-vindas a visitantes"
              enabled={automation.visitor_welcome_enabled}
              onToggle={(v) => setAutomation({ ...automation, visitor_welcome_enabled: v })}
            >
              <Field label="Atraso após o cadastro (horas)">
                <Input type="number" min={1} value={automation.visitor_welcome_delay_hours} onChange={(e) => setAutomation({ ...automation, visitor_welcome_delay_hours: Number(e.target.value) })} />
              </Field>
              <Textarea rows={2} value={automation.visitor_welcome_template} onChange={(e) => setAutomation({ ...automation, visitor_welcome_template: e.target.value })} />
            </AutomationBlock>

            <div className="flex justify-end gap-2 border-t pt-4 dark:border-zinc-800">
              <Button variant="outline" onClick={runAutomationNow} disabled={runningAutomation}>
                {runningAutomation ? <Loader2 className="h-4 w-4 animate-spin" /> : <Zap className="h-4 w-4" />}
                Executar agora
              </Button>
              <Button onClick={saveAutomation} disabled={savingAutomation}>
                {savingAutomation ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                Salvar
              </Button>
            </div>
          </div>
        )}
      </Drawer>

      {/* Execuções Drawer */}
      <Drawer open={runsOpen} onClose={() => setRunsOpen(false)} title="Execuções de agendamento" size="xl">
        {runs === null ? (
          <SkeletonRows />
        ) : runs.length === 0 ? (
          <EmptyState icon={<History className="h-10 w-10" />} title="Nenhuma execução" description="Os disparos agendados aparecem aqui." />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Comunicado</TH>
                <TH>Tipo</TH>
                <TH>Período</TH>
                <TH>Destinatários</TH>
                <TH>Disparado em</TH>
              </TRow>
            </THead>
            <TBody>
              {runs.map((r) => (
                <TRow key={r.id}>
                  <TD className="font-medium">{r.announcement_title}</TD>
                  <TD className="text-zinc-500">{runTypeLabel(r.schedule_type)}</TD>
                  <TD className="text-xs text-zinc-500">{r.period_key ?? "—"}</TD>
                  <TD>{r.recipient_count}</TD>
                  <TD className="text-zinc-500">{new Date(r.fired_at).toLocaleString("pt-BR")}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Drawer>

      {/* Delivery Detail Modal */}
      <Modal open={detailOpen} onClose={() => setDetailOpen(false)} title="Status de Entregas" size="xl">
        {stats && (
          <div className="mb-4 grid grid-cols-4 gap-2">
            <StatCard label="Total" value={String(stats.total)} tone="zinc" />
            <StatCard label="Enviados" value={String(stats.sent)} tone="green" />
            <StatCard label="Falhas" value={String(stats.failed)} tone="red" />
            <StatCard label="Pendentes" value={String(stats.pending)} tone="amber" />
          </div>
        )}
        <div className="max-h-80 overflow-y-auto">
          <Table>
            <THead>
              <TRow>
                <TH>Destino</TH>
                <TH>Origem</TH>
                <TH>Status</TH>
                <TH>Provedor</TH>
                <TH>Tentativas</TH>
                <TH>Erro</TH>
              </TRow>
            </THead>
            <TBody>
              {deliveries?.map((d) => (
                <TRow key={d.id}>
                  <TD className="font-mono text-xs">{d.recipient_name ? `${d.recipient_name} · ` : ""}{d.recipient}</TD>
                  <TD className="text-zinc-500">
                    {d.source === "birthday" ? "Aniversário"
                      : d.source === "roster" ? "Escala"
                      : d.source === "visitor_welcome" ? "Visitante"
                      : d.source === "schedule" ? "Agendado"
                      : "Manual"}
                  </TD>
                  <TD>
                    {d.status === "sent" ? (
                      <Badge tone="green"><CheckCircle className="h-3 w-3" /> Enviado</Badge>
                    ) : d.status === "failed" ? (
                      <Badge tone="red"><XCircle className="h-3 w-3" /> Falhou</Badge>
                    ) : (
                      <Badge tone="amber"><Clock className="h-3 w-3" /> Pendente</Badge>
                    )}
                  </TD>
                  <TD className="text-zinc-500">{d.provider}</TD>
                  <TD>{d.attempts}</TD>
                  <TD className="text-xs text-red-500">{d.error ?? "—"}</TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </div>
      </Modal>
    </div>
  );
}

function buildPayload(f: AnnForm): AnnouncementInput {
  const p: AnnouncementInput = {
    title: f.title,
    body: f.body,
    audience: f.audience,
    channel: f.channel,
    audience_filter: f.audience_filter,
    schedule_type: f.schedule_type,
    schedule_offset_minutes: f.schedule_offset_minutes,
  };
  if (f.schedule_type === "once" && f.schedule_at) p.schedule_at = new Date(f.schedule_at).toISOString();
  if (f.schedule_type === "daily") p.schedule_time = f.schedule_time;
  if (f.schedule_type === "event") p.schedule_event_id = f.schedule_event_id;
  return p;
}

function SegmentationFields({ filter, branches, groups, ministries, showMembers, showMinistries, onChange }: {
  filter: AudienceFilter;
  branches: { id: string; name: string }[];
  groups: SmallGroup[];
  ministries: Ministry[];
  showMembers: boolean;
  showMinistries: boolean;
  onChange: (f: AudienceFilter) => void;
}) {
  function toggle(key: keyof AudienceFilter, value: string) {
    const cur = (filter[key] as string[] | undefined) ?? [];
    const next = cur.includes(value) ? cur.filter((x) => x !== value) : [...cur, value];
    onChange({ ...filter, [key]: next });
  }
  function setAge(key: "age_min" | "age_max", raw: string) {
    onChange({ ...filter, [key]: raw === "" ? undefined : Number(raw) });
  }

  return (
    <div className="space-y-3 rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
      <div className="flex items-center gap-2 text-sm font-medium">
        <Users className="h-4 w-4" /> Segmentação
      </div>

      {showMembers && (
        <Field label="Grupos (células)">
          <div className="max-h-32 space-y-1 overflow-y-auto rounded-lg border p-2">
            {groups.length === 0 ? <p className="text-xs text-zinc-500">Nenhum grupo cadastrado.</p> : groups.map((g) => (
              <label key={g.id} className="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.group_ids ?? []).includes(g.id)} onChange={() => toggle("group_ids", g.id)} className="h-4 w-4" />
                {g.name}
              </label>
            ))}
          </div>
        </Field>
      )}

      {showMinistries && (
        <Field label="Ministérios">
          <div className="max-h-32 space-y-1 overflow-y-auto rounded-lg border p-2">
            {ministries.length === 0 ? <p className="text-xs text-zinc-500">Nenhum ministério cadastrado.</p> : ministries.map((m) => (
              <label key={m.id} className="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.ministry_ids ?? []).includes(m.id)} onChange={() => toggle("ministry_ids", m.id)} className="h-4 w-4" />
                {m.name}
              </label>
            ))}
          </div>
        </Field>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Filial">
          <div className="max-h-28 space-y-1 overflow-y-auto rounded-lg border p-2">
            {branches.map((b) => (
              <label key={b.id} className="flex items-center gap-2 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.branch_ids ?? []).includes(b.id)} onChange={() => toggle("branch_ids", b.id)} className="h-4 w-4" />
                {b.name}
              </label>
            ))}
          </div>
        </Field>

        <Field label="Sexo">
          <div className="flex flex-wrap gap-3 pt-1.5">
            {Object.entries(GENDER).map(([value, label]) => (
              <label key={value} className="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.genders ?? []).includes(value)} onChange={() => toggle("genders", value)} className="h-4 w-4" />
                {label}
              </label>
            ))}
          </div>
        </Field>

        <Field label="Estado civil">
          <div className="flex flex-wrap gap-3 pt-1.5">
            {Object.entries(MARITAL_STATUS).map(([value, label]) => (
              <label key={value} className="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.marital_statuses ?? []).includes(value)} onChange={() => toggle("marital_statuses", value)} className="h-4 w-4" />
                {label}
              </label>
            ))}
          </div>
        </Field>

        <Field label="Situação">
          <div className="flex flex-wrap gap-3 pt-1.5">
            {Object.entries(MEMBERSHIP_STATUS).map(([value, meta]) => (
              <label key={value} className="flex items-center gap-1.5 text-sm cursor-pointer">
                <input type="checkbox" checked={(filter.membership_statuses ?? []).includes(value)} onChange={() => toggle("membership_statuses", value)} className="h-4 w-4" />
                {meta.label}
              </label>
            ))}
          </div>
        </Field>

        <Field label="Faixa etária">
          <div className="flex items-center gap-2">
            <Input type="number" min={0} placeholder="De" value={filter.age_min ?? ""} onChange={(e) => setAge("age_min", e.target.value)} />
            <span className="text-zinc-400">até</span>
            <Input type="number" min={0} placeholder="Até" value={filter.age_max ?? ""} onChange={(e) => setAge("age_max", e.target.value)} />
          </div>
        </Field>
      </div>

      <p className="text-xs text-zinc-500">
        Sexo, estado civil, faixa etária e situação consideram apenas membros — visitantes não têm esses campos.
      </p>
    </div>
  );
}

function ScheduleFields({ form, setForm }: { form: AnnForm; setForm: (f: AnnForm) => void }) {
  const [events, setEvents] = useState<ChurchEvent[]>([]);

  useEffect(() => {
    if (form.schedule_type !== "event" || events.length > 0) return;
    const from = new Date().toISOString();
    const to = new Date(Date.now() + 120 * 24 * 3600 * 1000).toISOString();
    listEvents({ from, to }).then((r) => setEvents(r.events)).catch(() => setEvents([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [form.schedule_type]);

  return (
    <div className="space-y-3 rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
      <div className="flex items-center gap-2 text-sm font-medium">
        <CalendarClock className="h-4 w-4" /> Agendamento
      </div>

      <Field label="Quando disparar">
        <select
          value={form.schedule_type}
          onChange={(e) => setForm({ ...form, schedule_type: e.target.value as ScheduleType })}
          className="input w-full"
        >
          <option value="manual">Manual (só pelo botão Enviar)</option>
          <option value="once">Uma vez, em data/hora</option>
          <option value="daily">Diariamente em um horário</option>
          <option value="event">Em relação a um evento</option>
        </select>
      </Field>

      {form.schedule_type === "once" && (
        <Field label="Data e hora">
          <Input type="datetime-local" value={form.schedule_at} onChange={(e) => setForm({ ...form, schedule_at: e.target.value })} />
        </Field>
      )}

      {form.schedule_type === "daily" && (
        <Field label="Horário (fuso da igreja)">
          <Input type="time" value={form.schedule_time} onChange={(e) => setForm({ ...form, schedule_time: e.target.value })} />
        </Field>
      )}

      {form.schedule_type === "event" && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Evento">
            <select
              value={form.schedule_event_id}
              onChange={(e) => setForm({ ...form, schedule_event_id: e.target.value })}
              className="input w-full"
            >
              <option value="">Selecione...</option>
              {events.map((ev) => (
                <option key={ev.id} value={ev.id}>
                  {ev.kind_name ?? "Evento"} · {new Date(ev.starts_at).toLocaleString("pt-BR")}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Antecedência">
            <select
              value={form.schedule_offset_minutes}
              onChange={(e) => setForm({ ...form, schedule_offset_minutes: Number(e.target.value) })}
              className="input w-full"
            >
              <option value={-1440}>24 horas antes</option>
              <option value={-60}>1 hora antes</option>
              <option value={-30}>30 minutos antes</option>
              <option value={-15}>15 minutos antes</option>
              <option value={0}>No horário do evento</option>
              <option value={15}>15 minutos depois</option>
              <option value={30}>30 minutos depois</option>
              <option value={60}>1 hora depois</option>
            </select>
          </Field>
        </div>
      )}

      {form.schedule_type !== "manual" && (
        <p className="text-xs text-zinc-500">
          O disparo usa a segmentação e o canal configurados acima. Cada destinatário recebe uma única vez por disparo.
        </p>
      )}
    </div>
  );
}

function toAutomationForm(s: NotificationSettings): AutomationForm {
  return {
    birthdays_enabled: s.birthdays_enabled,
    birthday_template: s.birthday_template,
    roster_reminders_enabled: s.roster_reminders_enabled,
    roster_reminder_hours: s.roster_reminder_hours,
    roster_template: s.roster_template,
    visitor_welcome_enabled: s.visitor_welcome_enabled,
    visitor_welcome_delay_hours: s.visitor_welcome_delay_hours,
    visitor_welcome_template: s.visitor_welcome_template,
  };
}

function AutomationBlock({ title, enabled, onToggle, children }: {
  title: string;
  enabled: boolean;
  onToggle: (v: boolean) => void;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-zinc-200 p-3 dark:border-zinc-800">
      <label className="flex items-center justify-between cursor-pointer">
        <span className="text-sm font-medium">{title}</span>
        <input type="checkbox" checked={enabled} onChange={(e) => onToggle(e.target.checked)} className="h-4 w-4" />
      </label>
      <div className={enabled ? "space-y-3" : "space-y-3 opacity-50 pointer-events-none"}>
        {children}
      </div>
    </div>
  );
}
