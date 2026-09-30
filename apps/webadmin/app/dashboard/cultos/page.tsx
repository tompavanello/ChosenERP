"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { CalendarHeart, CalendarPlus, Pencil, Plus, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge, type Tone } from "@/components/ui/badge";
import { Drawer, Modal } from "@/components/ui/modal";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listCultos, createCulto, updateCulto, deleteCulto, generateCultoEvents,
  listEventKinds, type Culto, type EventKind,
} from "@/lib/api";
import { WEEKDAYS, WEEKDAY_ORDER } from "@/lib/constants";

const todayISO = () => new Date().toISOString().slice(0, 10);
function addDaysISO(days: number) {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

const EMPTY = {
  name: "", event_kind_id: "", weekday: "0", start_time: "19:00",
  duration_minutes: "90", location: "", notes: "", is_active: "true", sort_order: "0",
};

/**
 * CultosPage e a GRADE de horarios dos cultos da igreja. Cada culto vira uma
 * definicao recorrente (dia + hora + tipo + local) que pode ser PUBLICADA na
 * agenda de Eventos (`church_events`) por um periodo - de onde a chamada
 * nominal, os convocados e o calendario seguem valendo.
 */
export default function CultosPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("members.write") || hasPerm("ministries.write");

  const [cultos, setCultos] = useState<Culto[] | null>(null);
  const [kinds, setKinds] = useState<EventKind[]>([]);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Culto }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const [publish, setPublish] = useState<{ open: boolean; culto?: Culto }>({ open: false });
  const [pubForm, setPubForm] = useState({ from: todayISO(), to: addDaysISO(30) });
  const [publishing, setPublishing] = useState(false);

  const load = useCallback(async () => {
    try {
      const [c, k] = await Promise.all([listCultos(), listEventKinds()]);
      setCultos(c.cultos);
      setKinds(k.kinds);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar cultos", "error");
      setCultos([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const ordenados = useMemo(() => {
    const arr = [...(cultos ?? [])];
    arr.sort((a, b) => a.weekday - b.weekday || a.start_time.localeCompare(b.start_time));
    return arr;
  }, [cultos]);

  function openCreate() {
    setForm({ ...EMPTY, event_kind_id: kinds.find((k) => k.is_active && k.slug === "culto")?.id ?? "" });
    setDrawer({ open: true });
  }

  function openEdit(c: Culto) {
    setForm({
      name: c.name,
      event_kind_id: c.event_kind_id ?? "",
      weekday: String(c.weekday),
      start_time: c.start_time.slice(0, 5),
      duration_minutes: String(c.duration_minutes || 90),
      location: c.location ?? "",
      notes: c.notes ?? "",
      is_active: c.is_active ? "true" : "false",
      sort_order: String(c.sort_order),
    });
    setDrawer({ open: true, editing: c });
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.start_time) return;
    setSaving(true);
    const payload = {
      name: form.name.trim(),
      event_kind_id: form.event_kind_id || undefined,
      weekday: Number(form.weekday),
      start_time: form.start_time,
      duration_minutes: Number(form.duration_minutes) || 90,
      location: form.location || undefined,
      notes: form.notes || undefined,
      is_active: form.is_active === "true",
      sort_order: Number(form.sort_order) || 0,
    };
    try {
      if (drawer.editing) await updateCulto(drawer.editing.id, payload);
      else await createCulto(payload);
      toast("Culto salvo.");
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar culto", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(c: Culto) {
    if (!confirm(`Excluir o culto "${c.name}"? Os eventos ja publicados na agenda permanecem.`)) return;
    try {
      await deleteCulto(c.id);
      toast("Culto excluido.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function openPublish(culto?: Culto) {
    setPubForm({ from: todayISO(), to: addDaysISO(30) });
    setPublish({ open: true, culto });
  }

  async function doPublish(e: React.FormEvent) {
    e.preventDefault();
    if (pubForm.to < pubForm.from) {
      toast("Periodo invalido: 'ate' menor que 'de'.", "error");
      return;
    }
    setPublishing(true);
    try {
      const r = await generateCultoEvents({
        from: pubForm.from,
        to: pubForm.to,
        culto_id: publish.culto?.id,
      });
      toast(r.created > 0
        ? `${r.created} culto(s) publicado(s) na agenda.`
        : "Nada a publicar: as ocorrencias do periodo ja estavam na agenda.");
      setPublish({ open: false });
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao publicar", "error");
    } finally {
      setPublishing(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Cultos"
        description="Grade de horarios dos cultos; publique as ocorrencias na agenda de eventos"
        actions={canWrite ? (
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => openPublish()}><CalendarPlus className="h-4 w-4" /> Publicar na agenda</Button>
            <Button onClick={openCreate}><Plus className="h-4 w-4" /> Novo culto</Button>
          </div>
        ) : undefined}
      />

      <Card className="mb-3 p-3">
        <p className="text-xs text-zinc-500">
          Defina aqui os horarios fixos. Ao <span className="font-medium">publicar</span>, cada ocorrencia do
          periodo e criada como um evento na aba <Link href="/dashboard/events" className="text-sky-600 hover:underline">Eventos / Calendario</Link>,
          onde ficam a chamada e os convocados. Publicar de novo o mesmo periodo nao duplica.
        </p>
      </Card>

      <Card className="overflow-hidden p-0">
        {cultos === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : ordenados.length === 0 ? (
          <EmptyState
            icon={<CalendarHeart className="h-10 w-10" />}
            title="Nenhum culto na grade"
            description="Cadastre os horarios fixos (ex.: Culto de Domingo 19:00) e publique na agenda."
          />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Dia</TH><TH>Horario</TH><TH>Culto</TH><TH>Tipo</TH>
                <TH>Local</TH><TH>Duracao</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH>
              </TRow>
            </THead>
            <TBody>
              {ordenados.map((c) => (
                <TRow key={c.id}>
                  <TD className="text-sm text-zinc-500">{WEEKDAYS[c.weekday] ?? c.weekday}</TD>
                  <TD className="font-medium tabular-nums">{c.start_time}</TD>
                  <TD className="font-medium">{c.name}</TD>
                  <TD>
                    {c.event_kind_name
                      ? <Badge tone="sky" className="text-[10px]">{c.event_kind_name}</Badge>
                      : <span className="text-xs text-zinc-400">-</span>}
                  </TD>
                  <TD className="text-sm text-zinc-500">{c.location ?? "-"}</TD>
                  <TD className="text-sm text-zinc-500">{c.duration_minutes} min</TD>
                  <TD><Badge tone={c.is_active ? "green" : "zinc"} className="text-[10px]">{c.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                  <TD>
                    {canWrite && (
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" className="h-8 px-2" title="Publicar na agenda" onClick={() => openPublish(c)}>
                          <CalendarPlus className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(c)}>
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button variant="ghost" className="h-8 px-2 text-red-600" title="Excluir" onClick={() => remove(c)}>
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      {/* Drawer do culto */}
      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} title={drawer.editing ? "Editar culto" : "Novo culto"}>
        <form onSubmit={save} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nome" required className="sm:col-span-2">
            <Input required className="h-8 text-sm" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Ex: Culto de Domingo" />
          </Field>
          <Field label="Tipo de evento" hint="Usado ao publicar na agenda.">
            <Select className="h-8 text-sm" value={form.event_kind_id} onChange={(e) => setForm({ ...form, event_kind_id: e.target.value })}>
              <option value="">-</option>
              {kinds.filter((k) => k.is_active).map((k) => <option key={k.id} value={k.id}>{k.name}</option>)}
            </Select>
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

      {/* Modal de publicacao na agenda */}
      <Modal open={publish.open} onClose={() => setPublish({ open: false })} title="Publicar na agenda" size="sm">
        <form onSubmit={doPublish} className="space-y-3">
          <p className="text-xs text-zinc-500">
            {publish.culto
              ? <>Publica as ocorrencias de <span className="font-medium">{publish.culto.name}</span> no periodo.</>
              : <>Publica as ocorrencias de <span className="font-medium">todos os cultos ativos</span> no periodo.</>}
          </p>
          <div className="grid grid-cols-2 gap-3">
            <Field label="De" required>
              <Input required type="date" className="h-8 text-sm" value={pubForm.from} onChange={(e) => setPubForm({ ...pubForm, from: e.target.value })} />
            </Field>
            <Field label="Ate" required>
              <Input required type="date" className="h-8 text-sm" value={pubForm.to} onChange={(e) => setPubForm({ ...pubForm, to: e.target.value })} />
            </Field>
          </div>
          <p className="text-xs text-zinc-400">
            Idempotente: ocorrencias ja publicadas (mesmo culto e horario) sao ignoradas.
          </p>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setPublish({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={publishing}>{publishing ? "Publicando..." : "Publicar"}</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
