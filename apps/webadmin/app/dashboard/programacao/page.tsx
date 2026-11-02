"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  addDays, endOfMonth, endOfWeek, format, startOfMonth, startOfWeek,
} from "date-fns";
import { ptBR } from "date-fns/locale";
import {
  CalendarHeart, CalendarPlus, Check, ChevronLeft, ChevronRight, List, Pencil, Plus,
  Trash2, Users,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge, type Tone } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { Tabs } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { AgendaCalendar, type AgendaView } from "@/components/scheduler/agenda-calendar";
import {
  listProgramacoes, createProgramacao, updateProgramacao, deleteProgramacao,
  generateProgramacaoEvents, listAgenda, moveAgendaOccurrence, deleteAgendaOccurrence,
  setAgendaStatus, createEvent, updateEvent, listEventKinds, listMinistries,
  type Programacao, type ChurchEvent, type EventKind, type Ministry,
} from "@/lib/api";
import { WEEKDAYS, WEEKDAY_ORDER, PROGRAM_KINDS, EVENT_ORIGIN } from "@/lib/constants";
import { dateTimePt } from "@/lib/format";

const todayISO = () => format(new Date(), "yyyy-MM-dd");
const timeOf = (d: Date) => format(d, "HH:mm");

const EMPTY = {
  name: "", kind: "culto", title: "", weekday: "0", start_time: "19:00",
  duration_minutes: "90", location: "", notes: "", is_active: "true", sort_order: "0",
  owner_ministry_id: "",
};

const EMPTY_EVENT = {
  kind_id: "", title: "", date: todayISO(), start_time: "19:00", end_time: "",
  owner_ministry_id: "",
};

const VIEWS: { key: AgendaView; label: string }[] = [
  { key: "month", label: "Mes" },
  { key: "week", label: "Semana" },
  { key: "day", label: "Dia" },
  { key: "list", label: "Lista" },
];

function rangeFor(view: AgendaView, anchor: Date) {
  if (view === "week") return { from: startOfWeek(anchor, { weekStartsOn: 0 }), to: addDays(startOfWeek(anchor, { weekStartsOn: 0 }), 6) };
  if (view === "day") return { from: anchor, to: anchor };
  return { from: startOfWeek(startOfMonth(anchor), { weekStartsOn: 0 }), to: endOfWeek(endOfMonth(anchor), { weekStartsOn: 0 }) };
}

function titleOf(ev: ChurchEvent) {
  return ev.title || ev.kind_name || "Evento";
}

/**
 * Agenda da igreja: calendario (mes/semana/dia/lista) com arrastar, excluir e
 * filtros por tipo/ministerio/origem + a grade de horarios recorrentes
 * (modelos) que publica as ocorrencias na agenda. Tudo aponta para a mesma
 * ocorrencia em church_events, onde cada modulo atua (escala, chamada, RSVP).
 */
export default function ProgramacaoPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("members.write") || hasPerm("ministries.write");

  const [tab, setTab] = useState("agenda");
  const [view, setView] = useState<AgendaView>("month");
  const [anchor, setAnchor] = useState(new Date());
  const [filters, setFilters] = useState({ kind: "", ministry: "", origin: "", status: "" });
  const [events, setEvents] = useState<ChurchEvent[] | null>(null);
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [ministries, setMinistries] = useState<Ministry[]>([]);

  const [detail, setDetail] = useState<{ open: boolean; ev?: ChurchEvent }>({ open: false });
  const [createOpen, setCreateOpen] = useState(false);
  const [eventForm, setEventForm] = useState({ ...EMPTY_EVENT });
  const [savingEvent, setSavingEvent] = useState(false);

  const [itens, setItens] = useState<Programacao[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Programacao }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });
  const [saving, setSaving] = useState(false);
  const [publish, setPublish] = useState<{ open: boolean; item?: Programacao }>({ open: false });
  const [pubForm, setPubForm] = useState({ from: todayISO(), to: format(addDays(new Date(), 30), "yyyy-MM-dd") });
  const [publishing, setPublishing] = useState(false);

  useEffect(() => {
    listEventKinds().then((r) => setKinds(r.kinds)).catch(() => setKinds([]));
    listMinistries().then((r) => setMinistries(r.ministries)).catch(() => setMinistries([]));
  }, []);

  const loadAgenda = useCallback(async () => {
    const { from, to } = rangeFor(view, anchor);
    setEvents(null);
    try {
      const r = await listAgenda({
        from: format(from, "yyyy-MM-dd"),
        to: format(to, "yyyy-MM-dd"),
        kind: filters.kind || undefined,
        ministry: filters.ministry || undefined,
        origin: filters.origin || undefined,
        status: filters.status || undefined,
      });
      setEvents(r.events);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar a agenda", "error");
      setEvents([]);
    }
  }, [view, anchor, filters, toast]);

  useEffect(() => { loadAgenda(); }, [loadAgenda]);

  const loadModels = useCallback(async () => {
    try {
      const r = await listProgramacoes();
      setItens(r.programacoes);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar a programacao", "error");
      setItens([]);
    }
  }, [toast]);

  useEffect(() => { if (tab === "modelos") loadModels(); }, [tab, loadModels]);

  const ordenados = useMemo(() => {
    const arr = [...(itens ?? [])];
    arr.sort((a, b) => a.weekday - b.weekday || a.start_time.localeCompare(b.start_time));
    return arr;
  }, [itens]);

  async function move(ev: ChurchEvent, startsAt: string, endsAt: string) {
    try {
      if (ev.origin === "programacao") {
        await moveAgendaOccurrence(ev.id, { starts_at: startsAt, ends_at: endsAt || undefined, scope: "this" });
      } else {
        await updateEvent(ev.id, { starts_at: startsAt, ...(endsAt ? { ends_at: endsAt } : {}) });
      }
      toast(ev.origin === "programacao" ? "Ocorrencia remarcada (somente esta)." : "Evento movido.");
      await loadAgenda();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao mover", "error");
      await loadAgenda();
    }
  }

  async function remove(ev: ChurchEvent) {
    if (ev.origin === "programacao") {
      const all = confirm("Excluir TODA a serie?\n\nOK = serie inteira\nCancelar = somente esta ocorrencia");
      try {
        await deleteAgendaOccurrence(ev.id, all ? "all" : "this");
        toast(all ? "Serie desativada." : "Ocorrencia excluida.");
        await loadAgenda();
      } catch (err) {
        toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
      }
      return;
    }
    if (!confirm("Excluir esta ocorrencia?")) return;
    try {
      await deleteAgendaOccurrence(ev.id, "this");
      toast("Ocorrencia excluida.");
      await loadAgenda();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function openCreateAt(date: Date) {
    setEventForm({ ...EMPTY_EVENT, date: format(date, "yyyy-MM-dd"), start_time: timeOf(date), kind_id: kinds.find((k) => k.is_active)?.id ?? "" });
    setCreateOpen(true);
  }

  async function saveEvent(e: React.FormEvent) {
    e.preventDefault();
    setSavingEvent(true);
    try {
      await createEvent({
        kind_id: eventForm.kind_id || undefined,
        title: eventForm.title || undefined,
        starts_at: `${eventForm.date}T${eventForm.start_time}`,
        ends_at: eventForm.end_time ? `${eventForm.date}T${eventForm.end_time}` : undefined,
        participants_count: 0,
        attendance_mode: "nominal",
        owner_ministry_id: eventForm.owner_ministry_id || undefined,
      });
      toast("Ocorrencia criada.");
      setCreateOpen(false);
      await loadAgenda();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao criar", "error");
    } finally {
      setSavingEvent(false);
    }
  }

  async function changeStatus(status: ChurchEvent["status"]) {
    if (!detail.ev) return;
    try {
      await setAgendaStatus(detail.ev.id, status);
      toast("Situacao atualizada.");
      setDetail({ open: false });
      await loadAgenda();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao atualizar", "error");
    }
  }

  function openCreate() {
    setForm({ ...EMPTY });
    setDrawer({ open: true });
  }

  function openEdit(p: Programacao) {
    setForm({
      name: p.name,
      kind: p.kind,
      title: p.title ?? "",
      weekday: String(p.weekday),
      start_time: p.start_time.slice(0, 5),
      duration_minutes: String(p.duration_minutes || 90),
      location: p.location ?? "",
      notes: p.notes ?? "",
      is_active: p.is_active ? "true" : "false",
      sort_order: String(p.sort_order),
      owner_ministry_id: p.owner_ministry_id ?? "",
    });
    setDrawer({ open: true, editing: p });
  }

  async function saveModel(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.start_time) return;
    setSaving(true);
    const payload = {
      name: form.name.trim(),
      kind: form.kind,
      title: form.kind === "outro" ? form.title : "",
      weekday: Number(form.weekday),
      start_time: form.start_time,
      duration_minutes: Number(form.duration_minutes) || 90,
      location: form.location || undefined,
      notes: form.notes || undefined,
      is_active: form.is_active === "true",
      sort_order: Number(form.sort_order) || 0,
      owner_ministry_id: form.owner_ministry_id || undefined,
    };
    try {
      if (drawer.editing) await updateProgramacao(drawer.editing.id, payload);
      else await createProgramacao(payload);
      toast("Programacao salva.");
      setDrawer({ open: false });
      await loadModels();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function removeModel(p: Programacao) {
    if (!confirm(`Excluir "${p.name}"? As ocorrencias ja publicadas na agenda permanecem.`)) return;
    try {
      await deleteProgramacao(p.id);
      toast("Programacao excluida.");
      await loadModels();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function openPublish(item?: Programacao) {
    setPubForm({ from: todayISO(), to: format(addDays(new Date(), 30), "yyyy-MM-dd") });
    setPublish({ open: true, item });
  }

  async function doPublish(e: React.FormEvent) {
    e.preventDefault();
    if (pubForm.to < pubForm.from) {
      toast("Periodo invalido: 'ate' menor que 'de'.", "error");
      return;
    }
    setPublishing(true);
    try {
      const r = await generateProgramacaoEvents({ from: pubForm.from, to: pubForm.to, programacao_id: publish.item?.id });
      const partes: string[] = [];
      if (r.created) partes.push(`${r.created} nova(s)`);
      if (r.updated) partes.push(`${r.updated} atualizada(s)`);
      toast(partes.length ? `Agenda: ${partes.join(", ")}.` : "Nada a publicar: as ocorrencias ja estavam atualizadas.");
      setPublish({ open: false });
      await Promise.all([loadAgenda(), loadModels()]);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao publicar", "error");
    } finally {
      setPublishing(false);
    }
  }

  const periodLabel = view === "day"
    ? format(anchor, "EEEE, dd 'de' MMMM", { locale: ptBR })
    : view === "week"
      ? `${format(startOfWeek(anchor, { weekStartsOn: 0 }), "dd/MM")} - ${format(addDays(startOfWeek(anchor, { weekStartsOn: 0 }), 6), "dd/MM")}`
      : format(anchor, "MMMM 'de' yyyy", { locale: ptBR });

  return (
    <div className="page">
      <PageHeader
        title="Agenda e programacao"
        description="Calendario unico da igreja: arraste para remarcar, exclua e filtre por ministerio"
        actions={canWrite ? (
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => openPublish()}><CalendarPlus className="h-4 w-4" /> Publicar grade</Button>
            <Button onClick={() => openCreateAt(new Date())}><Plus className="h-4 w-4" /> Nova ocorrencia</Button>
          </div>
        ) : undefined}
      />

      <Tabs
        tabs={[
          { key: "agenda", label: "Agenda", icon: <CalendarHeart className="h-4 w-4" /> },
          { key: "modelos", label: "Modelos recorrentes", icon: <List className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "agenda" ? (
        <>
          <Card className="mb-3 p-3">
            <div className="flex flex-wrap items-center gap-2">
              <div className="flex items-center gap-1">
                <Button variant="ghost" className="h-8 px-2" onClick={() => setAnchor(addDays(anchor, view === "month" ? -30 : view === "week" ? -7 : -1))}><ChevronLeft className="h-4 w-4" /></Button>
                <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => setAnchor(new Date())}>Hoje</Button>
                <Button variant="ghost" className="h-8 px-2" onClick={() => setAnchor(addDays(anchor, view === "month" ? 30 : view === "week" ? 7 : 1))}><ChevronRight className="h-4 w-4" /></Button>
              </div>
              <span className="text-sm font-medium capitalize text-zinc-700 dark:text-zinc-200">{periodLabel}</span>
              <div className="ml-auto flex flex-wrap items-center gap-1">
                {VIEWS.map((v) => (
                  <Button key={v.key} variant={view === v.key ? "outline" : "ghost"} className="h-8 px-2 text-xs" onClick={() => setView(v.key)}>
                    {v.label}
                  </Button>
                ))}
              </div>
            </div>
            <div className="mt-2 flex flex-wrap items-end gap-2 border-t border-zinc-100 pt-2 dark:border-zinc-800">
              <Select className="h-8 w-36 text-xs" value={filters.kind} onChange={(e) => setFilters({ ...filters, kind: e.target.value })}>
                <option value="">Todos os tipos</option>
                {kinds.map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
              </Select>
              <Select className="h-8 w-40 text-xs" value={filters.ministry} onChange={(e) => setFilters({ ...filters, ministry: e.target.value })}>
                <option value="">Todos os ministerios</option>
                {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
              </Select>
              <Select className="h-8 w-32 text-xs" value={filters.origin} onChange={(e) => setFilters({ ...filters, origin: e.target.value })}>
                <option value="">Toda origem</option>
                {Object.entries(EVENT_ORIGIN).map(([v, o]) => <option key={v} value={v}>{o.label}</option>)}
              </Select>
              <Select className="h-8 w-32 text-xs" value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}>
                <option value="">Toda situacao</option>
                <option value="planned">Planejado</option>
                <option value="confirmed">Confirmado</option>
                <option value="done">Concluido</option>
                <option value="cancelled">Cancelado</option>
              </Select>
              {(filters.kind || filters.ministry || filters.origin || filters.status) && (
                <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => setFilters({ kind: "", ministry: "", origin: "", status: "" })}>
                  Limpar
                </Button>
              )}
              <span className="ml-auto text-xs text-zinc-400">{events?.length ?? 0} ocorrencia(s)</span>
            </div>
          </Card>

          {events === null ? (
            <Card className="p-4"><SkeletonRows rows={8} /></Card>
          ) : (
            <AgendaCalendar
              events={events}
              view={view}
              anchor={anchor}
              kinds={kinds}
              canWrite={canWrite}
              onOpen={(ev) => setDetail({ open: true, ev })}
              onMove={move}
              onDelete={remove}
              onCreateAt={openCreateAt}
            />
          )}
          <p className="mt-2 text-xs text-zinc-400">
            Arraste um cartao para remarcar. Em ocorrencias de programacao, o ajuste vale para esta data e nao volta ao publicar a grade.
          </p>
        </>
      ) : (
        <>
          <Card className="mb-3 p-3">
            <p className="text-xs text-zinc-500">
              Defina os horarios fixos. Ao <span className="font-medium">publicar</span>, cada ocorrencia do periodo vira um evento na agenda.
              Publicar de novo nao duplica nem recria ocorrencias canceladas/remarcadas.
            </p>
          </Card>

          <Card className="overflow-hidden p-0">
            {itens === null ? (
              <div className="p-4"><SkeletonRows rows={5} /></div>
            ) : ordenados.length === 0 ? (
              <EmptyState
                icon={<CalendarHeart className="h-10 w-10" />}
                title="Nenhum horario na grade"
                description="Cadastre os horarios fixos (Culto de Domingo 19:00, Relogio de Oracao...) e publique na agenda."
              />
            ) : (
              <Table>
                <THead>
                  <TRow>
                    <TH>Tipo</TH><TH>Dia</TH><TH>Horario</TH><TH>Nome</TH><TH>Ministerio</TH>
                    <TH>Local</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH>
                  </TRow>
                </THead>
                <TBody>
                  {ordenados.map((p) => {
                    const k = PROGRAM_KINDS[p.kind] ?? PROGRAM_KINDS.outro;
                    return (
                      <TRow key={p.id}>
                        <TD><Badge tone={k.tone as Tone} className="text-[10px]">{p.kind === "outro" && p.title ? p.title : k.label}</Badge></TD>
                        <TD className="text-sm text-zinc-500">{WEEKDAYS[p.weekday] ?? p.weekday}</TD>
                        <TD className="font-medium tabular-nums">{p.start_time}</TD>
                        <TD className="font-medium">{p.name}</TD>
                        <TD className="text-sm text-zinc-500">{p.owner_ministry_name ?? "-"}</TD>
                        <TD className="text-sm text-zinc-500">{p.location ?? "-"}</TD>
                        <TD><Badge tone={p.is_active ? "green" : "zinc"} className="text-[10px]">{p.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                        <TD>
                          {canWrite && (
                            <div className="flex justify-end gap-1">
                              <Button variant="ghost" className="h-8 px-2" title="Publicar na agenda" onClick={() => openPublish(p)}><CalendarPlus className="h-4 w-4" /></Button>
                              <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(p)}><Pencil className="h-4 w-4" /></Button>
                              <Button variant="ghost" className="h-8 px-2 text-red-600" title="Excluir" onClick={() => removeModel(p)}><Trash2 className="h-4 w-4" /></Button>
                            </div>
                          )}
                        </TD>
                      </TRow>
                    );
                  })}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {/* Detalhe da ocorrencia */}
      <Drawer open={detail.open} onClose={() => setDetail({ open: false })} title={detail.ev ? titleOf(detail.ev) : "Ocorrencia"}>
        {detail.ev && (
          <div className="space-y-4">
            <div className="rounded-lg border border-zinc-100 p-3 text-sm dark:border-zinc-800">
              <p className="font-medium">{format(new Date(detail.ev.starts_at), "dd/MM/yyyy", { locale: ptBR })}</p>
              <p className="text-zinc-500">
                {format(new Date(detail.ev.starts_at), "HH:mm")}
                {detail.ev.ends_at ? ` - ${format(new Date(detail.ev.ends_at), "HH:mm")}` : ""}
              </p>
              <p className="mt-1 text-xs text-zinc-400">
                {detail.ev.kind_name ?? "Sem tipo"} · {EVENT_ORIGIN[detail.ev.origin]?.label ?? detail.ev.origin}
                {detail.ev.owner_ministry_name ? ` · ${detail.ev.owner_ministry_name}` : ""}
              </p>
              {detail.ev.notes && <p className="mt-2 text-xs text-zinc-500">{detail.ev.notes}</p>}
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <Info label="Situacao" value={detail.ev.status} />
              <Info label="Participantes" value={String(detail.ev.participants_count)} />
              <Info label="Convidados" value={String(detail.ev.invited_count)} />
              <Info label="Confirmacoes" value={String(detail.ev.attendance_count)} />
            </div>
            {canWrite && (
              <div className="space-y-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
                <p className="text-xs font-medium text-zinc-500">Situacao</p>
                <div className="flex flex-wrap gap-1">
                  {(["planned", "confirmed", "done", "cancelled"] as const).map((s) => (
                    <Button key={s} variant={detail.ev!.status === s ? "primary" : "outline"} className="h-7 px-2 text-xs" onClick={() => changeStatus(s)}>
                      {detail.ev!.status === s && <Check className="mr-1 h-3 w-3" />}
                      {s === "planned" ? "Planejado" : s === "confirmed" ? "Confirmado" : s === "done" ? "Concluido" : "Cancelado"}
                    </Button>
                  ))}
                </div>
                <div className="flex flex-wrap gap-2 pt-2">
                  <Link href="/dashboard/events">
                    <Button variant="outline" className="h-8 text-xs"><Users className="mr-1 h-3.5 w-3.5" /> Chamada e convidados</Button>
                  </Link>
                  <Link href="/dashboard/rosters">
                    <Button variant="outline" className="h-8 text-xs">Escalas</Button>
                  </Link>
                  <Button variant="ghost" className="h-8 text-xs text-red-600" onClick={() => { setDetail({ open: false }); remove(detail.ev!); }}>
                    <Trash2 className="mr-1 h-3.5 w-3.5" /> Excluir
                  </Button>
                </div>
              </div>
            )}
            <p className="text-[11px] text-zinc-400">Criado em {dateTimePt(detail.ev.created_at)}.</p>
          </div>
        )}
      </Drawer>

      {/* Nova ocorrencia */}
      <Drawer open={createOpen} onClose={() => setCreateOpen(false)} title="Nova ocorrencia">
        <form onSubmit={saveEvent} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Tipo">
            <Select className="h-8 text-sm" value={eventForm.kind_id} onChange={(e) => setEventForm({ ...eventForm, kind_id: e.target.value })}>
              <option value="">Sem tipo</option>
              {kinds.map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
            </Select>
          </Field>
          <Field label="Titulo" hint="Opcional; se vazio usa o tipo.">
            <Input className="h-8 text-sm" value={eventForm.title} onChange={(e) => setEventForm({ ...eventForm, title: e.target.value })} />
          </Field>
          <Field label="Data" required>
            <Input required type="date" className="h-8 text-sm" value={eventForm.date} onChange={(e) => setEventForm({ ...eventForm, date: e.target.value })} />
          </Field>
          <Field label="Inicio" required>
            <Input required type="time" className="h-8 text-sm" value={eventForm.start_time} onChange={(e) => setEventForm({ ...eventForm, start_time: e.target.value })} />
          </Field>
          <Field label="Fim">
            <Input type="time" className="h-8 text-sm" value={eventForm.end_time} onChange={(e) => setEventForm({ ...eventForm, end_time: e.target.value })} />
          </Field>
          <Field label="Ministerio responsavel">
            <Select className="h-8 text-sm" value={eventForm.owner_ministry_id} onChange={(e) => setEventForm({ ...eventForm, owner_ministry_id: e.target.value })}>
              <option value="">Sem ministerio</option>
              {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
            </Select>
          </Field>
          <div className="flex justify-end gap-2 sm:col-span-2">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setCreateOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={savingEvent}>{savingEvent ? "Salvando..." : "Criar"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer do modelo */}
      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} title={drawer.editing ? "Editar programacao" : "Nova programacao"}>
        <form onSubmit={saveModel} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Tipo" required>
            <Select className="h-8 text-sm" value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
              {Object.entries(PROGRAM_KINDS).map(([v, k]) => <option key={v} value={v}>{k.label}</option>)}
            </Select>
          </Field>
          {form.kind === "outro" && (
            <Field label="Digite o titulo" hint="Aparece na agenda no lugar de 'Outro'.">
              <Input className="h-8 text-sm" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder="Ex: Ensaio de teatro" />
            </Field>
          )}
          <Field label="Nome" required>
            <Input required className="h-8 text-sm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Ex: Culto de Domingo" />
          </Field>
          <Field label="Dia da semana" required>
            <Select className="h-8 text-sm" value={form.weekday} onChange={(e) => setForm({ ...form, weekday: e.target.value })}>
              {WEEKDAY_ORDER.map((d) => <option key={d} value={d}>{WEEKDAYS[d]}</option>)}
            </Select>
          </Field>
          <Field label="Horario" required>
            <Input required type="time" className="h-8 text-sm" value={form.start_time} onChange={(e) => setForm({ ...form, start_time: e.target.value })} />
          </Field>
          <Field label="Duracao (minutos)">
            <Input type="number" min="15" step="15" className="h-8 text-sm" value={form.duration_minutes} onChange={(e) => setForm({ ...form, duration_minutes: e.target.value })} />
          </Field>
          <Field label="Ministerio responsavel" hint="Alimenta a visao 'Minha agenda'.">
            <Select className="h-8 text-sm" value={form.owner_ministry_id} onChange={(e) => setForm({ ...form, owner_ministry_id: e.target.value })}>
              <option value="">Sem ministerio</option>
              {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
            </Select>
          </Field>
          <Field label="Local">
            <Input className="h-8 text-sm" value={form.location} onChange={(e) => setForm({ ...form, location: e.target.value })} placeholder="Ex: Templo principal" />
          </Field>
          <Field label="Situacao">
            <Select className="h-8 text-sm" value={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.value })}>
              <option value="true">Ativo</option><option value="false">Inativo</option>
            </Select>
          </Field>
          <Field label="Observacoes" className="sm:col-span-2">
            <Textarea rows={2} className="text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} placeholder="Ex: transmissao ao vivo, escala do louvor..." />
          </Field>
          <div className="flex justify-end gap-2 sm:col-span-2">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Publicar */}
      <Modal open={publish.open} onClose={() => setPublish({ open: false })} title="Publicar na agenda" size="sm">
        <form onSubmit={doPublish} className="space-y-3">
          <p className="text-xs text-zinc-500">
            {publish.item
              ? <>Publica as ocorrencias de <span className="font-medium">{publish.item.name}</span> no periodo.</>
              : <>Publica as ocorrencias de <span className="font-medium">toda a programacao ativa</span> no periodo.</>}
          </p>
          <div className="grid grid-cols-2 gap-3">
            <Field label="De" required>
              <Input required type="date" className="h-8 text-sm" value={pubForm.from} onChange={(e) => setPubForm({ ...pubForm, from: e.target.value })} />
            </Field>
            <Field label="Ate" required>
              <Input required type="date" className="h-8 text-sm" value={pubForm.to} onChange={(e) => setPubForm({ ...pubForm, to: e.target.value })} />
            </Field>
          </div>
          <p className="text-xs text-zinc-400">Idempotente: ocorrencias ja publicadas, canceladas ou remarcadas sao respeitadas.</p>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setPublish({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={publishing}>{publishing ? "Publicando..." : "Publicar"}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

function Info({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-zinc-50 p-2 dark:bg-zinc-900/50">
      <p className="text-[10px] uppercase tracking-wide text-zinc-400">{label}</p>
      <p className="mt-0.5 font-medium capitalize">{value}</p>
    </div>
  );
}
