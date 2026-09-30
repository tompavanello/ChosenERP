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
  listProgramacoes, createProgramacao, updateProgramacao, deleteProgramacao,
  generateProgramacaoEvents, type Programacao,
} from "@/lib/api";
import { WEEKDAYS, WEEKDAY_ORDER, PROGRAM_KINDS } from "@/lib/constants";

const todayISO = () => new Date().toISOString().slice(0, 10);
function addDaysISO(days: number) {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}

const EMPTY = {
  name: "", kind: "culto", title: "", weekday: "0", start_time: "19:00",
  duration_minutes: "90", location: "", notes: "", is_active: "true", sort_order: "0",
};

/**
 * ProgramacaoPage e a GRADE de horarios recorrentes da igreja: cultos, relogio
 * de oracao, celulas, EBD, ensaios e reunioes - cada um com um TIPO FIXO.
 * A grade e publicada na agenda de Eventos (`church_events`), de onde a chamada
 * nominal, os convocados e o calendario seguem valendo.
 */
export default function ProgramacaoPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("members.write") || hasPerm("ministries.write");

  const [itens, setItens] = useState<Programacao[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: Programacao }>({ open: false });
  const [form, setForm] = useState({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const [publish, setPublish] = useState<{ open: boolean; item?: Programacao }>({ open: false });
  const [pubForm, setPubForm] = useState({ from: todayISO(), to: addDaysISO(30) });
  const [publishing, setPublishing] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await listProgramacoes();
      setItens(r.programacoes);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar a programacao", "error");
      setItens([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const ordenados = useMemo(() => {
    const arr = [...(itens ?? [])];
    arr.sort((a, b) => a.weekday - b.weekday || a.start_time.localeCompare(b.start_time));
    return arr;
  }, [itens]);

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
    });
    setDrawer({ open: true, editing: p });
  }

  async function save(e: React.FormEvent) {
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
    };
    try {
      if (drawer.editing) await updateProgramacao(drawer.editing.id, payload);
      else await createProgramacao(payload);
      toast("Programacao salva.");
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(p: Programacao) {
    if (!confirm(`Excluir "${p.name}"? As ocorrencias ja publicadas na agenda permanecem.`)) return;
    try {
      await deleteProgramacao(p.id);
      toast("Programacao excluida.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  function openPublish(item?: Programacao) {
    setPubForm({ from: todayISO(), to: addDaysISO(30) });
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
      const r = await generateProgramacaoEvents({
        from: pubForm.from,
        to: pubForm.to,
        programacao_id: publish.item?.id,
      });
      toast(r.created > 0
        ? `${r.created} ocorrencia(s) publicada(s) na agenda.`
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
        title="Programacao"
        description="Horarios recorrentes (cultos, oracao, celulas, EBD, ensaios) publicados na agenda"
        actions={canWrite ? (
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => openPublish()}><CalendarPlus className="h-4 w-4" /> Publicar na agenda</Button>
            <Button onClick={openCreate}><Plus className="h-4 w-4" /> Nova programacao</Button>
          </div>
        ) : undefined}
      />

      <Card className="mb-3 p-3">
        <p className="text-xs text-zinc-500">
          Defina aqui os horarios fixos. Ao <span className="font-medium">publicar</span>, cada ocorrencia do
          periodo vira um evento na <Link href="/dashboard/events" className="text-sky-600 hover:underline">agenda</Link>,
          onde ficam a chamada e os convocados. Publicar de novo o mesmo periodo nao duplica (eventos avulsos ficam em Eventos).
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
                <TH>Tipo</TH><TH>Dia</TH><TH>Horario</TH><TH>Nome</TH>
                <TH>Local</TH><TH>Duracao</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH>
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
                    <TD className="text-sm text-zinc-500">{p.location ?? "-"}</TD>
                    <TD className="text-sm text-zinc-500">{p.duration_minutes} min</TD>
                    <TD><Badge tone={p.is_active ? "green" : "zinc"} className="text-[10px]">{p.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                    <TD>
                      {canWrite && (
                        <div className="flex justify-end gap-1">
                          <Button variant="ghost" className="h-8 px-2" title="Publicar na agenda" onClick={() => openPublish(p)}>
                            <CalendarPlus className="h-4 w-4" />
                          </Button>
                          <Button variant="ghost" className="h-8 px-2" title="Editar" onClick={() => openEdit(p)}>
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button variant="ghost" className="h-8 px-2 text-red-600" title="Excluir" onClick={() => remove(p)}>
                            <Trash2 className="h-4 w-4" />
                          </Button>
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

      {/* Drawer da programacao */}
      <Drawer open={drawer.open} onClose={() => setDrawer({ open: false })} title={drawer.editing ? "Editar programacao" : "Nova programacao"}>
        <form onSubmit={save} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
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
          <p className="text-xs text-zinc-400">
            Idempotente: ocorrencias ja publicadas (mesma programacao e horario) sao ignoradas.
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
