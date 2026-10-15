"use client";

import { useEffect, useState } from "react";
import { Pencil, Play, Plus, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Drawer } from "@/components/ui/modal";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import { createVideo, deleteVideo, listVideos, updateVideo, type ContentVideo } from "@/lib/api";
import { datePt } from "@/lib/format";

const KINDS = [
  { value: "sermao", label: "Sermão" },
  { value: "estudo", label: "Estudo" },
  { value: "musica", label: "Música" },
  { value: "testemunho", label: "Testemunho" },
  { value: "outro", label: "Outro" },
];

type FormState = {
  title: string;
  url: string;
  kind: string;
  speaker: string;
  recorded_at: string;
  is_published: boolean;
};

const EMPTY: FormState = { title: "", url: "", kind: "sermao", speaker: "", recorded_at: "", is_published: true };

export default function VideosPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("content.write");

  const [videos, setVideos] = useState<ContentVideo[] | null>(null);
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<ContentVideo | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY);
  const [saving, setSaving] = useState(false);

  async function load() {
    try {
      setVideos((await listVideos()).videos);
    } catch (e) {
      toast(e instanceof Error ? e.message : "Erro", "error");
      setVideos([]);
    }
  }
  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openNew() {
    setEditing(null);
    setForm(EMPTY);
    setOpen(true);
  }
  function openEdit(v: ContentVideo) {
    setEditing(v);
    setForm({
      title: v.title,
      url: v.youtube_id,
      kind: v.kind,
      speaker: v.speaker ?? "",
      recorded_at: v.recorded_at ?? "",
      is_published: v.is_published,
    });
    setOpen(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = { ...form };
      if (editing) await updateVideo(editing.id, payload);
      else await createVideo(payload);
      toast(editing ? "Vídeo atualizado." : "Vídeo criado.");
      setOpen(false);
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(v: ContentVideo) {
    if (!confirm(`Excluir "${v.title}"?`)) return;
    try {
      await deleteVideo(v.id);
      toast("Vídeo excluído.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader
        title="Vídeos"
        description="Sermões, estudos e músicas do YouTube para o app do membro"
        actions={canWrite ? <Button onClick={openNew}><Plus className="h-4 w-4" /> Novo vídeo</Button> : undefined}
      />

      <Card className="overflow-hidden p-0">
        {videos === null ? (
          <SkeletonRows rows={4} />
        ) : videos.length === 0 ? (
          <EmptyState icon={<Play className="h-10 w-10" />} title="Nenhum vídeo" description="Adicione links do YouTube para aparecerem no app." />
        ) : (
          <Table>
            <THead>
              <TRow><TH>Vídeo</TH><TH>Tipo</TH><TH>Preletor</TH><TH>Data</TH><TH>Situação</TH><TH className="text-right">Ações</TH></TRow>
            </THead>
            <TBody>
              {videos.map((v) => (
                <TRow key={v.id}>
                  <TD>
                    <div className="flex items-center gap-2">
                      {v.thumbnail_url && (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={v.thumbnail_url} alt="" className="h-9 w-16 rounded object-cover" />
                      )}
                      <span className="font-medium">{v.title}</span>
                    </div>
                  </TD>
                  <TD className="text-zinc-500">{KINDS.find((k) => k.value === v.kind)?.label ?? v.kind}</TD>
                  <TD className="text-zinc-500">{v.speaker ?? "-"}</TD>
                  <TD className="text-zinc-500">{datePt(v.recorded_at)}</TD>
                  <TD><Badge tone={v.is_published ? "green" : "zinc"}>{v.is_published ? "Publicado" : "Rascunho"}</Badge></TD>
                  <TD className="text-right">
                    {canWrite && (
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(v)}><Pencil className="h-3.5 w-3.5" /></Button>
                        <Button variant="ghost" size="sm" onClick={() => remove(v)}><Trash2 className="h-3.5 w-3.5 text-red-600" /></Button>
                      </div>
                    )}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} title={editing ? "Editar vídeo" : "Novo vídeo"}>
        <form onSubmit={save} className="space-y-3">
          <Field label="Título *"><Input required value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
          <Field label="URL ou ID do YouTube *"><Input required value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} placeholder="https://youtu.be/..." /></Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Tipo">
              <Select value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
                {KINDS.map((k) => (<option key={k.value} value={k.value}>{k.label}</option>))}
              </Select>
            </Field>
            <Field label="Preletor"><Input value={form.speaker} onChange={(e) => setForm({ ...form, speaker: e.target.value })} /></Field>
          </div>
          <Field label="Data"><Input type="date" value={form.recorded_at} onChange={(e) => setForm({ ...form, recorded_at: e.target.value })} /></Field>
          <label className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-300">
            <input type="checkbox" checked={form.is_published} onChange={(e) => setForm({ ...form, is_published: e.target.checked })} />
            Publicado no app
          </label>
          <div className="flex justify-end gap-2 pt-2">
            <Button variant="ghost" type="button" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
