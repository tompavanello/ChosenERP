"use client";

import { useEffect, useState } from "react";
import { BookOpen, Download, ExternalLink, FileText, Link2, Plus, Trash2 } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Input, Field, Select } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows, EmptyState } from "@/components/ui/skeleton";
import { Drawer } from "@/components/ui/modal";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/components/providers/auth-provider";
import {
  listMaterials, uploadMaterial, createMaterialLink, deleteMaterial, downloadMaterial,
  listGroups, type StudyMaterial, type SmallGroup,
} from "@/lib/api";

function fileSize(n?: number | null): string {
  if (!n) return "";
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export default function MaterialsPage() {
  const { toast } = useToast();
  const { hasPerm } = useAuth();
  const canWrite = hasPerm("ministries.write");

  const [items, setItems] = useState<StudyMaterial[] | null>(null);
  const [groups, setGroups] = useState<SmallGroup[]>([]);
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<"file" | "link">("file");
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ title: "", description: "", group_id: "", url: "", is_published: "true" });
  const [file, setFile] = useState<File | null>(null);

  const reload = async () => {
    setItems(await listMaterials().then((r) => r.materials).catch(() => []));
  };
  useEffect(() => {
    reload();
    listGroups().then((r) => setGroups(r.groups)).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openCreate() {
    setForm({ title: "", description: "", group_id: "", url: "", is_published: "true" });
    setFile(null);
    setMode("file");
    setOpen(true);
  }

  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!form.title.trim()) {
      toast("Informe o titulo.", "error");
      return;
    }
    setSaving(true);
    try {
      if (mode === "link") {
        if (!/^https?:\/\//.test(form.url.trim())) {
          toast("Informe uma URL http/https.", "error");
          setSaving(false);
          return;
        }
        await createMaterialLink({
          title: form.title, description: form.description, group_id: form.group_id,
          url: form.url, is_published: form.is_published === "true",
        });
      } else {
        if (!file) {
          toast("Selecione um arquivo.", "error");
          setSaving(false);
          return;
        }
        const fd = new FormData();
        fd.set("title", form.title);
        fd.set("description", form.description);
        fd.set("group_id", form.group_id);
        fd.set("is_published", form.is_published);
        fd.set("file", file);
        await uploadMaterial(fd);
      }
      toast("Material salvo.");
      setOpen(false);
      await reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(m: StudyMaterial) {
    if (!confirm(`Excluir o material "${m.title}"?`)) return;
    try {
      await deleteMaterial(m.id);
      toast("Material excluido.");
      await reload();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro", "error");
    }
  }

  return (
    <div className="page">
      <PageHeader title="Materiais de estudo" description="Arquivos e links para a igreja ou para um grupo/celula (app do membro)" />

      <div className="mb-4 flex justify-end">
        {canWrite && <Button onClick={openCreate}><Plus className="h-4 w-4" /> Novo material</Button>}
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <SkeletonRows rows={5} />
        ) : items.length === 0 ? (
          <EmptyState icon={<BookOpen className="h-10 w-10" />} title="Nenhum material" description="Envie apostilas, estudos ou links para o app do membro." />
        ) : (
          <Table>
            <THead><TRow><TH>Material</TH><TH>Destino</TH><TH>Tipo</TH><TH>Situacao</TH><TH className="text-right">Acoes</TH></TRow></THead>
            <TBody>
              {items.map((m) => (
                <TRow key={m.id}>
                  <TD>
                    <div className="font-medium">{m.title}</div>
                    {m.description && <div className="text-xs text-zinc-500">{m.description}</div>}
                  </TD>
                  <TD className="text-zinc-500">{m.group_name ?? "Igreja (todos)"}</TD>
                  <TD>
                    <Badge tone="sky">
                      <span className="inline-flex items-center gap-1">
                        {m.kind === "link" ? <Link2 className="h-3 w-3" /> : <FileText className="h-3 w-3" />}
                        {m.kind === "link" ? "Link" : `Arquivo ${fileSize(m.file_size)}`}
                      </span>
                    </Badge>
                  </TD>
                  <TD><Badge tone={m.is_published ? "green" : "zinc"}>{m.is_published ? "Publicado" : "Rascunho"}</Badge></TD>
                  <TD className="text-right">
                    {m.kind === "link" ? (
                      <Button variant="ghost" size="sm" onClick={() => window.open(m.url ?? "#", "_blank")}><ExternalLink className="h-3.5 w-3.5" /> Abrir</Button>
                    ) : (
                      <Button variant="ghost" size="sm" onClick={() => downloadMaterial(m.id, m.file_name ?? "material")}><Download className="h-3.5 w-3.5" /> Baixar</Button>
                    )}
                    {canWrite && <Button variant="ghost" size="sm" title="Excluir" onClick={() => remove(m)}><Trash2 className="h-3.5 w-3.5 text-red-500" /></Button>}
                  </TD>
                </TRow>
              ))}
            </TBody>
          </Table>
        )}
      </Card>

      <Drawer open={open} onClose={() => setOpen(false)} size="lg" title="Novo material">
        <form onSubmit={save} className="space-y-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Field label="Titulo *" className="sm:col-span-2"><Input required className="h-8 text-sm" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} /></Field>
            <Field label="Descricao" className="sm:col-span-2"><Input className="h-8 text-sm" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} /></Field>
            <Field label="Destino">
              <Select className="h-8 text-sm" value={form.group_id} onChange={(e) => setForm({ ...form, group_id: e.target.value })}>
                <option value="">Igreja (todos)</option>
                {groups.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
              </Select>
            </Field>
            <Field label="Situacao">
              <Select className="h-8 text-sm" value={form.is_published} onChange={(e) => setForm({ ...form, is_published: e.target.value })}>
                <option value="true">Publicado</option><option value="false">Rascunho</option>
              </Select>
            </Field>
            <Field label="Tipo" className="sm:col-span-2">
              <Select className="h-8 text-sm" value={mode} onChange={(e) => setMode(e.target.value as "file" | "link")}>
                <option value="file">Arquivo (PDF, Office, imagem, audio...)</option>
                <option value="link">Link (URL)</option>
              </Select>
            </Field>
            {mode === "file" ? (
              <Field label="Arquivo * (ate 25MB)" className="sm:col-span-2">
                <input type="file" className="block w-full text-sm" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
              </Field>
            ) : (
              <Field label="URL *" className="sm:col-span-2"><Input required className="h-8 text-sm" placeholder="https://..." value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} /></Field>
            )}
          </div>
          <div className="flex justify-end gap-2 border-t border-zinc-100 pt-3 dark:border-zinc-800">
            <Button variant="ghost" type="button" className="h-8 text-sm" onClick={() => setOpen(false)}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
