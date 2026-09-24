"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  AlertTriangle, CalendarClock, Check, Pencil, Plus, Trash2, UserPlus, Users, X,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listRosters, createRoster, updateRoster, deleteRoster, getRoster,
  setRosterAssignments, respondRosterAssignment, getRosterConflicts, getRosterSuggestions,
  listMinistries, listMembers, listEvents, listEventKinds,
  type Roster, type RosterConflict, type RosterSuggestion, type Ministry, type Member, type ChurchEvent, type EventKind,
} from "@/lib/api";
import { dateTimePt } from "@/lib/format";

const STATUS_TONE: Record<string, "zinc" | "sky" | "green" | "red"> = {
  rascunho: "zinc", publicada: "sky", concluida: "green", cancelada: "red",
};
const ASSIGN_TONE: Record<string, "zinc" | "green" | "red"> = {
  convidado: "zinc", confirmado: "green", recusado: "red",
};

const localNow = () => {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

const EMPTY = {
  title: "", ministry_id: "", event_id: "", event_kind_id: "", create_event: false,
  starts_at: localNow(), ends_at: "", location: "", notes: "", status: "rascunho",
};

export default function RostersPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("ministries.write");

  const [rosters, setRosters] = useState<Roster[] | null>(null);
  const [ministries, setMinistries] = useState<Ministry[]>([]);
  const [members, setMembers] = useState<Member[]>([]);
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [filters, setFilters] = useState({ from: "", to: "", ministry: "" });
  const [saving, setSaving] = useState(false);

  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Roster }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });

  const [manage, setManage] = useState<Roster | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Roster | null>(null);
  const [selected, setSelected] = useState<Record<string, string>>({});
  const [conflicts, setConflicts] = useState<RosterConflict[]>([]);
  const [suggestions, setSuggestions] = useState<RosterSuggestion[]>([]);
  const [memberSearch, setMemberSearch] = useState("");

  const load = useCallback(async () => {
    try {
      setRosters((await listRosters(filters)).rosters);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar escalas", "error");
      setRosters([]);
    }
  }, [filters, toast]);

  useEffect(() => { load(); }, [load]);
  useEffect(() => {
    listMinistries().then((r) => setMinistries(r.ministries)).catch(() => setMinistries([]));
    listMembers().then((r) => setMembers(r.members)).catch(() => setMembers([]));
    listEvents().then((r) => setEvents(r.events)).catch(() => setEvents([]));
    listEventKinds().then((r) => setKinds(r.kinds)).catch(() => setKinds([]));
  }, []);

  const memberName = (id: string) => members.find((m) => m.id === id)?.full_name ?? id;

  // ---- CRUD ----
  function openCreate() {
    setForm({ ...EMPTY });
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

  const filteredMembers = useMemo(() => {
    const q = memberSearch.trim().toLowerCase();
    return members.filter((m) => !q || m.full_name.toLowerCase().includes(q));
  }, [members, memberSearch]);

  return (
    <div className="page">
      <PageHeader
        title="Escalas"
        description="Convocacao de voluntarios, confirmacao de presenca e conflito de agenda"
        actions={canWrite ? <Button onClick={openCreate}><Plus className="h-4 w-4" /> Nova escala</Button> : undefined}
      />

      <Card className="mb-4 flex flex-wrap items-end gap-3">
        <div><label className="label">De</label><Input type="date" className="h-8 text-sm" value={filters.from} onChange={(e) => setFilters({ ...filters, from: e.target.value })} /></div>
        <div><label className="label">Ate</label><Input type="date" className="h-8 text-sm" value={filters.to} onChange={(e) => setFilters({ ...filters, to: e.target.value })} /></div>
        <div>
          <label className="label">Ministerio</label>
          <Select className="h-8 w-56 text-sm" value={filters.ministry} onChange={(e) => setFilters({ ...filters, ministry: e.target.value })}>
            <option value="">Todos</option>
            {ministries.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
          </Select>
        </div>
      </Card>

      <Card className="overflow-hidden p-0">
        {rosters === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : rosters.length === 0 ? (
          <EmptyState icon={<CalendarClock className="h-10 w-10" />} title="Nenhuma escala" description="Crie escalas para cultos, eventos e ministerios." />
        ) : (
          <Table>
            <THead><TRow><TH>Inicio</TH><TH>Titulo</TH><TH>Ministerio</TH><TH>Evento</TH><TH>Confirmados</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
            <TBody>
              {rosters.map((r) => (
                <TRow key={r.id}>
                  <TD className="whitespace-nowrap text-sm">{dateTimePt(r.starts_at)}</TD>
                  <TD className="font-medium">{r.title}</TD>
                  <TD className="text-sm text-zinc-500">{r.ministry_name ?? "-"}</TD>
                  <TD className="text-sm text-zinc-500">
                    {r.event_name ?? r.event_kind_name ?? "-"}
                    {r.generated_event && <span className="ml-1 text-[10px] uppercase text-sky-600" title="Evento gerado pela escala">auto</span>}
                  </TD>
                  <TD className="text-sm tabular-nums">{r.confirmed_count}/{r.assignment_count}</TD>
                  <TD><Badge tone={STATUS_TONE[r.status]}>{r.status}</Badge></TD>
                  <TD>
                    <div className="flex justify-end gap-1">
                      <Button variant="ghost" className="h-8 px-2" title="Gerenciar escalados" onClick={() => openManage(r)}><Users className="h-4 w-4" /></Button>
                      {canWrite && <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(r)}><Pencil className="h-4 w-4" /></Button>}
                      {canWrite && <Button variant="ghost" className="h-8 px-2" title="Excluir" onClick={() => setDeleteTarget(r)}><Trash2 className="h-4 w-4" /></Button>}
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      {/* Drawer criar/editar */}
      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} size="xl" title={drawer.editing ? "Editar escala" : "Nova escala"}>
        <form onSubmit={saveRoster} className="space-y-3">
          <Field label="Titulo" required>
            <Input required className="h-8 text-sm" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
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
              <Input required type="datetime-local" className="h-8 text-sm" value={form.starts_at} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} />
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
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
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
        <div className="grid gap-4 lg:grid-cols-2">
          <div className="rounded border border-zinc-200 dark:border-zinc-700">
            <div className="border-b border-zinc-100 p-2 text-sm font-semibold dark:border-zinc-800">
              Escalados ({Object.keys(selected).length})
              <span className="ml-1 block text-xs font-normal text-zinc-400">
                Pessoas alem do ministerio{manage?.ministry_name ? ` - ${manage.ministry_name}` : ""}
              </span>
            </div>
            <ul className="max-h-96 space-y-1 overflow-y-auto p-2">
              {Object.entries(selected).map(([mid, role]) => {
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
              {Object.keys(selected).length === 0 && <li className="p-2 text-xs text-zinc-400">Ninguem escalado ainda.</li>}
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
