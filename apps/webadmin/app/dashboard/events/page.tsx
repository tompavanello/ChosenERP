"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  addMonths, eachDayOfInterval, endOfMonth, endOfWeek, format, isSameDay, isSameMonth,
  parseISO, startOfMonth, startOfWeek,
} from "date-fns";
import { ptBR } from "date-fns/locale";
import { CalendarDays, Check, ChevronLeft, ChevronRight, Pencil, Plus, Trash2, Users } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { Tabs } from "@/components/ui/tabs";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listEventKinds, createEventKind, updateEventKind, deleteEventKind,
  listEvents, createEvent, updateEvent, deleteEvent,
  listEventAttendance, saveEventAttendance, listEventInvitees, setEventInvitees,
  listMembers, listMinistries,
  type ChurchEvent, type EventKind, type EventInvitee, type Member, type Ministry,
} from "@/lib/api";
import { currency, dateTimePt } from "@/lib/format";

const EMPTY_EVENT = {
  kind_id: "", date: new Date().toISOString().slice(0, 10), start_time: "19:00",
  end_date: "", end_time: "", attendance_mode: "nominal", participants_count: "",
  estimated_cost: "", notes: "",
};

const pad = (n: number) => String(n).padStart(2, "0");
const dayKey = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
// overlaps informa se o evento cobre o dia (considerando multi-dia).
function overlaps(ev: ChurchEvent, day: Date) {
  const start = new Date(ev.starts_at);
  const end = ev.ends_at ? new Date(ev.ends_at) : start;
  const d = new Date(day.getFullYear(), day.getMonth(), day.getDate());
  const s = new Date(start.getFullYear(), start.getMonth(), start.getDate());
  const e = new Date(end.getFullYear(), end.getMonth(), end.getDate());
  return d >= s && d <= e;
}

export default function EventsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("members.write") || hasPerm("ministries.write");

  const [tab, setTab] = useState("eventos");
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [events, setEvents] = useState<ChurchEvent[] | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [ministries, setMinistries] = useState<Ministry[]>([]);
  const [filters, setFilters] = useState({ from: "", to: "", kind: "" });

  const [eventDrawer, setEventDrawer] = useState<{ open: boolean; editing?: ChurchEvent }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY_EVENT });
  const [present, setPresent] = useState<Set<string>>(new Set());
  const [invMembers, setInvMembers] = useState<Set<string>>(new Set());
  const [invMinistries, setInvMinistries] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [saving, setSaving] = useState(false);

  const [kindDrawer, setKindDrawer] = useState<{ open: boolean; editing?: EventKind }>({ open: false });
  const [kindForm, setKindForm] = useState({ name: "", slug: "", sort_order: "0", is_active: "true", color: "#0ea5e9" });

  const [calMonth, setCalMonth] = useState(() => startOfMonth(new Date()));
  const [calEvents, setCalEvents] = useState<ChurchEvent[]>([]);
  const [dragId, setDragId] = useState<string | null>(null);
  const [hover, setHover] = useState<{ ev: ChurchEvent; x: number; y: number } | null>(null);
  const [inviteeCache, setInviteeCache] = useState<Record<string, EventInvitee[]>>({});

  const load = useCallback(async () => {
    try {
      const [k, e] = await Promise.all([listEventKinds(), listEvents(filters)]);
      setKinds(k.kinds);
      setEvents(e.events);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar eventos", "error");
      setEvents([]);
    }
  }, [filters, toast]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    listMembers().then((r) => setMembers(r.members)).catch(() => setMembers([]));
    listMinistries().then((r) => setMinistries(r.ministries)).catch(() => setMinistries([]));
  }, []);

  const loadCalendar = useCallback(async (month: Date) => {
    const from = format(startOfMonth(month), "yyyy-MM-dd");
    const to = format(endOfMonth(month), "yyyy-MM-dd");
    try {
      const r = await listEvents({ from, to });
      setCalEvents(r.events);
    } catch { setCalEvents([]); }
  }, []);
  useEffect(() => { loadCalendar(calMonth); }, [calMonth, loadCalendar]);

  const filteredMembers = useMemo(() => {
    const q = search.trim().toLowerCase();
    return members.filter((m) => !q || m.full_name.toLowerCase().includes(q));
  }, [members, search]);

  function openCreate() {
    setForm({ ...EMPTY_EVENT, kind_id: kinds.find((k) => k.is_active)?.id ?? "" });
    setPresent(new Set());
    setInvMembers(new Set());
    setInvMinistries(new Set());
    setSearch("");
    setEventDrawer({ open: true });
  }

  async function openEdit(ev: ChurchEvent) {
    const d = new Date(ev.starts_at);
    const e = ev.ends_at ? new Date(ev.ends_at) : null;
    setForm({
      kind_id: ev.kind_id ?? "",
      date: dayKey(d),
      start_time: `${pad(d.getHours())}:${pad(d.getMinutes())}`,
      end_date: e ? dayKey(e) : "",
      end_time: e ? `${pad(e.getHours())}:${pad(e.getMinutes())}` : "",
      attendance_mode: ev.attendance_mode,
      participants_count: String(ev.participants_count || ""),
      estimated_cost: ev.estimated_cost != null ? String(ev.estimated_cost) : "",
      notes: ev.notes ?? "",
    });
    try {
      const [a, inv] = await Promise.all([listEventAttendance(ev.id), listEventInvitees(ev.id)]);
      setPresent(new Set(a.attendance.filter((x) => x.present).map((x) => x.member_id)));
      setInvMembers(new Set(inv.invitees.filter((x) => x.member_id).map((x) => x.member_id as string)));
      setInvMinistries(new Set(inv.invitees.filter((x) => x.ministry_id).map((x) => x.ministry_id as string)));
    } catch {
      setPresent(new Set()); setInvMembers(new Set()); setInvMinistries(new Set());
    }
    setSearch("");
    setEventDrawer({ open: true, editing: ev });
  }

  async function saveEvent(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const total = form.attendance_mode === "count"
        ? Number(form.participants_count) || 0
        : (form.participants_count === "" ? present.size : Number(form.participants_count));
      const endDate = form.end_date || form.date;
      const payload: Record<string, unknown> = {
        kind_id: form.kind_id || undefined,
        starts_at: `${form.date}T${form.start_time}`,
        ends_at: form.end_time ? `${endDate}T${form.end_time}` : undefined,
        participants_count: total,
        attendance_mode: form.attendance_mode,
        estimated_cost: form.estimated_cost === "" ? undefined : Number(form.estimated_cost),
        notes: form.notes || undefined,
      };
      const ev = eventDrawer.editing ? await updateEvent(eventDrawer.editing.id, payload) : await createEvent(payload);
      await saveEventAttendance(ev.id, total, form.attendance_mode === "nominal" ? [...present] : []);
      await setEventInvitees(ev.id, [...invMembers], [...invMinistries]);
      toast("Evento salvo.");
      setEventDrawer({ open: false });
      await load();
      loadCalendar(calMonth);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar evento", "error");
    } finally {
      setSaving(false);
    }
  }

  async function removeEvent(ev: ChurchEvent) {
    if (!confirm("Excluir este evento, a chamada e os convocados?")) return;
    try {
      await deleteEvent(ev.id);
      toast("Evento excluido.");
      await load();
      loadCalendar(calMonth);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  const toggle = (set: React.Dispatch<React.SetStateAction<Set<string>>>, id: string) =>
    set((s) => { const n = new Set(s); if (n.has(id)) n.delete(id); else n.add(id); return n; });

  const kindColor = (kindId?: string) => kinds.find((k) => k.id === kindId)?.color ?? "#94a3b8";

  function openCreateOnDate(day: Date) {
    setForm({ ...EMPTY_EVENT, date: dayKey(day), kind_id: kinds.find((k) => k.is_active)?.id ?? "" });
    setPresent(new Set()); setInvMembers(new Set()); setInvMinistries(new Set()); setSearch("");
    setEventDrawer({ open: true });
  }

  const fmtLocal = (d: Date) =>
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;

  // Move o evento preservando a duracao (um retiro de 3 dias continua de 3 dias).
  async function dropOnDay(day: Date, e: React.DragEvent) {
    e.preventDefault();
    e.stopPropagation();
    const id = dragId; setDragId(null);
    const ev = calEvents.find((x) => x.id === id);
    if (!ev) return;
    const start = new Date(ev.starts_at);
    const end = ev.ends_at ? new Date(ev.ends_at) : null;
    const dur = end ? end.getTime() - start.getTime() : 0;
    const newStart = new Date(day.getFullYear(), day.getMonth(), day.getDate(), start.getHours(), start.getMinutes());
    const newEnd = end ? new Date(newStart.getTime() + dur) : null;
    try {
      await updateEvent(ev.id, {
        starts_at: fmtLocal(newStart),
        ...(newEnd ? { ends_at: fmtLocal(newEnd) } : {}),
      });
      toast("Evento movido.");
      await loadCalendar(calMonth);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao mover evento", "error");
    }
  }

  async function showInvitees(ev: ChurchEvent, e: React.MouseEvent) {
    setHover({ ev, x: e.clientX, y: e.clientY });
    if (!inviteeCache[ev.id]) {
      try {
        const r = await listEventInvitees(ev.id);
        setInviteeCache((c) => ({ ...c, [ev.id]: r.invitees }));
      } catch { /* silencioso */ }
    }
  }

  function openKindCreate() { setKindForm({ name: "", slug: "", sort_order: "0", is_active: "true", color: "#0ea5e9" }); setKindDrawer({ open: true }); }
  function openKindEdit(k: EventKind) { setKindForm({ name: k.name, slug: k.slug, sort_order: String(k.sort_order), is_active: k.is_active ? "true" : "false", color: k.color ?? "#0ea5e9" }); setKindDrawer({ open: true, editing: k }); }
  async function saveKind(e: React.FormEvent) {
    e.preventDefault();
    try {
      const payload = { name: kindForm.name, slug: kindForm.slug, sort_order: Number(kindForm.sort_order) || 0, is_active: kindForm.is_active === "true", color: kindForm.color };
      if (kindDrawer.editing) await updateEventKind(kindDrawer.editing.id, payload);
      else await createEventKind(payload);
      toast("Tipo salvo.");
      setKindDrawer({ open: false });
      await load();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeKind(k: EventKind) {
    try { await deleteEventKind(k.id); toast("Tipo excluido."); await load(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  // Grade do calendario
  const days = useMemo(() => {
    const start = startOfWeek(startOfMonth(calMonth), { weekStartsOn: 0 });
    const end = endOfWeek(endOfMonth(calMonth), { weekStartsOn: 0 });
    return eachDayOfInterval({ start, end });
  }, [calMonth]);

  return (
    <div className="page">
      <PageHeader
        title="Eventos"
        description="Registro de eventos, convocados, chamada e frequencia"
        actions={canWrite ? <Button onClick={openCreate}><Plus className="h-4 w-4" /> Novo evento</Button> : undefined}
      />

      <Tabs
        tabs={[
          { key: "eventos", label: "Eventos", icon: <CalendarDays className="h-4 w-4" /> },
          { key: "calendario", label: "Calendario", icon: <CalendarDays className="h-4 w-4" /> },
          { key: "tipos", label: "Tipos de evento", icon: <Users className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={setTab}
      />

      {tab === "eventos" && (
        <>
          <Card className="mb-4 flex flex-wrap items-end gap-3">
            <div><label className="label">De</label><Input type="date" className="h-8 text-sm" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></div>
            <div><label className="label">Ate</label><Input type="date" className="h-8 text-sm" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></div>
            <div>
              <label className="label">Tipo</label>
              <Select className="h-8 w-56 text-sm" value={filters.kind} onChange={(e) => setFilters({ ...filters, kind: e.target.value })}>
                <option value="">Todos</option>
                {kinds.map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
              </Select>
            </div>
          </Card>

          <Card className="overflow-hidden p-0">
            {events === null ? (
              <div className="p-4"><SkeletonRows rows={5} /></div>
            ) : events.length === 0 ? (
              <EmptyState icon={<CalendarDays className="h-10 w-10" />} title="Nenhum evento" description="Registre cultos, EBD, retiros e demais eventos." />
            ) : (
              <Table>
                <THead><TRow><TH>Periodo</TH><TH>Tipo</TH><TH>Presenca</TH><TH className="text-right">Convocados</TH><TH className="text-right">Custo est.</TH><TH className="text-right">Custo real</TH><TH className="text-right">Acoes</TH></TRow></THead>
                <TBody>
                  {events.map((ev) => (
                    <TRow key={ev.id}>
                      <TD className="font-medium">
                        {dateTimePt(ev.starts_at)}
                        {ev.ends_at && <span className="block text-xs text-zinc-400">ate {dateTimePt(ev.ends_at)}</span>}
                      </TD>
                      <TD><Badge tone="sky">{ev.kind_name ?? "-"}</Badge></TD>
                      <TD>
                        <span className="text-sm">{ev.participants_count} participante(s)</span>
                        <span className="block text-xs text-zinc-400">
                          {ev.attendance_mode === "count" ? "total digitado" : `chamada: ${ev.attendance_count}`}
                        </span>
                      </TD>
                      <TD className="text-right tabular-nums">{ev.invited_count}</TD>
                      <TD className="text-right tabular-nums">{ev.estimated_cost != null ? currency(ev.estimated_cost) : "-"}</TD>
                      <TD className="text-right tabular-nums">{ev.cost_actual > 0 ? currency(ev.cost_actual) : "-"}</TD>
                      <TD>
                        <div className="flex justify-end gap-1">
                          {canWrite && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(ev)}><Pencil className="h-4 w-4" /></Button>}
                          {canWrite && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeEvent(ev)}><Trash2 className="h-4 w-4" /></Button>}
                        </div>
                      </TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            )}
          </Card>
        </>
      )}

      {tab === "calendario" && (
        <Card className="p-4">
          <div className="mb-4 flex items-center justify-between">
            <Button variant="outline" size="sm" onClick={() => setCalMonth((m) => addMonths(m, -1))}><ChevronLeft className="h-4 w-4" /></Button>
            <h3 className="text-sm font-semibold capitalize">{format(calMonth, "MMMM 'de' yyyy", { locale: ptBR })}</h3>
            <Button variant="outline" size="sm" onClick={() => setCalMonth((m) => addMonths(m, 1))}><ChevronRight className="h-4 w-4" /></Button>
          </div>
          <div className="grid grid-cols-7 gap-1 text-center text-[11px] font-medium uppercase text-zinc-400">
            {["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sab"].map((d) => <div key={d} className="py-1">{d}</div>)}
          </div>
          <div className="grid grid-cols-7 gap-1">
            {days.map((day) => {
              const list = calEvents.filter((ev) => overlaps(ev, day));
              const today = isSameDay(day, new Date());
              return (
                <div
                  key={day.toISOString()}
                  onClick={() => canWrite && openCreateOnDate(day)}
                  onDragOver={(e) => e.preventDefault()}
                  onDrop={(e) => dropOnDay(day, e)}
                  className={`min-h-28 cursor-pointer rounded border p-1 text-left transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-800/40 dark:border-zinc-800 ${isSameMonth(day, calMonth) ? "" : "opacity-40"} ${today ? "border-sky-400 bg-sky-50/50 dark:bg-sky-950/20" : "border-zinc-200"}`}
                >
                  <div className="mb-1 text-right text-xs text-zinc-400">{format(day, "d")}</div>
                  <div className="space-y-1">
                    {list.map((ev) => {
                      const color = kindColor(ev.kind_id);
                      return (
                        <button
                          key={ev.id}
                          type="button"
                          draggable
                          onDragStart={() => setDragId(ev.id)}
                          onClick={(e) => { e.stopPropagation(); openEdit(ev); }}
                          onMouseEnter={(e) => showInvitees(ev, e)}
                          onMouseLeave={() => setHover(null)}
                          className="block w-full truncate rounded px-1 py-0.5 text-left text-[11px] hover:opacity-80"
                          style={{ backgroundColor: color + "22", color, borderLeft: `3px solid ${color}` }}
                          title={ev.kind_name ?? "Evento"}
                        >
                          {format(parseISO(ev.starts_at), "HH:mm")} {ev.kind_name ?? "Evento"}
                        </button>
                      );
                    })}
                  </div>
                </div>
              );
            })}
          </div>

          {hover && (
            <div
              className="pointer-events-none fixed z-50 max-w-xs rounded border border-zinc-200 bg-white p-2 text-xs shadow-lg dark:border-zinc-700 dark:bg-zinc-900"
              style={{ left: hover.x + 12, top: hover.y + 12 }}
            >
              <p className="mb-1 font-semibold">Convocados ({(inviteeCache[hover.ev.id] ?? []).length})</p>
              {(inviteeCache[hover.ev.id] ?? []).length === 0 ? (
                <p className="text-zinc-400">Nenhum convocado.</p>
              ) : (
                <ul className="max-h-40 space-y-0.5 overflow-y-auto">
                  {(inviteeCache[hover.ev.id] ?? []).map((i) => (
                    <li key={i.id} className="truncate">
                      {i.ministry_id ? `Ministerio: ${i.ministry_name}` : i.member_name}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </Card>
      )}

      {tab === "tipos" && (
        <Card className="overflow-hidden p-0">
          <div className="flex items-center justify-between border-b border-zinc-100 p-4 dark:border-zinc-800">
            <h3 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">Tipos de evento</h3>
            {canWrite && <Button size="sm" onClick={openKindCreate}><Plus className="h-4 w-4" /> Tipo</Button>}
          </div>
          <Table>
            <THead><TRow><TH>Nome</TH><TH>Slug</TH><TH>Ordem</TH><TH>Ativo</TH><TH className="text-right">Acoes</TH></TRow></THead>
            <TBody>
              {kinds.map((k) => (
                <TRow key={k.id}>
                  <TD className="font-medium">{k.name}</TD>
                  <TD className="text-zinc-500">{k.slug}</TD>
                  <TD className="text-zinc-500">{k.sort_order}</TD>
                  <TD><Badge tone={k.is_active ? "green" : "zinc"}>{k.is_active ? "Sim" : "Nao"}</Badge></TD>
                  <TD>
                    <div className="flex justify-end gap-1">
                      {canWrite && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openKindEdit(k)}><Pencil className="h-4 w-4" /></Button>}
                      {canWrite && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => removeKind(k)}><Trash2 className="h-4 w-4" /></Button>}
                    </div>
                  </TD>
                </TRow>
              ))}
              {kinds.length === 0 && <TRow><TD colSpan={5} className="py-8 text-center text-zinc-400">Nenhum tipo cadastrado.</TD></TRow>}
            </TBody>
          </Table>
        </Card>
      )}

      {/* Drawer do evento */}
      <Drawer open={eventDrawer.open} onClose={() => setEventDrawer({ open: false })} size="2xl" title={eventDrawer.editing ? "Editar evento" : "Novo evento"}>
        <form onSubmit={saveEvent} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <Field label="Tipo de evento">
              <Select className="h-8 text-sm" value={form.kind_id} onChange={(e) => setForm({ ...form, kind_id: e.target.value })}>
                <option value="">-</option>
                {kinds.filter((k) => k.is_active).map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
              </Select>
            </Field>
            <Field label="Custo estimado (R$)" hint="Opcional.">
              <Input type="number" min="0" step="0.01" className="h-8 text-sm" value={form.estimated_cost} onChange={(e) => setForm({ ...form, estimated_cost: e.target.value })} />
            </Field>
            <Field label="Inicio - data *"><Input required type="date" className="h-8 text-sm" value={form.date} onChange={(e) => setForm({ ...form, date: e.target.value })} /></Field>
            <Field label="Inicio - hora *"><Input required type="time" className="h-8 text-sm" value={form.start_time} onChange={(e) => setForm({ ...form, start_time: e.target.value })} /></Field>
            <Field label="Termino - data" hint="Use para eventos de mais de um dia (retiro, acampamento).">
              <Input type="date" className="h-8 text-sm" value={form.end_date} onChange={(e) => setForm({ ...form, end_date: e.target.value })} />
            </Field>
            <Field label="Termino - hora"><Input type="time" className="h-8 text-sm" value={form.end_time} onChange={(e) => setForm({ ...form, end_time: e.target.value })} /></Field>
            <Field label="Registro de presenca">
              <Select className="h-8 text-sm" value={form.attendance_mode} onChange={(e) => setForm({ ...form, attendance_mode: e.target.value })}>
                <option value="nominal">Por chamada (nominal)</option>
                <option value="count">Apenas o numero</option>
              </Select>
            </Field>
            <Field label="Quantidade de presentes" hint={form.attendance_mode === "nominal" ? "Deixe vazio para usar a chamada." : "Total digitado."}>
              <Input type="number" min="0" className="h-8 text-sm" value={form.participants_count} onChange={(e) => setForm({ ...form, participants_count: e.target.value })} />
            </Field>
            <Field label="Observacoes" className="sm:col-span-2 lg:col-span-3">
              <Textarea rows={2} className="text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
            </Field>
          </div>

          <div className="grid gap-3 lg:grid-cols-2">
          {/* Convocados */}
          <div className="rounded border border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center gap-2 border-b border-zinc-100 p-2 text-sm font-semibold dark:border-zinc-800">
              <Users className="h-4 w-4 text-sky-600" /> Convocados (obrigados a participar)
              <span className="ml-auto text-xs font-normal text-zinc-400">{invMembers.size} pessoa(s) - {invMinistries.size} ministerio(s)</span>
            </div>
            <div className="grid gap-2 p-2 sm:grid-cols-2">
              <div>
                <p className="mb-1 text-xs font-medium uppercase text-zinc-400">Ministerios</p>
                <ul className="max-h-36 space-y-0.5 overflow-y-auto">
                  {ministries.map((mi) => (
                    <li key={mi.id}>
                      <label className="flex cursor-pointer items-center gap-2 rounded px-1 py-1 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                        <input type="checkbox" checked={invMinistries.has(mi.id)} onChange={() => toggle(setInvMinistries, mi.id)} className="h-4 w-4 rounded border-zinc-300 text-sky-600" />
                        <span className="truncate">{mi.name}</span>
                      </label>
                    </li>
                  ))}
                  {ministries.length === 0 && <li className="px-1 py-2 text-xs text-zinc-400">Sem ministerios.</li>}
                </ul>
              </div>
              <div>
                <p className="mb-1 text-xs font-medium uppercase text-zinc-400">Pessoas</p>
                <Input className="mb-1 h-8 text-sm" placeholder="Buscar membro..." value={search} onChange={(e) => setSearch(e.target.value)} />
                <ul className="max-h-32 space-y-0.5 overflow-y-auto">
                  {filteredMembers.map((m) => (
                    <li key={m.id}>
                      <label className="flex cursor-pointer items-center gap-2 rounded px-1 py-1 text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800">
                        <input type="checkbox" checked={invMembers.has(m.id)} onChange={() => toggle(setInvMembers, m.id)} className="h-4 w-4 rounded border-zinc-300 text-sky-600" />
                        <span className="truncate">{m.full_name}</span>
                      </label>
                    </li>
                  ))}
                  {filteredMembers.length === 0 && <li className="px-1 py-2 text-xs text-zinc-400">Nenhum membro.</li>}
                </ul>
              </div>
            </div>
          </div>

          {/* Chamada nominal */}
          {form.attendance_mode === "nominal" && (
            <div className="rounded border border-zinc-200 dark:border-zinc-700">
              <div className="flex items-center gap-2 border-b border-zinc-100 p-2 text-sm font-semibold dark:border-zinc-800">
                <Check className="h-4 w-4 text-emerald-600" /> Chamada nominal
                <span className="ml-auto text-xs font-normal text-zinc-400">{present.size} presente(s)</span>
              </div>
              <Input className="m-2 h-8 text-sm" placeholder="Buscar membro para a chamada..." value={search} onChange={(e) => setSearch(e.target.value)} />
              <ul className="max-h-48 overflow-y-auto px-1 pb-2">
                {filteredMembers.map((m) => {
                  const on = present.has(m.id);
                  return (
                    <li key={m.id}>
                      <button type="button" onClick={() => toggle(setPresent, m.id)} className={`flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800 ${on ? "text-emerald-700 dark:text-emerald-400" : ""}`}>
                        <span className="truncate">{m.full_name}</span>
                        {on && <Check className="h-4 w-4" />}
                      </button>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
          </div>

          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setEventDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer do tipo */}
      <Drawer open={kindDrawer.open} onClose={() => setKindDrawer({ open: false })} title={kindDrawer.editing ? "Editar tipo" : "Novo tipo"}>
        <form onSubmit={saveKind} className="space-y-3">
          <Field label="Nome *"><Input required className="h-8 text-sm" value={kindForm.name} onChange={(e) => setKindForm({ ...kindForm, name: e.target.value })} /></Field>
          <Field label="Slug *" hint="Identificador estavel (ex.: culto)."><Input required className="h-8 text-sm" value={kindForm.slug} onChange={(e) => setKindForm({ ...kindForm, slug: e.target.value.toLowerCase().replace(/[^a-z0-9_]+/g, "_") })} /></Field>
          <Field label="Ordem"><Input type="number" className="h-8 text-sm" value={kindForm.sort_order} onChange={(e) => setKindForm({ ...kindForm, sort_order: e.target.value })} /></Field>
          <Field label="Cor">
            <input
              type="color"
              className="h-8 w-20 cursor-pointer rounded border border-zinc-300 bg-transparent dark:border-zinc-700"
              value={kindForm.color}
              onChange={(e) => setKindForm({ ...kindForm, color: e.target.value })}
            />
          </Field>
          <Field label="Situacao">
            <Select className="h-8 text-sm" value={kindForm.is_active} onChange={(e) => setKindForm({ ...kindForm, is_active: e.target.value })}>
              <option value="true">Ativo</option><option value="false">Inativo</option>
            </Select>
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setKindDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm">Salvar</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
