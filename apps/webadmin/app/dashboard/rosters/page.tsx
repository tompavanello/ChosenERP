"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  addDays, endOfMonth, endOfWeek, format, startOfMonth, startOfWeek,
} from "date-fns";
import { ptBR } from "date-fns/locale";
import {
  AlertTriangle, CalendarClock, CalendarDays, Check, ChevronLeft, ChevronRight,
  Copy, Download, FileSpreadsheet, FileText, List, MoreHorizontal, Pencil, Plus,
  Trash2, UserPlus, Users, X,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { Dropdown } from "@/components/ui/dropdown";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { Tabs } from "@/components/ui/tabs";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { RosterCalendar } from "@/components/scheduler/roster-calendar";
import type { CalendarView } from "@/components/scheduler/calendar-types";
import { ExportMenu } from "@/components/reports/export-menu";
import { KpiCard } from "@/components/reports/kpi-card";
import {
  listRosters, createRoster, updateRoster, deleteRoster, getRoster,
  setRosterAssignments, respondRosterAssignment, getRosterConflicts, getRosterSuggestions,
  getRosterConflictsByPeriod, downloadExport, openPrint,
  listMinistries, listMembers, listEvents, listEventKinds,
  type Roster, type RosterConflict, type RosterPeriodConflict, type RosterSuggestion,
  type Ministry, type Member, type ChurchEvent, type EventKind,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const STATUS_TONE: Record<string, "zinc" | "sky" | "green" | "red"> = {
  rascunho: "zinc", publicada: "sky", concluida: "green", cancelada: "red",
};
const ASSIGN_TONE: Record<string, "zinc" | "green" | "red"> = {
  convidado: "zinc", confirmado: "green", recusado: "red",
};

const pad = (n: number) => String(n).padStart(2, "0");
const localISO = (d: Date) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
const localNow = () => localISO(new Date());

const EMPTY = {
  title: "", ministry_id: "", event_id: "", event_kind_id: "", create_event: false,
  starts_at: localNow(), ends_at: "", location: "", notes: "", status: "rascunho",
};

const VIEWS: { key: CalendarView; label: string }[] = [
  { key: "month", label: "Mes" },
  { key: "week", label: "Semana" },
  { key: "day", label: "Dia" },
  { key: "list", label: "Lista" },
];

function rangeFor(view: CalendarView, anchor: Date) {
  if (view === "week") {
    const s = startOfWeek(anchor, { weekStartsOn: 0 });
    return { from: s, to: addDays(s, 6) };
  }
  if (view === "day") return { from: anchor, to: anchor };
  return {
    from: startOfWeek(startOfMonth(anchor), { weekStartsOn: 0 }),
    to: endOfWeek(endOfMonth(anchor), { weekStartsOn: 0 }),
  };
}

export default function RostersPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("ministries.write");

  const [tab, setTab] = useState<"escalas" | "calendario">("escalas");
  const [view, setView] = useState<CalendarView>("month");
  const [anchor, setAnchor] = useState(new Date());
  const [rosters, setRosters] = useState<Roster[] | null>(null);
  const [periodConflicts, setPeriodConflicts] = useState<RosterPeriodConflict[]>([]);
  const [calendarEvents, setCalendarEvents] = useState<ChurchEvent[]>([]);
  const [ministries, setMinistries] = useState<Ministry[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [filters, setFilters] = useState({ from: "", to: "", ministry: "", status: "" });
  const [saving, setSaving] = useState(false);

  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Roster }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });

  const [manage, setManage] = useState<Roster | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Roster | null>(null);
  const [selected, setSelected] = useState<Record<string, string>>({});
  const [conflicts, setConflicts] = useState<RosterConflict[]>([]);
  const [suggestions, setSuggestions] = useState<RosterSuggestion[]>([]);
  const [memberSearch, setMemberSearch] = useState("");
  const [assignFilter, setAssignFilter] = useState<"todos" | "confirmado" | "convidado" | "recusado">("todos");

  const range = useMemo(() => {
    if (tab === "calendario") {
      const r = rangeFor(view, anchor);
      return { from: format(r.from, "yyyy-MM-dd"), to: format(r.to, "yyyy-MM-dd") };
    }
    return { from: filters.from, to: filters.to };
  }, [tab, view, anchor, filters.from, filters.to]);

  const load = useCallback(async () => {
    try {
      const [r, c, e] = await Promise.all([
        listRosters({ from: range.from || undefined, to: range.to || undefined, ministry: filters.ministry || undefined }),
        getRosterConflictsByPeriod({ from: range.from || undefined, to: range.to || undefined, ministry: filters.ministry || undefined }),
        tab === "calendario"
          ? listEvents({ from: range.from || undefined, to: range.to || undefined }).catch(() => ({ events: [] as ChurchEvent[] }))
          : Promise.resolve({ events: [] as ChurchEvent[] }),
      ]);
      setRosters(r.rosters);
      setPeriodConflicts(c.conflicts);
      // O calendario de escalas ja mostra as escalas; evita duplicar o evento
      // gerado por elas no fundo.
      setCalendarEvents(e.events.filter((ev) => ev.origin !== "escala"));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar escalas", "error");
      setRosters([]);
      setPeriodConflicts([]);
      setCalendarEvents([]);
    }
  }, [range.from, range.to, filters.ministry, tab, toast]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    listMinistries().then((r) => setMinistries(r.ministries)).catch(() => setMinistries([]));
    listMembers().then((r) => setMembers(r.members)).catch(() => setMembers([]));
    listEvents().then((r) => setEvents(r.events)).catch(() => setEvents([]));
    listEventKinds().then((r) => setKinds(r.kinds)).catch(() => setKinds([]));
  }, []);

  const memberName = (id: string) => members.find((m) => m.id === id)?.full_name ?? id;
  const conflictRosterIds = useMemo(() => {
    const set = new Set<string>();
    for (const c of periodConflicts) { set.add(c.roster_a_id); set.add(c.roster_b_id); }
    return set;
  }, [periodConflicts]);

  const visibleRosters = useMemo(
    () => (rosters ?? []).filter((r) => !filters.status || r.status === filters.status),
    [rosters, filters.status],
  );

  const totals = useMemo(() => {
    const list = visibleRosters;
    const assignments = list.reduce((s, r) => s + r.assignment_count, 0);
    const confirmed = list.reduce((s, r) => s + r.confirmed_count, 0);
    return {
      count: list.length,
      assignments,
      confirmed,
      pending: Math.max(0, assignments - confirmed),
      noTeam: list.filter((r) => r.assignment_count === 0).length,
      fill: assignments > 0 ? (confirmed / assignments) * 100 : 0,
    };
  }, [visibleRosters]);

  // ---- CRUD ----
  function openCreate() {
    setForm({ ...EMPTY, ministry_id: filters.ministry });
    setDrawer({ open: true });
  }
  function openCreateAt(date: Date) {
    setForm({ ...EMPTY, starts_at: localISO(date), ministry_id: filters.ministry });
    setDrawer({ open: true });
  }
  function openEdit(r: Roster) {
    setForm({
      title: r.title, ministry_id: r.ministry_id ?? "", event_id: r.event_id ?? "",
      event_kind_id: r.event_kind_id ?? "", create_event: false,
      starts_at: r.starts_at.slice(0, 16), ends_at: r.ends_at ? r.ends_at.slice(0, 16) : "",
      location: r.location ?? "", notes: r.notes ?? "", status: r.status,
    });
    setDrawer({ open: true, editing: r });
  }
  async function saveRoster(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload: Record<string, unknown> = {
        title: form.title,
        ministry_id: form.ministry_id || undefined,
        event_id: form.event_id || undefined,
        event_kind_id: form.event_kind_id || undefined,
        create_event: form.create_event,
        starts_at: form.starts_at,
        ends_at: form.ends_at || undefined,
        location: form.location || undefined,
        notes: form.notes || undefined,
        status: form.status,
      };
      if (drawer.editing) await updateRoster(drawer.editing.id, payload);
      else await createRoster(payload);
      toast("Escala salva.");
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar escala", "error");
    } finally {
      setSaving(false);
    }
  }
  async function duplicateRoster(r: Roster) {
    try {
      const full = await getRoster(r.id);
      const created = await createRoster({
        title: `${r.title} (copia)`,
        ministry_id: r.ministry_id || undefined,
        event_kind_id: r.event_kind_id || undefined,
        starts_at: r.starts_at,
        ends_at: r.ends_at || undefined,
        location: r.location || undefined,
        notes: r.notes || undefined,
        status: "rascunho",
        assignments: (full.assignments ?? []).map((a) => ({ member_id: a.member_id, role: a.role })),
      });
      toast("Escala duplicada em rascunho.");
      await load();
      openEdit(created);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao duplicar", "error");
    }
  }
  async function removeRoster(r: Roster, deleteEvent: boolean) {
    try {
      await deleteRoster(r.id, deleteEvent);
      toast(deleteEvent ? "Escala e evento excluidos." : "Escala excluida.");
      setDeleteTarget(null);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  async function moveRoster(r: Roster, startsAt: string, endsAt: string) {
    if (!canWrite) return;
    if (r.status === "cancelada") {
      toast("Escala cancelada nao pode ser remarcada.", "info");
      await load();
      return;
    }
    if (r.status === "publicada" && !confirm(
      "Esta escala esta publicada. Remarcar move tambem o evento gerado. Continuar?",
    )) {
      await load();
      return;
    }
    try {
      await updateRoster(r.id, { starts_at: startsAt, ...(endsAt ? { ends_at: endsAt } : {}) });
      toast(r.generated_event ? "Escala e evento remarcados." : "Escala remarcada.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao remarcar", "error");
      await load();
    }
  }

  // ---- Gerenciar escalados ----
  async function openManage(r: Roster) {
    try {
      const full = await getRoster(r.id);
      setManage(full);
      const map: Record<string, string> = {};
      for (const a of full.assignments ?? []) map[a.member_id] = a.role ?? "";
      setSelected(map);
      setConflicts([]);
      setSuggestions([]);
      setMemberSearch("");
      setAssignFilter("todos");
      refreshConflicts(full.id);
      if (full.ministry_id) loadSuggestions(full);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao abrir escala", "error");
    }
  }
  async function loadSuggestions(r: Roster) {
    if (!r.ministry_id) return;
    try {
      const res = await getRosterSuggestions({ ministry_id: r.ministry_id, starts_at: r.starts_at, ends_at: r.ends_at, roster_id: r.id });
      setSuggestions(res.suggestions);
    } catch { setSuggestions([]); }
  }
  async function refreshConflicts(id: string) {
    try { setConflicts((await getRosterConflicts(id)).conflicts); }
    catch { setConflicts([]); }
  }
  async function saveAssignments() {
    if (!manage) return;
    try {
      const list = Object.entries(selected).map(([member_id, role]) => ({ member_id, role: role || undefined }));
      const updated = await setRosterAssignments(manage.id, list);
      setManage(updated);
      toast("Escalados salvos.");
      await load();
      await refreshConflicts(manage.id);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar escalados", "error");
    }
  }
  async function respond(assignmentId: string, status: "confirmado" | "recusado" | "convidado") {
    if (!manage) return;
    try {
      await respondRosterAssignment(manage.id, assignmentId, status);
      const full = await getRoster(manage.id);
      setManage(full);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }
  async function copyLastTeam() {
    if (!manage?.ministry_id) {
      toast("Vincule um ministerio para copiar a equipe anterior.", "info");
      return;
    }
    try {
      const res = await listRosters({ ministry: manage.ministry_id, to: manage.starts_at.slice(0, 10) });
      const previous = res.rosters
        .filter((r) => r.id !== manage.id && r.starts_at < manage.starts_at)
        .sort((a, b) => b.starts_at.localeCompare(a.starts_at))[0];
      if (!previous) {
        toast("Nenhuma escala anterior deste ministerio encontrada.", "info");
        return;
      }
      const full = await getRoster(previous.id);
      const map = { ...selected };
      for (const a of full.assignments ?? []) map[a.member_id] = a.role ?? "";
      setSelected(map);
      toast(`Equipe copiada de "${previous.title}".`);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao copiar equipe", "error");
    }
  }

  const filteredMembers = useMemo(() => {
    const q = memberSearch.trim().toLowerCase();
    return members.filter((m) => !q || m.full_name.toLowerCase().includes(q));
  }, [members, memberSearch]);

  const assignCounts = useMemo(() => {
    const list = manage?.assignments ?? [];
    const selectedIds = new Set(Object.keys(selected));
    const mine = list.filter((a) => selectedIds.has(a.member_id));
    const confirmed = mine.filter((a) => a.status === "confirmado").length;
    const refused = mine.filter((a) => a.status === "recusado").length;
    const total = selectedIds.size;
    return { total, confirmed, refused, pending: Math.max(0, total - confirmed - refused) };
  }, [manage, selected]);

  const visibleAssignments = useMemo(() => {
    return Object.entries(selected).filter(([mid]) => {
      if (assignFilter === "todos") return true;
      const a = manage?.assignments?.find((x) => x.member_id === mid);
      if (assignFilter === "convidado") return !a || a.status === "convidado";
      return a?.status === assignFilter;
    });
  }, [selected, manage, assignFilter]);

  const periodLabel = view === "day"
    ? format(anchor, "EEEE, dd 'de' MMMM", { locale: ptBR })
    : view === "week"
      ? `${format(startOfWeek(anchor, { weekStartsOn: 0 }), "dd/MM")} - ${format(addDays(startOfWeek(anchor, { weekStartsOn: 0 }), 6), "dd/MM")}`
      : format(anchor, "MMMM 'de' yyyy", { locale: ptBR });

  return (
    <div className="page">
      <PageHeader
        title="Escalas"
        description="Convocacao de voluntarios, confirmacao de presenca e conflito de agenda"
        actions={canWrite ? <Button data-tour="rosters-new" onClick={openCreate}><Plus className="h-4 w-4" /> Nova escala</Button> : undefined}
      />

      <Tabs
        tabs={[
          { key: "escalas", label: "Escalas", icon: <List className="h-4 w-4" /> },
          { key: "calendario", label: "Calendario", icon: <CalendarDays className="h-4 w-4" /> },
        ]}
        active={tab}
        onChange={(k) => setTab(k as "escalas" | "calendario")}
      />

      <div className="mb-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
        <KpiCard label="Escalas" value={String(totals.count)} icon={CalendarClock} tone="sky" />
        <KpiCard label="Escalados" value={String(totals.assignments)} icon={Users} tone="indigo" />
        <KpiCard label="Preenchimento" value={`${totals.fill.toFixed(0)}%`} icon={Check} tone="green" hint={`${totals.confirmed} confirmado(s)`} />
        <KpiCard label="Nao confirmados" value={String(totals.pending)} icon={UserPlus} tone="amber" hint={`${totals.noTeam} escala(s) sem equipe`} />
        <KpiCard label="Conflitos" value={String(periodConflicts.length)} icon={AlertTriangle} tone="red" hint="voluntarios em escalas sobrepostas" />
      </div>

      {periodConflicts.length > 0 && (
        <Card className="mb-4 border-amber-200 bg-amber-50/50 p-3 dark:border-amber-900 dark:bg-amber-950/20">
          <div className="mb-1 flex items-center gap-2 text-sm font-semibold text-amber-800 dark:text-amber-300">
            <AlertTriangle className="h-4 w-4" /> Conflitos no periodo ({periodConflicts.length})
          </div>
          <ul className="max-h-32 space-y-1 overflow-y-auto text-xs text-amber-800/90 dark:text-amber-200/80">
            {periodConflicts.map((c, i) => (
              <li key={`${c.member_id}-${c.roster_a_id}-${c.roster_b_id}-${i}`}>
                <b>{c.member_name}</b> esta em <b>{c.roster_a_title}</b> ({dateTimePt(c.roster_a_starts_at)}) e <b>{c.roster_b_title}</b> ({dateTimePt(c.roster_b_starts_at)}).
              </li>
            ))}
          </ul>
        </Card>
      )}

      <Card className="mb-4 flex flex-wrap items-end gap-3 p-3">
        {tab === "escalas" && (
          <>
            <div><label className="label">De</label><Input type="date" className="h-8 text-sm" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></div>
            <div><label className="label">Ate</label><Input type="date" className="h-8 text-sm" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></div>
          </>
        )}
        {tab === "calendario" && (
          <div className="flex items-center gap-1">
            <Button variant="ghost" className="h-8 px-2" onClick={() => setAnchor(addDays(anchor, view === "month" ? -30 : view === "week" ? -7 : -1))}><ChevronLeft className="h-4 w-4" /></Button>
            <Button variant="ghost" className="h-8 px-2 text-xs" onClick={() => setAnchor(new Date())}>Hoje</Button>
            <Button variant="ghost" className="h-8 px-2" onClick={() => setAnchor(addDays(anchor, view === "month" ? 30 : view === "week" ? 7 : 1))}><ChevronRight className="h-4 w-4" /></Button>
            <span className="ml-2 text-sm font-medium capitalize text-zinc-700 dark:text-zinc-200">{periodLabel}</span>
          </div>
        )}
        <div>
          <label className="label">Ministerio</label>
          <Select className="h-8 w-56 text-sm" value={filters.ministry} onChange={(e) => setFilters({ ...filters, ministry: e.target.value })}>
            <option value="">Todos</option>
            {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
          </Select>
        </div>
        <div>
          <label className="label">Situacao</label>
          <Select className="h-8 w-40 text-sm" value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}>
            <option value="">Todas</option>
            <option value="rascunho">Rascunho</option>
            <option value="publicada">Publicada</option>
            <option value="concluida">Concluida</option>
            <option value="cancelada">Cancelada</option>
          </Select>
        </div>
        {tab === "calendario" && (
          <div className="ml-auto flex flex-wrap items-center gap-1">
            {VIEWS.map((v) => (
              <Button key={v.key} variant={view === v.key ? "outline" : "ghost"} className="h-8 px-2 text-xs" onClick={() => setView(v.key)}>
                {v.label}
              </Button>
            ))}
          </div>
        )}
      </Card>

      {tab === "calendario" ? (
        rosters === null ? (
          <Card className="p-4"><SkeletonRows rows={8} /></Card>
        ) : (
          <>
            <RosterCalendar
              rosters={visibleRosters}
              events={calendarEvents}
              view={view}
              anchor={anchor}
              canWrite={canWrite}
              conflictRosterIds={conflictRosterIds}
              onOpen={openManage}
              onMove={moveRoster}
              onDelete={setDeleteTarget}
              onCreateAt={openCreateAt}
            />
            <p className="mt-2 text-xs text-zinc-400">
              Arraste uma escala para remarcar. Escalas publicadas movem tambem o evento gerado (com confirmacao).
              Os eventos da igreja aparecem em segundo plano.
            </p>
          </>
        )
      ) : (
        <Card className="overflow-hidden p-0">
          {rosters === null ? (
            <div className="p-4"><SkeletonRows rows={5} /></div>
          ) : visibleRosters.length === 0 ? (
            <EmptyState icon={<CalendarClock className="h-10 w-10" />} title="Nenhuma escala" description="Crie escalas para cultos, eventos e ministerios." />
          ) : (
            <Table>
              <THead><TRow><TH>Inicio</TH><TH>Titulo</TH><TH>Ministerio</TH><TH>Evento</TH><TH>Preenchimento</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
              <TBody>
                {visibleRosters.map((r) => (
                  <TRow key={r.id}>
                    <TD className="whitespace-nowrap text-sm">{dateTimePt(r.starts_at)}</TD>
                    <TD className="font-medium">
                      <span className="flex items-center gap-1">
                        {r.title}
                        {conflictRosterIds.has(r.id) && (
                          <span title="Conflito de agenda"><AlertTriangle className="h-3.5 w-3.5 text-amber-600" /></span>
                        )}
                      </span>
                    </TD>
                    <TD className="text-sm text-zinc-500">{r.ministry_name ?? "-"}</TD>
                    <TD className="text-sm text-zinc-500">
                      {r.event_name ?? r.event_kind_name ?? "-"}
                      {r.generated_event && <span className="ml-1 text-[10px] uppercase text-sky-600" title="Evento gerado pela escala">auto</span>}
                    </TD>
                    <TD>
                      <div className="flex items-center gap-2">
                        <div className="h-1.5 w-16 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                          <div className="h-1.5 rounded-full bg-emerald-500" style={{ width: `${r.assignment_count > 0 ? (r.confirmed_count / r.assignment_count) * 100 : 0}%` }} />
                        </div>
                        <span className="text-xs tabular-nums text-zinc-500">{r.confirmed_count}/{r.assignment_count}</span>
                      </div>
                    </TD>
                    <TD><Badge tone={STATUS_TONE[r.status]}>{r.status}</Badge></TD>
                    <TD>
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" data-tour="roster-manage" className="h-8 px-2" title="Gerenciar escalados" onClick={() => openManage(r)}><Users className="h-4 w-4" /></Button>
                        <Dropdown
                          trigger={<MoreHorizontal className="h-4 w-4" />}
                          items={[
                            { label: "Editar", icon: <Pencil className="h-4 w-4" />, onClick: () => openEdit(r), disabled: !canWrite },
                            { label: "Duplicar", icon: <Copy className="h-4 w-4" />, onClick: () => duplicateRoster(r), disabled: !canWrite },
                            { label: "Imprimir / PDF", icon: <FileText className="h-4 w-4" />, onClick: () => exportRoster(r, "pdf") },
                            { label: "Excel (XLSX)", icon: <FileSpreadsheet className="h-4 w-4" />, onClick: () => exportRoster(r, "xlsx") },
                            { label: "CSV", icon: <Download className="h-4 w-4" />, onClick: () => exportRoster(r, "csv") },
                            { label: "Excluir", icon: <Trash2 className="h-4 w-4" />, danger: true, onClick: () => setDeleteTarget(r), disabled: !canWrite },
                          ]}
                        />
                      </div>
                    </TD>
                  </TRow>
                ))}
              </TBody>
            </Table>
          )}
        </Card>
      )}

      {/* Drawer criar/editar */}
      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} size="xl" title={drawer.editing ? "Editar escala" : "Nova escala"}>
        <form onSubmit={saveRoster} className="space-y-3" data-tour="roster-form">
          <Field label="Titulo" required>
            <Input data-tour="roster-title" required className="h-8 text-sm" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
          </Field>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Ministerio">
              <Select className="h-8 text-sm" value={form.ministry_id} onChange={(e) => setForm({ ...form, ministry_id: e.target.value })}>
                <option value="">-</option>
                {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
              </Select>
            </Field>
            <Field label="Evento existente" hint="Vincule a escala a um evento ja cadastrado.">
              <Select className="h-8 text-sm" value={form.event_id} onChange={(e) => setForm({ ...form, event_id: e.target.value })}>
                <option value="">-</option>
                {events.map((ev) => <option key={ev.id} value={ev.id}>{dateTimePt(ev.starts_at)} - {ev.kind_name ?? "Evento"}</option>)}
              </Select>
            </Field>
            <Field label="Tipo de evento" hint="So a escala, ou a base do evento automatico.">
              <Select className="h-8 text-sm" value={form.event_kind_id} onChange={(e) => setForm({ ...form, event_kind_id: e.target.value })}>
                <option value="">-</option>
                {kinds.filter((k) => k.is_active).map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
              </Select>
            </Field>
            <label className="flex items-center gap-2 text-sm sm:col-span-2">
              <input
                type="checkbox"
                checked={form.create_event}
                disabled={!!form.event_id}
                onChange={(e) => setForm({ ...form, create_event: e.target.checked })}
                className="h-4 w-4 rounded border-zinc-300 text-sky-600"
              />
              <span>
                Gerar evento automatico
                <span className="block text-xs text-zinc-400">
                  Cria o evento na grade e sincroniza os escalados como convocados/responsaveis. Requer um tipo de evento e nenhum evento vinculado.
                </span>
              </span>
            </label>
            <Field label="Inicio" required>
              <Input data-tour="roster-start" required type="datetime-local" className="h-8 text-sm" value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} />
            </Field>
            <Field label="Termino">
              <Input type="datetime-local" className="h-8 text-sm" value={form.ends_at} onChange={(e) => setForm({ ...form, ends_at: e.target.value })} />
            </Field>
            <Field label="Local">
              <Input className="h-8 text-sm" value={form.location} onChange={(e) => setForm({ ...form, location: e.target.value })} />
            </Field>
            <Field label="Situacao">
              <Select className="h-8 text-sm" value={form.status} onChange={(e) => setForm({ ...form, status: e.target.value })}>
                <option value="rascunho">Rascunho</option>
                <option value="publicada">Publicada</option>
                <option value="concluida">Concluida</option>
                <option value="cancelada">Cancelada</option>
              </Select>
            </Field>
          </div>
          <Field label="Observacoes">
            <Textarea rows={2} className="text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </Field>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" data-tour="roster-save" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>

      {/* Drawer gerenciar escalados */}
      <Drawer open={manage !== null} onClose={() => setManage(null)} size="2xl" title={`Escalados - ${manage?.title ?? ""}`}>
        {conflicts.length > 0 && (
          <div className="mb-3 rounded border border-amber-300 bg-amber-50 p-2 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-300">
            <AlertTriangle className="mr-1 inline h-4 w-4" />
            Conflito de agenda: {conflicts.map((c) => `${c.member_name} (${c.other_roster_title})`).join("; ")}
          </div>
        )}

        <div className="mb-3 flex flex-wrap items-center gap-1">
          {([
            ["todos", `Todos (${assignCounts.total})`],
            ["confirmado", `Confirmados (${assignCounts.confirmed})`],
            ["convidado", `Pendentes (${assignCounts.pending})`],
            ["recusado", `Recusados (${assignCounts.refused})`],
          ] as const).map(([key, label]) => (
            <Button key={key} variant={assignFilter === key ? "outline" : "ghost"} className="h-7 px-2 text-xs" onClick={() => setAssignFilter(key)}>
              {label}
            </Button>
          ))}
          {manage?.ministry_id && (
            <Button variant="outline" className="ml-auto h-7 px-2 text-xs" onClick={copyLastTeam}>
              <Copy className="mr-1 h-3.5 w-3.5" /> Copiar ultima equipe
            </Button>
          )}
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded border border-zinc-200 dark:border-zinc-700">
            <div className="border-b border-zinc-100 p-2 text-sm font-semibold dark:border-zinc-800">
              Escalados ({visibleAssignments.length})
              <span className="ml-1 block text-xs font-normal text-zinc-400">
                Pessoas alem do ministerio{manage?.ministry_name ? ` - ${manage.ministry_name}` : ""}
              </span>
            </div>
            <ul className="max-h-96 space-y-1 overflow-y-auto p-2">
              {visibleAssignments.map(([mid, role]) => {
                const asg = manage?.assignments?.find((a) => a.member_id === mid);
                return (
                  <li key={mid} className="rounded border border-zinc-100 p-2 text-sm dark:border-zinc-800">
                    <div className="flex items-center gap-2">
                      <span className="min-w-0 flex-1 truncate font-medium">{memberName(mid)}</span>
                      {asg && <Badge tone={ASSIGN_TONE[asg.status]}>{asg.status}</Badge>}
                      <button type="button" title="Remover" onClick={() => setSelected((s) => { const n = { ...s }; delete n[mid]; return n; })} className="text-zinc-400 hover:text-red-600"><X className="h-4 w-4" /></button>
                    </div>
                    <div className="mt-1 flex items-center gap-2">
                      <Input className="h-7 text-xs" placeholder="Funcao (vocal, diaconia...)" value={role}
                        onChange={(e) => setSelected((s) => ({ ...s, [mid]: e.target.value }))} />
                      {asg && asg.status !== "confirmado" && <Button variant="ghost" className="h-7 px-2 text-emerald-600" title="Confirmar" onClick={() => respond(asg.id, "confirmado")}><Check className="h-4 w-4" /></Button>}
                      {asg && asg.status !== "recusado" && <Button variant="ghost" className="h-7 px-2 text-red-600" title="Recusar" onClick={() => respond(asg.id, "recusado")}><X className="h-4 w-4" /></Button>}
                    </div>
                  </li>
                );
              })}
              {visibleAssignments.length === 0 && <li className="p-2 text-xs text-zinc-400">Ninguem neste filtro.</li>}
            </ul>
            <div className="flex justify-end gap-2 border-t border-zinc-100 p-2 dark:border-zinc-800">
              <Button className="h-8 text-sm" onClick={saveAssignments} disabled={!canWrite}>Salvar escalados</Button>
            </div>
          </div>

          <div className="rounded border border-zinc-200 dark:border-zinc-700">
            <div className="border-b border-zinc-100 p-2 text-sm font-semibold dark:border-zinc-800">
              Adicionar pessoas
            </div>
            {manage?.ministry_id && (
              <div className="border-b border-zinc-100 p-2 dark:border-zinc-800">
                <p className="mb-1 text-xs font-medium uppercase text-zinc-400">Sugeridos pelo ministerio</p>
                <ul className="max-h-32 space-y-0.5 overflow-y-auto">
                  {suggestions.filter((s) => !selected[s.member_id]).map((s) => (
                    <li key={s.member_id} className="flex items-center justify-between gap-2 rounded px-1 py-1 text-sm">
                      <span className="truncate">
                        {s.member_name}
                        {s.busy && <span className="ml-1 text-xs text-amber-600">(ocupado)</span>}
                        {s.frequency && <span className="ml-1 text-xs text-zinc-400">- {s.frequency.replace("_", " ")}</span>}
                      </span>
                      <Button variant="ghost" className="h-7 px-2" title="Adicionar" onClick={() => setSelected((prev) => ({ ...prev, [s.member_id]: prev[s.member_id] ?? "" }))}><UserPlus className="h-4 w-4" /></Button>
                    </li>
                  ))}
                  {suggestions.length === 0 && <li className="px-1 py-2 text-xs text-zinc-400">Sem sugestoes. Vincule voluntarios ao ministerio.</li>}
                </ul>
              </div>
            )}
            <div className="p-2">
              <Input className="mb-2 h-8 text-sm" placeholder="Buscar membro..." value={memberSearch} onChange={(e) => setMemberSearch(e.target.value)} />
              <ul className="max-h-64 space-y-0.5 overflow-y-auto">
                {filteredMembers.filter((m) => !selected[m.id]).slice(0, 50).map((m) => (
                  <li key={m.id}>
                    <button type="button" className="flex w-full items-center justify-between gap-2 rounded px-2 py-1.5 text-left text-sm hover:bg-zinc-50 dark:hover:bg-zinc-800"
                      onClick={() => setSelected((prev) => ({ ...prev, [m.id]: prev[m.id] ?? "" }))}>
                      <span className="truncate">{m.full_name}</span>
                      <UserPlus className="h-4 w-4 text-zinc-400" />
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
        {manage && (
          <div className="mt-3 flex justify-end">
            <ExportMenu
              path={`/api/v1/rosters/${manage.id}/export`}
              filenameBase={`escala-${manage.id}`}
              trigger={<Button variant="outline" className="h-8 text-sm"><CalendarClock className="mr-1 h-3.5 w-3.5" /> Exportar escala</Button>}
            />
          </div>
        )}
      </Drawer>

      <Modal open={deleteTarget !== null} onClose={() => setDeleteTarget(null)} title="Excluir escala">
        {deleteTarget && (
          <div className="space-y-4 text-sm">
            {deleteTarget.generated_event ? (
              <>
                <p>
                  A escala <strong>{deleteTarget.title}</strong> gerou um evento na grade de eventos.
                  Deseja excluir a escala da grade tambem?
                </p>
                <div className="flex flex-wrap justify-end gap-2">
                  <Button variant="ghost" onClick={() => setDeleteTarget(null)}>Cancelar</Button>
                  <Button variant="outline" onClick={() => removeRoster(deleteTarget, false)}>Manter evento</Button>
                  <Button
                    variant="outline"
                    className="border-red-300 text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
                    onClick={() => removeRoster(deleteTarget, true)}
                  >
                    <Trash2 className="h-4 w-4" /> Excluir escala e evento
                  </Button>
                </div>
              </>
            ) : (
              <>
                <p>Excluir a escala <strong>{deleteTarget.title}</strong>?</p>
                <div className="flex justify-end gap-2">
                  <Button variant="ghost" onClick={() => setDeleteTarget(null)}>Cancelar</Button>
                  <Button
                    variant="outline"
                    className="border-red-300 text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
                    onClick={() => removeRoster(deleteTarget, false)}
                  >
                    <Trash2 className="h-4 w-4" /> Excluir
                  </Button>
                </div>
              </>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}

/** Exporta uma escala via URL (o menu visual usa o mesmo endpoint). */
async function exportRoster(r: Roster, format: "csv" | "xlsx" | "pdf") {
  const path = `/api/v1/rosters/${r.id}/export?format=${format}`;
  if (format === "pdf") await openPrint(path);
  else await downloadExport(path, `escala-${r.id}.${format}`);
}
