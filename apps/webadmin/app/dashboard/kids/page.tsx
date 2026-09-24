"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Baby, Plus, Search, Pencil, Trash2, BookOpen, Users, CalendarDays,
  ClipboardCheck, TrendingUp, Printer, UserPlus, X, CheckCircle, XCircle, MinusCircle,
} from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Tabs } from "@/components/ui/tabs";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { StatCard } from "@/components/ui/stat-card";
import { Modal, Drawer } from "@/components/ui/modal";
import { Badge } from "@/components/ui/badge";
import { useToast } from "@/components/ui/toast";
import { useBranches } from "@/lib/swr-hooks";
import {
  listMembers, listKidTracks, createKidTrack, updateKidTrack, deleteKidTrack,
  listKidLessons, createKidLesson, updateKidLesson, deleteKidLesson,
  listKidClasses, createKidClass, updateKidClass, deleteKidClass,
  listKidEnrollments, createKidEnrollment, updateKidEnrollment, deleteKidEnrollment,
  listKidGuardians, addKidGuardian, deleteKidGuardian,
  listKidSessions, createKidSession, updateKidSession, deleteKidSession,
  getKidRoster, kidCheckin, kidCheckout, kidAbsence, getKidEvolution,
  type Member, type KidTrack, type KidLesson, type KidClass, type KidEnrollment,
  type KidGuardian, type KidSession, type KidRosterEntry, type KidEvolution,
} from "@/lib/api";

const TABS = [
  { key: "classes", label: "Turmas", icon: <Users className="h-4 w-4" /> },
  { key: "content", label: "Conteúdo", icon: <BookOpen className="h-4 w-4" /> },
  { key: "participants", label: "Participantes", icon: <Baby className="h-4 w-4" /> },
  { key: "sessions", label: "Encontros", icon: <CalendarDays className="h-4 w-4" /> },
  { key: "checkin", label: "Check-in", icon: <ClipboardCheck className="h-4 w-4" /> },
  { key: "evolution", label: "Evolução", icon: <TrendingUp className="h-4 w-4" /> },
];

function ageFrom(birth?: string): number | null {
  if (!birth) return null;
  const d = new Date(birth);
  if (isNaN(d.getTime())) return null;
  const now = new Date();
  let a = now.getFullYear() - d.getFullYear();
  const m = now.getMonth() - d.getMonth();
  if (m < 0 || (m === 0 && now.getDate() < d.getDate())) a--;
  return a;
}

function dtLocal(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

export default function KidsPage() {
  const { toast } = useToast();
  const [tab, setTab] = useState("classes");
  const [members, setMembers] = useState<Member[]>([]);
  const [tracks, setTracks] = useState<KidTrack[]>([]);
  const [classes, setClasses] = useState<KidClass[]>([]);
  const [classId, setClassId] = useState("");
  const [enrollments, setEnrollments] = useState<KidEnrollment[]>([]);
  const [sessions, setSessions] = useState<KidSession[]>([]);
  const [sessionId, setSessionId] = useState("");
  const [roster, setRoster] = useState<KidRosterEntry[]>([]);
  const [trackId, setTrackId] = useState("");
  const [lessons, setLessons] = useState<KidLesson[]>([]);
  const [evolution, setEvolution] = useState<KidEvolution | null>(null);
  const [loading, setLoading] = useState(true);

  const reloadClasses = useCallback(async () => {
    const [tr, cl] = await Promise.all([listKidTracks(), listKidClasses()]);
    setTracks(tr.tracks);
    setClasses(cl.classes);
    setClassId((cur) => cur || cl.classes[0]?.id || "");
    setTrackId((cur) => cur || tr.tracks[0]?.id || "");
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const m = await listMembers();
        setMembers(m.members);
        await reloadClasses();
      } catch (e) {
        toast(e instanceof Error ? e.message : "Erro ao carregar", "error");
      } finally {
        setLoading(false);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (tab === "participants" && classId) loadEnrollments(classId);
    if (tab === "sessions") loadSessions();
    if (tab === "evolution" && classId) loadEvolution(classId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, classId]);

  useEffect(() => {
    if (trackId) loadLessons(trackId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trackId]);

  async function loadEnrollments(id: string) {
    try { setEnrollments((await listKidEnrollments(id)).enrollments); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }
  async function loadSessions() {
    try { setSessions((await listKidSessions(classId ? { class_id: classId } : {})).sessions); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }
  async function loadLessons(id: string) {
    try { setLessons((await listKidLessons(id)).lessons); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }
  async function loadEvolution(id: string) {
    try { setEvolution(await getKidEvolution(id)); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }
  async function loadRoster(id: string) {
    setSessionId(id);
    try { setRoster((await getKidRoster(id)).roster); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }

  if (loading) return <div className="page"><SkeletonRows /></div>;

  return (
    <div className="page space-y-6">
      <PageHeader title="Ministério Infantil (Kids)" description="Trilha, turmas, participantes, check-in e evolução" />
      <Tabs tabs={TABS} active={tab} onChange={setTab} />

      {tab === "classes" && (
        <ClassesTab classes={classes} tracks={tracks} members={members}
          reload={reloadClasses} toast={toast} />
      )}
      {tab === "content" && (
        <ContentTab tracks={tracks} trackId={trackId} setTrackId={setTrackId} lessons={lessons}
          reloadTracks={reloadClasses} reloadLessons={loadLessons} toast={toast} />
      )}
      {tab === "participants" && (
        <ParticipantsTab classes={classes} classId={classId} setClassId={setClassId}
          enrollments={enrollments} members={members} reload={() => classId && loadEnrollments(classId)} toast={toast} />
      )}
      {tab === "sessions" && (
        <SessionsTab classes={classes} classId={classId} setClassId={setClassId} sessions={sessions}
          reload={loadSessions} openCheckin={(id) => { setTab("checkin"); loadRoster(id); }} toast={toast} />
      )}
      {tab === "checkin" && (
        <CheckinTab classes={classes} sessions={sessions} sessionId={sessionId} roster={roster}
          onSelectSession={loadRoster} reloadSessions={loadSessions} toast={toast} />
      )}
      {tab === "evolution" && (
        <EvolutionTab classes={classes} classId={classId} setClassId={setClassId}
          evolution={evolution} reload={() => classId && loadEvolution(classId)} toast={toast} />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Turmas
// ---------------------------------------------------------------------------

function ClassesTab({ classes, tracks, members, reload, toast }: {
  classes: KidClass[]; tracks: KidTrack[]; members: Member[];
  reload: () => Promise<void>; toast: (m: string, t?: "error") => void;
}) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<Record<string, unknown>>({ is_active: true });
  const { data: branchData } = useBranches();
  const branches = branchData?.branches ?? [];

  function openNew() {
    setForm({ name: "", age_min: "", age_max: "", track_id: "", room: "", leader_member_id: "", is_active: true, branch_id: branches[0]?.id ?? "" });
    setOpen(true);
  }
  function openEdit(c: KidClass) {
    setForm({
      id: c.id,
      name: c.name, age_min: c.age_min ?? "", age_max: c.age_max ?? "",
      track_id: c.track_id, room: c.room, leader_member_id: c.leader_member_id, is_active: c.is_active,
    });
    setOpen(true);
  }
  async function save(e: React.FormEvent) {
    e.preventDefault();
    try {
      const payload = { ...form };
      if (!payload.id) await createKidClass(payload);
      else await updateKidClass(String(payload.id), payload);
      toast("Turma salva.");
      setOpen(false);
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function remove(c: KidClass) {
    if (!confirm(`Excluir a turma "${c.name}"? Matrículas e encontros serão removidos.`)) return;
    try { await deleteKidClass(c.id); toast("Turma excluída."); await reload(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  return (
    <>
      <div className="flex justify-end">
        <Button onClick={openNew}><Plus className="h-4 w-4" /> Nova Turma</Button>
      </div>
      {classes.length === 0 ? (
        <EmptyState icon={<Users className="h-10 w-10" />} title="Nenhuma turma" description="Crie a primeira turma do ministério infantil." />
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead><TRow><TH>Turma</TH><TH>Faixa</TH><TH>Trilha</TH><TH>Sala</TH><TH>Líder</TH><TH>Ativos</TH><TH>Status</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {classes.map((c) => (
                <TRow key={c.id}>
                  <TD className="font-medium">{c.name}</TD>
                  <TD className="text-zinc-500">{ageLabel(c.age_min, c.age_max)}</TD>
                  <TD className="text-zinc-500">{c.track_name || "—"}</TD>
                  <TD className="text-zinc-500">{c.room || "—"}</TD>
                  <TD className="text-zinc-500">{c.leader_name || "—"}</TD>
                  <TD>{c.enrollment_count}</TD>
                  <TD>{c.is_active ? <Badge tone="green">Ativa</Badge> : <Badge tone="zinc">Inativa</Badge>}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="ghost" onClick={() => openEdit(c)}><Pencil className="h-3 w-3" /> Editar</Button>
                      <Button size="sm" variant="ghost" onClick={() => remove(c)}><Trash2 className="h-3 w-3" /></Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={open} onClose={() => setOpen(false)} title={(form as any).id ? "Editar Turma" : "Nova Turma"}>
        <form onSubmit={save} className="space-y-3">
          <Field label="Nome *"><Input required value={String(form.name ?? "")} onChange={(e) => setForm({ ...form, name: e.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Idade mín."><Input type="number" min={0} value={String(form.age_min ?? "")} onChange={(e) => setForm({ ...form, age_min: e.target.value })} /></Field>
            <Field label="Idade máx."><Input type="number" min={0} value={String(form.age_max ?? "")} onChange={(e) => setForm({ ...form, age_max: e.target.value })} /></Field>
          </div>
          <Field label="Trilha (currículo)">
            <select className="input w-full" value={String(form.track_id ?? "")} onChange={(e) => setForm({ ...form, track_id: e.target.value })}>
              <option value="">Sem trilha</option>
              {tracks.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </Field>
          <Field label="Sala / local"><Input value={String(form.room ?? "")} onChange={(e) => setForm({ ...form, room: e.target.value })} /></Field>
          {!form.id && (
            <Field label="Filial *">
              <select className="input w-full" value={String(form.branch_id ?? "")} onChange={(e) => setForm({ ...form, branch_id: e.target.value })}>
                {branches.map((b) => <option key={b.id} value={b.id}>{b.name}</option>)}
              </select>
            </Field>
          )}
          <Field label="Líder">
            <select className="input w-full" value={String(form.leader_member_id ?? "")} onChange={(e) => setForm({ ...form, leader_member_id: e.target.value })}>
              <option value="">—</option>
              {members.map((m) => <option key={m.id} value={m.id}>{m.full_name}</option>)}
            </select>
          </Field>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" className="h-4 w-4" checked={!!form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} /> Turma ativa
          </label>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit">Salvar</Button>
          </div>
        </form>
      </Drawer>
    </>
  );
}

function ageLabel(min?: number, max?: number): string {
  if (min == null && max == null) return "—";
  if (min != null && max != null) return `${min}–${max} anos`;
  if (min != null) return `${min}+ anos`;
  return `até ${max} anos`;
}

// ---------------------------------------------------------------------------
// Conteúdo
// ---------------------------------------------------------------------------

function ContentTab({ tracks, trackId, setTrackId, lessons, reloadTracks, reloadLessons, toast }: {
  tracks: KidTrack[]; trackId: string; setTrackId: (id: string) => void; lessons: KidLesson[];
  reloadTracks: () => Promise<void>; reloadLessons: (id: string) => Promise<void>;
  toast: (m: string, t?: "error") => void;
}) {
  const [trackOpen, setTrackOpen] = useState(false);
  const [trackForm, setTrackForm] = useState<Record<string, unknown>>({});
  const [lessonOpen, setLessonOpen] = useState(false);
  const [lessonForm, setLessonForm] = useState<Record<string, unknown>>({});

  async function saveTrack(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (trackForm.id) await updateKidTrack(String(trackForm.id), trackForm);
      else await createKidTrack(trackForm);
      toast("Trilha salva.");
      setTrackOpen(false);
      await reloadTracks();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeTrack(t: KidTrack) {
    if (!confirm(`Excluir a trilha "${t.name}" e suas lições?`)) return;
    try { await deleteKidTrack(t.id); toast("Trilha excluída."); setTrackId(""); await reloadTracks(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function saveLesson(e: React.FormEvent) {
    e.preventDefault();
    if (!trackId) { toast("Selecione uma trilha.", "error"); return; }
    try {
      if (lessonForm.id) await updateKidLesson(String(lessonForm.id), lessonForm);
      else await createKidLesson(trackId, lessonForm);
      toast("Lição salva.");
      setLessonOpen(false);
      await reloadLessons(trackId);
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function removeLesson(l: KidLesson) {
    if (!confirm(`Excluir a lição "${l.title}"?`)) return;
    try { await deleteKidLesson(l.id); await reloadLessons(trackId); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[280px_1fr]">
      <Card className="p-3">
        <div className="mb-2 flex items-center justify-between">
          <p className="text-sm font-semibold">Trilhas</p>
          <Button size="sm" variant="ghost" onClick={() => { setTrackForm({ name: "", description: "", age_min: "", age_max: "", is_active: true }); setTrackOpen(true); }}><Plus className="h-4 w-4" /></Button>
        </div>
        <div className="space-y-1">
          {tracks.map((t) => (
            <button key={t.id} onClick={() => setTrackId(t.id)}
              className={`flex w-full items-center justify-between rounded-md px-2 py-1.5 text-left text-sm ${trackId === t.id ? "bg-sky-50 text-sky-700 dark:bg-sky-500/10" : "hover:bg-zinc-100 dark:hover:bg-zinc-900"}`}>
              <span className="truncate">{t.name}</span>
              <span className="ml-2 flex shrink-0 items-center gap-1 text-[11px] text-zinc-400">
                {t.lesson_count}
                <Pencil className="h-3 w-3" onClick={(e) => { e.stopPropagation(); setTrackForm({ id: t.id, name: t.name, description: t.description, age_min: t.age_min ?? "", age_max: t.age_max ?? "", is_active: t.is_active }); setTrackOpen(true); }} />
                <Trash2 className="h-3 w-3" onClick={(e) => { e.stopPropagation(); removeTrack(t); }} />
              </span>
            </button>
          ))}
          {tracks.length === 0 && <p className="px-2 py-3 text-xs text-zinc-500">Nenhuma trilha.</p>}
        </div>
      </Card>

      <Card className="p-3">
        <div className="mb-2 flex items-center justify-between">
          <p className="text-sm font-semibold">Lições {trackId ? "da trilha" : ""}</p>
          <Button size="sm" onClick={() => { setLessonForm({ position: "", title: "", objective: "", verse: "", content: "", materials: "" }); setLessonOpen(true); }} disabled={!trackId}>
            <Plus className="h-4 w-4" /> Lição
          </Button>
        </div>
        {lessons.length === 0 ? (
          <p className="px-2 py-6 text-center text-sm text-zinc-500">Nenhuma lição. Crie a primeira.</p>
        ) : (
          <Table>
            <THead><TRow><TH className="w-10">#</TH><TH>Título</TH><TH>Objetivo</TH><TH>Versículo</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {lessons.map((l) => (
                <TRow key={l.id}>
                  <TD className="text-zinc-400">{l.position}</TD>
                  <TD className="font-medium">{l.title}</TD>
                  <TD className="text-zinc-500 line-clamp-1 max-w-xs">{l.objective || "—"}</TD>
                  <TD className="text-zinc-500">{l.verse || "—"}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="ghost" onClick={() => { setLessonForm({ id: l.id, position: l.position, title: l.title, objective: l.objective, verse: l.verse, content: l.content, materials: l.materials }); setLessonOpen(true); }}><Pencil className="h-3 w-3" /></Button>
                      <Button size="sm" variant="ghost" onClick={() => removeLesson(l)}><Trash2 className="h-3 w-3" /></Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={trackOpen} onClose={() => setTrackOpen(false)} title={trackForm.id ? "Editar Trilha" : "Nova Trilha"}>
        <form onSubmit={saveTrack} className="space-y-3">
          <Field label="Nome *"><Input required value={String(trackForm.name ?? "")} onChange={(e) => setTrackForm({ ...trackForm, name: e.target.value })} /></Field>
          <Field label="Descrição"><Textarea rows={3} value={String(trackForm.description ?? "")} onChange={(e) => setTrackForm({ ...trackForm, description: e.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Idade mín."><Input type="number" min={0} value={String(trackForm.age_min ?? "")} onChange={(e) => setTrackForm({ ...trackForm, age_min: e.target.value })} /></Field>
            <Field label="Idade máx."><Input type="number" min={0} value={String(trackForm.age_max ?? "")} onChange={(e) => setTrackForm({ ...trackForm, age_max: e.target.value })} /></Field>
          </div>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" className="h-4 w-4" checked={!!trackForm.is_active} onChange={(e) => setTrackForm({ ...trackForm, is_active: e.target.checked })} /> Ativa</label>
          <div className="flex justify-end gap-2 pt-2"><Button variant="ghost" type="button" onClick={() => setTrackOpen(false)}>Cancelar</Button><Button type="submit">Salvar</Button></div>
        </form>
      </Drawer>

      <Drawer open={lessonOpen} onClose={() => setLessonOpen(false)} title={lessonForm.id ? "Editar Lição" : "Nova Lição"} size="xl">
        <form onSubmit={saveLesson} className="space-y-3">
          <div className="grid grid-cols-[100px_1fr] gap-3">
            <Field label="Ordem"><Input type="number" min={1} value={String(lessonForm.position ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, position: e.target.value })} /></Field>
            <Field label="Título *"><Input required value={String(lessonForm.title ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, title: e.target.value })} /></Field>
          </div>
          <Field label="Objetivo"><Input value={String(lessonForm.objective ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, objective: e.target.value })} /></Field>
          <Field label="Versículo"><Input value={String(lessonForm.verse ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, verse: e.target.value })} /></Field>
          <Field label="Conteúdo / roteiro"><Textarea rows={5} value={String(lessonForm.content ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, content: e.target.value })} /></Field>
          <Field label="Materiais"><Textarea rows={2} value={String(lessonForm.materials ?? "")} onChange={(e) => setLessonForm({ ...lessonForm, materials: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2"><Button variant="ghost" type="button" onClick={() => setLessonOpen(false)}>Cancelar</Button><Button type="submit">Salvar</Button></div>
        </form>
      </Drawer>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Participantes
// ---------------------------------------------------------------------------

function ParticipantsTab({ classes, classId, setClassId, enrollments, members, reload, toast }: {
  classes: KidClass[]; classId: string; setClassId: (id: string) => void; enrollments: KidEnrollment[];
  members: Member[]; reload: () => void; toast: (m: string, t?: "error") => void;
}) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<Record<string, unknown>>({});
  const [guardianFor, setGuardianFor] = useState<KidEnrollment | null>(null);
  const [guardians, setGuardians] = useState<KidGuardian[]>([]);
  const [gForm, setGForm] = useState<Record<string, unknown>>({ relationship: "", is_primary: false });

  async function save(e: React.FormEvent) {
    e.preventDefault();
    try {
      if (form.id) await updateKidEnrollment(String(form.id), form);
      else await createKidEnrollment(classId, form);
      toast("Matrícula salva.");
      setOpen(false);
      reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function remove(en: KidEnrollment) {
    if (!confirm(`Remover ${en.member_name} da turma?`)) return;
    try { await deleteKidEnrollment(en.id); toast("Matrícula removida."); reload(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function openGuardians(en: KidEnrollment) {
    setGuardianFor(en);
    setGForm({ relationship: "", is_primary: false, member_id: "" });
    try { setGuardians((await listKidGuardians(en.id)).guardians); }
    catch (e) { toast(e instanceof Error ? e.message : "Erro", "error"); }
  }
  async function addG(e: React.FormEvent) {
    e.preventDefault();
    if (!guardianFor) return;
    if (!gForm.member_id) { toast("Selecione o responsável.", "error"); return; }
    try {
      await addKidGuardian(guardianFor.id, gForm);
      setGuardians((await listKidGuardians(guardianFor.id)).guardians);
      setGForm({ relationship: "", is_primary: false, member_id: "" });
      reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function delG(id: string) {
    if (!guardianFor) return;
    try { await deleteKidGuardian(id); setGuardians((await listKidGuardians(guardianFor.id)).guardians); reload(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  return (
    <>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <Field label="Turma">
          <select className="input w-64" value={classId} onChange={(e) => setClassId(e.target.value)}>
            {classes.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </Field>
        <Button onClick={() => { setForm({ member_id: "", status: "active", start_date: "", dietary_restrictions: "", notes: "" }); setOpen(true); }} disabled={!classId}>
          <UserPlus className="h-4 w-4" /> Matricular
        </Button>
      </div>

      {enrollments.length === 0 ? (
        <EmptyState icon={<Baby className="h-10 w-10" />} title="Nenhuma criança" description="Matricule crianças nesta turma." />
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead><TRow><TH>Criança</TH><TH>Idade</TH><TH>Status</TH><TH>Responsáveis</TH><TH>Restrição alimentar</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {enrollments.map((en) => (
                <TRow key={en.id}>
                  <TD className="font-medium">{en.member_name}</TD>
                  <TD className="text-zinc-500">{ageFrom(en.birth_date) ?? "—"}</TD>
                  <TD><Badge tone={en.status === "active" ? "green" : en.status === "paused" ? "amber" : "zinc"}>{en.status === "active" ? "Ativa" : en.status === "paused" ? "Pausada" : "Encerrada"}</Badge></TD>
                  <TD>{en.guardians_count}</TD>
                  <TD className="text-zinc-500">{en.dietary_restrictions || "—"}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="outline" onClick={() => openGuardians(en)}><Users className="h-3 w-3" /> Responsáveis</Button>
                      <Button size="sm" variant="ghost" onClick={() => { setForm({ id: en.id, status: en.status, start_date: en.start_date, dietary_restrictions: en.dietary_restrictions, notes: en.notes }); setOpen(true); }}><Pencil className="h-3 w-3" /></Button>
                      <Button size="sm" variant="ghost" onClick={() => remove(en)}><Trash2 className="h-3 w-3" /></Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={open} onClose={() => setOpen(false)} title={form.id ? "Editar Matrícula" : "Matricular Criança"}>
        <form onSubmit={save} className="space-y-3">
          {!form.id && (
            <Field label="Criança (membro) *">
              <select required className="input w-full" value={String(form.member_id ?? "")} onChange={(e) => setForm({ ...form, member_id: e.target.value })}>
                <option value="">Selecione...</option>
                {members.map((m) => <option key={m.id} value={m.id}>{m.full_name}{m.birth_date ? ` · ${ageFrom(m.birth_date)}a` : ""}</option>)}
              </select>
            </Field>
          )}
          <Field label="Status">
            <select className="input w-full" value={String(form.status ?? "active")} onChange={(e) => setForm({ ...form, status: e.target.value })}>
              <option value="active">Ativa</option><option value="paused">Pausada</option><option value="ended">Encerrada</option>
            </select>
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Início"><Input type="date" value={String(form.start_date ?? "")} onChange={(e) => setForm({ ...form, start_date: e.target.value })} /></Field>
            <Field label="Encerramento"><Input type="date" value={String(form.end_date ?? "")} onChange={(e) => setForm({ ...form, end_date: e.target.value })} /></Field>
          </div>
          <Field label="Restrição alimentar / alergia"><Input value={String(form.dietary_restrictions ?? "")} onChange={(e) => setForm({ ...form, dietary_restrictions: e.target.value })} /></Field>
          <Field label="Observações"><Textarea rows={2} value={String(form.notes ?? "")} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2"><Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button><Button type="submit">Salvar</Button></div>
        </form>
      </Drawer>

      <Modal open={!!guardianFor} onClose={() => setGuardianFor(null)} title={`Responsáveis · ${guardianFor?.member_name ?? ""}`}>
        <div className="space-y-4">
          <Table>
            <THead><TRow><TH>Responsável</TH><TH>Parentesco</TH><TH>Principal</TH><TH /></TRow></THead>
            <TBody>
              {guardians.map((g) => (
                <TRow key={g.id}>
                  <TD>{g.member_name}</TD><TD className="text-zinc-500">{g.relationship || "—"}</TD>
                  <TD>{g.is_primary ? <Badge tone="sky">Principal</Badge> : "—"}</TD>
                  <TD className="text-right"><Button size="sm" variant="ghost" onClick={() => delG(g.id)}><X className="h-3 w-3" /></Button></TD>
                </TRow>
              ))}
              {guardians.length === 0 && <TRow><TD colSpan={4} className="text-center text-sm text-zinc-500">Nenhum responsável.</TD></TRow>}
            </TBody>
          </Table>
          <form onSubmit={addG} className="flex items-end gap-2">
            <Field label="Membro"><select className="input w-56" value={String(gForm.member_id ?? "")} onChange={(e) => setGForm({ ...gForm, member_id: e.target.value })}><option value="">Selecione...</option>{members.map((m) => <option key={m.id} value={m.id}>{m.full_name}</option>)}</select></Field>
            <Field label="Parentesco"><Input className="w-32" value={String(gForm.relationship ?? "")} onChange={(e) => setGForm({ ...gForm, relationship: e.target.value })} placeholder="Mãe, Pai..." /></Field>
            <label className="mb-2 flex items-center gap-1 text-sm"><input type="checkbox" className="h-4 w-4" checked={!!gForm.is_primary} onChange={(e) => setGForm({ ...gForm, is_primary: e.target.checked })} /> Principal</label>
            <Button type="submit" className="mb-0.5"><Plus className="h-4 w-4" /> Add</Button>
          </form>
        </div>
      </Modal>
    </>
  );
}

// ---------------------------------------------------------------------------
// Encontros
// ---------------------------------------------------------------------------

function SessionsTab({ classes, classId, setClassId, sessions, reload, openCheckin, toast }: {
  classes: KidClass[]; classId: string; setClassId: (id: string) => void; sessions: KidSession[];
  reload: () => Promise<void>; openCheckin: (id: string) => void; toast: (m: string, t?: "error") => void;
}) {
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState<Record<string, unknown>>({});
  const [lessons, setLessons] = useState<KidLesson[]>([]);

  useEffect(() => {
    const c = classes.find((x) => x.id === classId);
    if (c?.track_id) listKidLessons(c.track_id).then((r) => setLessons(r.lessons)).catch(() => setLessons([]));
    else setLessons([]);
  }, [classId, classes]);

  async function save(e: React.FormEvent) {
    e.preventDefault();
    try {
      const payload: Record<string, unknown> = { ...form, class_id: classId };
      if (payload.starts_at) payload.starts_at = new Date(String(payload.starts_at)).toISOString();
      if (form.id) await updateKidSession(String(form.id), payload);
      else await createKidSession(payload);
      toast("Encontro salvo.");
      setOpen(false);
      await reload();
    } catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function remove(s: KidSession) {
    if (!confirm("Excluir este encontro? Os check-ins serão removidos.")) return;
    try { await deleteKidSession(s.id); toast("Encontro excluído."); await reload(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }

  return (
    <>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <Field label="Turma"><select className="input w-64" value={classId} onChange={(e) => setClassId(e.target.value)}>{classes.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></Field>
        <Button onClick={() => { setForm({ starts_at: "", lesson_id: "", status: "scheduled", notes: "" }); setOpen(true); }} disabled={!classId}><Plus className="h-4 w-4" /> Novo Encontro</Button>
      </div>

      {sessions.length === 0 ? (
        <EmptyState icon={<CalendarDays className="h-10 w-10" />} title="Nenhum encontro" description="Agende um encontro e faça a chamada." />
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead><TRow><TH>Data/hora</TH><TH>Lição</TH><TH>Status</TH><TH>Presentes</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {sessions.map((s) => (
                <TRow key={s.id}>
                  <TD className="font-medium">{new Date(s.starts_at).toLocaleString("pt-BR")}</TD>
                  <TD className="text-zinc-500">{s.lesson_title || "—"}</TD>
                  <TD><Badge tone={s.status === "closed" ? "zinc" : s.status === "open" ? "green" : "sky"}>{s.status}</Badge></TD>
                  <TD>{s.checkin_count}</TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      <Button size="sm" variant="outline" onClick={() => openCheckin(s.id)}><ClipboardCheck className="h-3 w-3" /> Check-in</Button>
                      <Button size="sm" variant="ghost" onClick={() => { setForm({ id: s.id, starts_at: dtLocal(s.starts_at), lesson_id: s.lesson_id, status: s.status, notes: s.notes }); setOpen(true); }}><Pencil className="h-3 w-3" /></Button>
                      <Button size="sm" variant="ghost" onClick={() => remove(s)}><Trash2 className="h-3 w-3" /></Button>
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}

      <Drawer open={open} onClose={() => setOpen(false)} title={form.id ? "Editar Encontro" : "Novo Encontro"}>
        <form onSubmit={save} className="space-y-3">
          <Field label="Data e hora *"><Input type="datetime-local" required value={String(form.starts_at ?? "")} onChange={(e) => setForm({ ...form, starts_at: e.target.value })} /></Field>
          <Field label="Lição">
            <select className="input w-full" value={String(form.lesson_id ?? "")} onChange={(e) => setForm({ ...form, lesson_id: e.target.value })}>
              <option value="">—</option>
              {lessons.map((l) => <option key={l.id} value={l.id}>{l.position}. {l.title}</option>)}
            </select>
          </Field>
          <Field label="Status">
            <select className="input w-full" value={String(form.status ?? "scheduled")} onChange={(e) => setForm({ ...form, status: e.target.value })}>
              <option value="scheduled">Agendado</option><option value="open">Aberto</option><option value="closed">Encerrado</option>
            </select>
          </Field>
          <Field label="Observações"><Textarea rows={2} value={String(form.notes ?? "")} onChange={(e) => setForm({ ...form, notes: e.target.value })} /></Field>
          <div className="flex justify-end gap-2 pt-2"><Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button><Button type="submit">Salvar</Button></div>
        </form>
      </Drawer>
    </>
  );
}

// ---------------------------------------------------------------------------
// Check-in
// ---------------------------------------------------------------------------

function CheckinTab({ classes, sessions, sessionId, roster, onSelectSession, reloadSessions, toast }: {
  classes: KidClass[]; sessions: KidSession[]; sessionId: string; roster: KidRosterEntry[];
  onSelectSession: (id: string) => void; reloadSessions: () => Promise<void>;
  toast: (m: string, t?: "error") => void;
}) {
  useEffect(() => { if (sessions.length === 0) reloadSessions(); /* eslint-disable-next-line */ }, []);

  async function doCheckin(e: KidRosterEntry) {
    try { await kidCheckin(sessionId, { enrollment_id: e.enrollment_id }); refresh(); toast(`${e.member_name} fez check-in.`); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function doCheckout(e: KidRosterEntry) {
    const code = prompt(`Código de segurança para retirar ${e.member_name} (deixe vazio para não conferir):`, e.security_code);
    if (code === null) return;
    try { await kidCheckout(sessionId, { enrollment_id: e.enrollment_id, security_code: code }); refresh(); toast("Check-out registrado."); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function doAbsent(e: KidRosterEntry) {
    try { await kidAbsence(sessionId, { enrollment_id: e.enrollment_id, absent: true }); refresh(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  async function clear(e: KidRosterEntry) {
    try { await kidAbsence(sessionId, { enrollment_id: e.enrollment_id, absent: false }); refresh(); }
    catch (err) { toast(err instanceof Error ? err.message : "Erro", "error"); }
  }
  function refresh() { if (sessionId) onSelectSession(sessionId); }

  function printLabel(e: KidRosterEntry) {
    const w = window.open("", "_blank", "width=420,height=320");
    if (!w) return;
    const cls = sessions.find((s) => s.id === sessionId);
    w.document.write(`<!doctype html><html><head><meta charset="utf-8"><title>Etiqueta</title>
      <style>body{font-family:sans-serif;padding:16px}.code{font-size:44px;font-weight:bold;letter-spacing:6px}
      .name{font-size:20px;font-weight:600;margin:8px 0}.muted{color:#555;font-size:12px}</style></head><body>
      <div class="name">${e.member_name}</div>
      <div>Turma: ${cls?.class_name ?? ""}</div>
      <div>Responsáveis: ${e.guardians || "—"}</div>
      ${e.dietary_restrictions ? `<div class="muted">⚠ Alergia/restrição: ${e.dietary_restrictions}</div>` : ""}
      <div class="code">${e.security_code || "- - - -"}</div>
      <div class="muted">Código de segurança para retirada</div>
      </body></html>`);
    w.document.close();
    w.focus();
    w.setTimeout(() => w.print(), 400);
  }

  const present = roster.filter((r) => r.status === "present").length;

  return (
    <>
      <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
        <Field label="Encontro">
          <select className="input w-96" value={sessionId} onChange={(e) => onSelectSession(e.target.value)}>
            <option value="">Selecione um encontro...</option>
            {sessions.map((s) => <option key={s.id} value={s.id}>{s.class_name} · {new Date(s.starts_at).toLocaleString("pt-BR")} {s.lesson_title ? `· ${s.lesson_title}` : ""}</option>)}
          </select>
        </Field>
        {sessionId && <Badge tone="green">{present} presente(s)</Badge>}
      </div>

      {!sessionId ? (
        <EmptyState icon={<ClipboardCheck className="h-10 w-10" />} title="Selecione um encontro" description="Escolha um encontro para fazer a chamada." />
      ) : roster.length === 0 ? (
        <EmptyState icon={<Baby className="h-10 w-10" />} title="Sem crianças ativas" description="Matricule crianças na turma deste encontro." />
      ) : (
        <Card className="overflow-hidden p-0">
          <Table>
            <THead><TRow><TH>Criança</TH><TH>Responsáveis</TH><TH>Restrição</TH><TH>Status</TH><TH>Código</TH><TH>Entrada/Saída</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {roster.map((e) => (
                <TRow key={e.enrollment_id}>
                  <TD className="font-medium">{e.member_name}</TD>
                  <TD className="text-zinc-500 max-w-[180px] truncate">{e.guardians || "—"}</TD>
                  <TD>{e.dietary_restrictions ? <Badge tone="amber">{e.dietary_restrictions}</Badge> : "—"}</TD>
                  <TD>
                    {e.status === "present" ? <Badge tone="green"><CheckCircle className="h-3 w-3" /> Presente</Badge>
                      : e.status === "absent" ? <Badge tone="red"><XCircle className="h-3 w-3" /> Ausente</Badge>
                      : <Badge tone="zinc">—</Badge>}
                  </TD>
                  <TD className="font-mono">{e.security_code || "—"}</TD>
                  <TD className="text-xs text-zinc-500">
                    {e.checkin_at ? new Date(e.checkin_at).toLocaleTimeString("pt-BR") : "—"}
                    {e.checkout_at ? ` → ${new Date(e.checkout_at).toLocaleTimeString("pt-BR")}` : ""}
                  </TD>
                  <TD className="text-right">
                    <div className="flex justify-end gap-1">
                      {e.status !== "present" && <Button size="sm" onClick={() => doCheckin(e)}><CheckCircle className="h-3 w-3" /> Check-in</Button>}
                      {e.status === "present" && !e.checkout_at && <Button size="sm" variant="outline" onClick={() => doCheckout(e)}>Check-out</Button>}
                      {e.status === "present" && <Button size="sm" variant="ghost" onClick={() => printLabel(e)} title="Imprimir etiqueta"><Printer className="h-3 w-3" /></Button>}
                      {e.status !== "absent" && <Button size="sm" variant="ghost" onClick={() => doAbsent(e)} title="Marcar ausente"><MinusCircle className="h-3 w-3" /></Button>}
                      {e.status === "absent" && <Button size="sm" variant="ghost" onClick={() => clear(e)} title="Limpar"><X className="h-3 w-3" /></Button>}
                    </div>
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        </Card>
      )}
    </>
  );
}

// ---------------------------------------------------------------------------
// Evolução
// ---------------------------------------------------------------------------

function EvolutionTab({ classes, classId, setClassId, evolution, reload, toast }: {
  classes: KidClass[]; classId: string; setClassId: (id: string) => void;
  evolution: KidEvolution | null; reload: () => void; toast: (m: string, t?: "error") => void;
}) {
  const [range, setRange] = useState({ from: "", to: "" });

  return (
    <>
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <Field label="Turma"><select className="input w-56" value={classId} onChange={(e) => setClassId(e.target.value)}>{classes.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></Field>
        <Field label="De"><Input type="date" value={range.from} onChange={(e) => setRange({ ...range, from: e.target.value })} /></Field>
        <Field label="Até"><Input type="date" value={range.to} onChange={(e) => setRange({ ...range, to: e.target.value })} /></Field>
        <Button onClick={reload} className="mb-0.5"><Search className="h-4 w-4" /> Aplicar</Button>
      </div>

      {evolution && (
        <>
          <div className="mb-4 grid gap-3 sm:grid-cols-3">
            <StatCard label="Encontros no período" value={String(evolution.total_sessions)} tone="sky" />
            <StatCard label="Lições da trilha" value={String(evolution.total_lessons)} tone="zinc" />
            <StatCard label="Crianças ativas" value={String(evolution.rows.length)} tone="green" />
          </div>
          {evolution.rows.length === 0 ? (
            <EmptyState icon={<TrendingUp className="h-10 w-10" />} title="Sem dados" description="Sem crianças ativas ou encontros no período." />
          ) : (
            <Card className="overflow-hidden p-0">
              <Table>
                <THead><TRow><TH>Criança</TH><TH>Presenças</TH><TH>Frequência</TH><TH>Lições</TH><TH>Progresso</TH><TH>Última lição</TH></TRow></THead>
                <TBody>
                  {evolution.rows.map((r) => (
                    <TRow key={r.enrollment_id}>
                      <TD className="font-medium">{r.member_name}</TD>
                      <TD>{r.attended}/{r.total_sessions}</TD>
                      <TD><PctBar value={r.attendance_pct} /></TD>
                      <TD>{r.lessons_attended}/{r.total_lessons}</TD>
                      <TD><PctBar value={r.progress_pct} /></TD>
                      <TD className="text-zinc-500">{r.last_lesson || "—"}</TD>
                    </TRow>
                  ))}
                </TBody>
              </Table>
            </Card>
          )}
        </>
      )}
    </>
  );
}

function PctBar({ value }: { value: number }) {
  const v = Math.round(value);
  return (
    <div className="flex items-center gap-2">
      <div className="h-2 w-24 overflow-hidden rounded-full bg-zinc-200 dark:bg-zinc-800">
        <div className="h-full rounded-full bg-sky-500" style={{ width: `${Math.min(100, v)}%` }} />
      </div>
      <span className="text-xs text-zinc-500">{v}%</span>
    </div>
  );
}
