"use client";

import { useEffect, useState } from "react";
import { BookMarked, Pencil, Plus, Sparkles, Trash2, X } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Textarea } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  createDevotionalPlan,
  deleteDevotionalPlan,
  generateDevotional,
  getDevotionalPlan,
  listDevotionalPlans,
  putDevotionalDays,
  updateDevotionalPlan,
  type DevotionalDay,
  type DevotionalPlan,
} from "@/lib/api";
import { datePt } from "@/lib/format";

type PlanForm = {
  title: string;
  description: string;
  cover_image_url: string;
  start_date: string;
  is_active: boolean;
};
const EMPTY_PLAN: PlanForm = { title: "", description: "", cover_image_url: "", start_date: "", is_active: true };

function emptyDay(n: number): DevotionalDay {
  return { day_number: n, title: "", bible_reference: "", bible_text: "", body: "", prayer: "", video_url: "" };
}

export default function DevotionalsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("content.write");

  const [plans, setPlans] = useState<DevotionalPlan[] | null>(null);
  const [open, setOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [planForm, setPlanForm] = useState<PlanForm>(EMPTY_PLAN);
  const [days, setDays] = useState<DevotionalDay[]>([]);
  const [ai, setAi] = useState({ theme: "", reference: "", audience: "", days: 7 });
  const [generating, setGenerating] = useState(false);
  const [saving, setSaving] = useState(false);

  async function load() {
    try {
      setPlans((await listDevotionalPlans()).plans);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
      setPlans([]);
    }
  }
  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openNew() {
    setEditingId(null);
    setPlanForm(EMPTY_PLAN);
    setDays([emptyDay(1)]);
    setAi({ theme: "", reference: "", audience: "", days: 7 });
    setOpen(true);
  }

  async function openEdit(p: DevotionalPlan) {
    setEditingId(p.id);
    setPlanForm({
      title: p.title,
      description: p.description ?? "",
      cover_image_url: p.cover_image_url ?? "",
      start_date: p.start_date ?? "",
      is_active: p.is_active,
    });
    setDays([]);
    setOpen(true);
    try {
      const d = await getDevotionalPlan(p.id);
      setDays(d.days.map((x) => ({ ...x })));
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  function patchDay(i: number, patch: Partial<DevotionalDay>) {
    setDays((prev) => prev.map((d, idx) => (idx === i ? { ...d, ...patch } : d)));
  }
  function addDay() {
    setDays((prev) => [...prev, emptyDay(prev.length + 1)]);
  }
  function removeDay(i: number) {
    setDays((prev) => prev.filter((_, idx) => idx !== i).map((d, idx) => ({ ...d, day_number: idx + 1 })));
  }

  async function generate() {
    setGenerating(true);
    try {
      const r = await generateDevotional(ai);
      setDays(r.days.map((d, i) => ({ ...d, day_number: i + 1 })));
      toast("Rascunho gerado pela IA. Revise antes de salvar.");
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erro";
      toast(msg === "ai_disabled" ? "IA não configurada (AI_API_KEY)." : msg, "error");
    } finally {
      setGenerating(false);
    }
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = {
        title: planForm.title,
        description: planForm.description || undefined,
        cover_image_url: planForm.cover_image_url || undefined,
        start_date: planForm.start_date || undefined,
        is_active: planForm.is_active,
      };
      let id = editingId;
      if (id) await updateDevotionalPlan(id, payload);
      else id = (await createDevotionalPlan(payload)).id;
      await putDevotionalDays(id, days);
      toast("Plano devocional salvo.");
      setOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(p: DevotionalPlan) {
    if (!confirm(`Excluir o plano "${p.title}"?`)) return;
    try {
      await deleteDevotionalPlan(p.id);
      toast("Plano excluído.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Devocionais"
        description="Planos devocionais com dias e progresso para o app do membro"
        actions={canWrite ? <Button onClick={openNew}><Plus className="h-4 w-4" /> Novo plano</Button> : undefined}
      />

      <Card className="overflow-hidden p-0">
        {plans === null ? (
          <SkeletonRows rows={4} />
        ) : plans.length === 0 ? (
          <EmptyState icon={<BookMarked className="h-10 w-10" />} title="Nenhum plano" description="Crie um plano devocional (com ajuda da IA) para o app." />
        ) : (
          <Table>
            <THead><TRow><TH>Título</TH><TH>Dias</TH><TH>Início</TH><TH>Situação</TH><TH className="text-right">Ações</TH></TRow></THead>
            <TBody>
              {plans.map((p) => (
                <TRow key={p.id}>
                  <TD className="font-medium">{p.title}</TD>
                  <TD className="text-zinc-500">{p.day_count}</TD>
                  <TD className="text-zinc-500">{datePt(p.start_date)}</TD>
                  <TD><Badge tone={p.is_active ? "green" : "zinc"}>{p.is_active ? "Ativo" : "Inativo"}</Badge></TD>
                  <TD className="text-right">
                    {canWrite && (
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(p)}><Pencil className="h-3.5 w-3.5" /></Button>
                        <Button variant="ghost" size="sm" onClick={() => remove(p)}><Trash2 className="h-3.5 w-3.5 text-red-600" /></Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title={editingId ? "Editar plano devocional" : "Novo plano devocional"}>
        <form onSubmit={save} className="space-y-4">
          <Field label="Título *"><Input required value={planForm.title} onChange={(e) => setPlanForm({ ...planForm, title: e.target.value })} /></Field>
          <Field label="Descrição"><Textarea value={planForm.description} onChange={(e) => setPlanForm({ ...planForm, description: e.target.value })} /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Início"><Input type="date" value={planForm.start_date} onChange={(e) => setPlanForm({ ...planForm, start_date: e.target.value })} /></Field>
            <Field label="Capa (URL)"><Input value={planForm.cover_image_url} onChange={(e) => setPlanForm({ ...planForm, cover_image_url: e.target.value })} /></Field>
          </div>
          <label className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-300">
            <input type="checkbox" checked={planForm.is_active} onChange={(e) => setPlanForm({ ...planForm, is_active: e.target.checked })} />
            Ativo (visível no app)
          </label>

          <Card className="bg-zinc-50 dark:bg-zinc-900">
            <p className="mb-2 flex items-center gap-2 text-sm font-medium"><Sparkles className="h-4 w-4 text-[var(--brand,theme(colors.sky.500))]" /> Gerar com IA</p>
            <div className="grid grid-cols-2 gap-2">
              <Field label="Tema"><Input value={ai.theme} onChange={(e) => setAi({ ...ai, theme: e.target.value })} placeholder="Fé em tempos difíceis" /></Field>
              <Field label="Referência"><Input value={ai.reference} onChange={(e) => setAi({ ...ai, reference: e.target.value })} placeholder="Salmos 23" /></Field>
              <Field label="Público"><Input value={ai.audience} onChange={(e) => setAi({ ...ai, audience: e.target.value })} placeholder="Jovens" /></Field>
              <Field label="Dias"><Input type="number" min={1} max={31} value={ai.days} onChange={(e) => setAi({ ...ai, days: Number(e.target.value) })} /></Field>
            </div>
            <Button type="button" variant="outline" className="mt-2 w-full" onClick={generate} disabled={generating}>
              <Sparkles className="h-4 w-4" /> {generating ? "Gerando..." : "Gerar rascunho"}
            </Button>
            <p className="mt-1 text-xs text-zinc-500">A IA gera um rascunho; revise e edite antes de salvar.</p>
          </Card>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-sm font-medium">Dias ({days.length})</p>
              <Button type="button" variant="ghost" size="sm" onClick={addDay}><Plus className="h-3.5 w-3.5" /> Dia</Button>
            </div>
            {days.map((d, i) => (
              <Card key={i} className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-semibold">Dia {d.day_number}</span>
                  <button type="button" onClick={() => removeDay(i)} className="text-zinc-400 hover:text-red-600"><X className="h-4 w-4" /></button>
                </div>
                <Input value={d.title} onChange={(e) => patchDay(i, { title: e.target.value })} placeholder="Título do dia" />
                <div className="grid grid-cols-2 gap-2">
                  <Input value={d.bible_reference ?? ""} onChange={(e) => patchDay(i, { bible_reference: e.target.value })} placeholder="Referência (João 3:16)" />
                  <Input value={d.video_url ?? ""} onChange={(e) => patchDay(i, { video_url: e.target.value })} placeholder="Vídeo (YouTube, opcional)" />
                </div>
                <Textarea value={d.bible_text ?? ""} onChange={(e) => patchDay(i, { bible_text: e.target.value })} placeholder="Texto bíblico" />
                <Textarea value={d.body} onChange={(e) => patchDay(i, { body: e.target.value })} placeholder="Reflexão" />
                <Textarea value={d.prayer ?? ""} onChange={(e) => patchDay(i, { prayer: e.target.value })} placeholder="Oração" />
              </Card>
            ))}
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
