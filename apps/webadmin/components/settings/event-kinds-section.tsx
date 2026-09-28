"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { CalendarHeart, Pencil, Plus, Trash2 } from "lucide-react";
import { Badge, type Tone } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Table, THead, TBody, TRow, TH, TD } from "@/components/ui/table";
import { Drawer } from "@/components/ui/modal";
import { Field, Input, Select } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listMemberEventKinds, createMemberEventKind, updateMemberEventKind, deleteMemberEventKind,
  type MemberEventKind,
} from "@/lib/api";
import {
  EVENT_CATEGORY, EVENT_DATE_FIELDS, EVENT_TONES, MEMBERSHIP_STATUS, EXIT_REASONS,
} from "@/lib/constants";

function effectSummary(k: MemberEventKind): string {
  const parts: string[] = [];
  if (k.clears_exit) parts.push("limpa a saida (reativacao)");
  if (k.sets_status) parts.push(`situacao -> ${MEMBERSHIP_STATUS[k.sets_status]?.label ?? k.sets_status}`);
  if (k.sets_exit_reason) parts.push(`motivo -> ${EXIT_REASONS[k.sets_exit_reason] ?? k.sets_exit_reason}`);
  if (k.sets_baptism) parts.push("grava batismo");
  if (k.sets_date_field && k.sets_date_field !== "none") {
    parts.push(EVENT_DATE_FIELDS[k.sets_date_field] ?? k.sets_date_field);
  }
  return parts.length ? parts.join("; ") : "Somente registro";
}

type Form = {
  name: string;
  category: string;
  tone: string;
  sets_status: string;
  sets_exit_reason: string;
  clears_exit: boolean;
  sets_baptism: boolean;
  sets_date_field: string;
  is_active: boolean;
  sort_order: string;
};

const EMPTY: Form = {
  name: "", category: "outro", tone: "zinc", sets_status: "", sets_exit_reason: "",
  clears_exit: false, sets_baptism: false, sets_date_field: "none", is_active: true, sort_order: "0",
};

export function EventKindsSection({ canWrite }: { canWrite: boolean }) {
  const { toast } = useToast();
  const [items, setItems] = useState<MemberEventKind[] | null>(null);
  const [drawer, setDrawer] = useState<{ open: boolean; editing?: MemberEventKind }>({ open: false });
  const [form, setForm] = useState<Form>({ ...EMPTY });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const r = await listMemberEventKinds();
      setItems(r.event_kinds);
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao carregar eventos", "error");
      setItems([]);
    }
  }, [toast]);

  useEffect(() => { load(); }, [load]);

  const grouped = useMemo(() => {
    const arr = [...(items ?? [])];
    arr.sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name));
    return arr;
  }, [items]);

  function openCreate() {
    setForm({ ...EMPTY });
    setDrawer({ open: true });
  }

  function openEdit(k: MemberEventKind) {
    setForm({
      name: k.name, category: k.category, tone: k.tone,
      sets_status: k.sets_status ?? "", sets_exit_reason: k.sets_exit_reason ?? "",
      clears_exit: k.clears_exit, sets_baptism: k.sets_baptism,
      sets_date_field: k.sets_date_field || "none",
      is_active: k.is_active, sort_order: String(k.sort_order),
    });
    setDrawer({ open: true, editing: k });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    // Evento que leva a Inativo precisa dizer o motivo da inatividade.
    if (form.sets_status === "inactive" && !form.sets_exit_reason) {
      toast("Defina o motivo da inatividade para eventos que levam a 'Inativo'.", "error");
      return;
    }
    setSaving(true);
    const payload = {
      name: form.name,
      category: form.category,
      tone: form.tone,
      sets_status: form.sets_status,
      sets_exit_reason: form.sets_exit_reason,
      clears_exit: form.clears_exit,
      sets_baptism: form.sets_baptism,
      sets_date_field: form.sets_date_field,
      is_active: form.is_active,
      sort_order: Number(form.sort_order) || 0,
    };
    try {
      if (drawer.editing) {
        await updateMemberEventKind(drawer.editing.id, payload);
        toast("Evento atualizado.");
      } else {
        await createMemberEventKind(payload);
        toast("Evento criado.");
      }
      setDrawer({ open: false });
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar evento", "error");
    } finally {
      setSaving(false);
    }
  }

  async function remove(k: MemberEventKind) {
    if (!confirm(`Excluir o evento "${k.name}"?`)) return;
    try {
      await deleteMemberEventKind(k.id);
      toast("Evento excluido.");
      await load();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao excluir", "error");
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-xs text-zinc-400">
          A igreja define os eventos da vida eclesiastica e o que cada um movimenta no membro ao ser lancado.
        </p>
        {canWrite && (
          <Button className="h-8 text-sm" onClick={openCreate}><Plus className="h-4 w-4" /> Novo evento</Button>
        )}
      </div>

      <Card className="overflow-hidden p-0">
        {items === null ? (
          <div className="p-4"><SkeletonRows rows={5} /></div>
        ) : grouped.length === 0 ? (
          <EmptyState
            icon={<CalendarHeart className="h-10 w-10" />}
            title="Nenhum evento"
            description="Crie os eventos da vida eclesiastica (batismo, recepcao, ordenacao, baixa...)."
          />
        ) : (
          <Table>
            <THead>
              <TRow>
                <TH>Evento</TH><TH>Categoria</TH><TH>O que movimenta</TH>
                <TH>Situacao</TH><TH className="text-right">Acoes</TH>
              </TRow>
            </THead>
            <TBody>
              {grouped.map((k) => (
                <TRow key={k.id}>
                  <TD>
                    <div className="flex items-center gap-2">
                      <Badge tone={k.tone as Tone} className="text-[10px]">{k.name}</Badge>
                      <span className="text-[11px] text-zinc-400">{k.slug}</span>
                    </div>
                  </TD>
                  <TD className="text-sm">{EVENT_CATEGORY[k.category]?.label ?? k.category}</TD>
                  <TD className="text-xs text-zinc-500">{effectSummary(k)}</TD>
                  <TD>
                    <Badge tone={k.is_active ? "green" : "zinc"} className="text-[10px]">
                      {k.is_active ? "Ativo" : "Inativo"}
                    </Badge>
                  </TD>
                  <TD className="text-right">
                    {canWrite && (
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" className="h-7 px-2" onClick={() => openEdit(k)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" className="h-7 px-2 text-red-600" onClick={() => remove(k)}>
                          <Trash2 className="h-3.5 w-3.5" />
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

      <Drawer
        open={drawer.open}
        onClose={() => setDrawer({ open: false })}
        title={drawer.editing ? "Editar evento" : "Novo evento"}
      >
        <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nome" required className="sm:col-span-2">
            <Input
              required
              className="h-8 text-sm"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              placeholder="Ex: Batismo e profissao de fe"
            />
          </Field>
          <Field label="Categoria">
            <Select className="h-8 text-sm" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}>
              {Object.entries(EVENT_CATEGORY)
                .sort((a, b) => a[1].order - b[1].order)
                .map(([v, c]) => (<option key={v} value={v}>{c.label}</option>))}
            </Select>
          </Field>
          <Field label="Tom do badge">
            <Select className="h-8 text-sm" value={form.tone} onChange={(e) => setForm({ ...form, tone: e.target.value })}>
              {EVENT_TONES.map((t) => (<option key={t} value={t}>{t}</option>))}
            </Select>
          </Field>

          <Field label="Ao lancar, define a situacao">
            <Select className="h-8 text-sm" value={form.sets_status} onChange={(e) => setForm({ ...form, sets_status: e.target.value })}>
              <option value="">Nao altera</option>
              {Object.entries(MEMBERSHIP_STATUS).map(([v, s]) => (<option key={v} value={v}>{s.label}</option>))}
            </Select>
          </Field>
          <Field label="Motivo da inatividade">
            <Select className="h-8 text-sm" value={form.sets_exit_reason} onChange={(e) => setForm({ ...form, sets_exit_reason: e.target.value })}>
              <option value="">Nao define</option>
              {Object.entries(EXIT_REASONS).map(([v, l]) => (<option key={v} value={v}>{l}</option>))}
            </Select>
          </Field>
          <Field label="Atualiza data">
            <Select className="h-8 text-sm" value={form.sets_date_field} onChange={(e) => setForm({ ...form, sets_date_field: e.target.value })}>
              {Object.entries(EVENT_DATE_FIELDS).map(([v, l]) => (<option key={v} value={v}>{l}</option>))}
            </Select>
          </Field>
          <Field label="Ordem">
            <Input type="number" className="h-8 text-sm" value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: e.target.value })} />
          </Field>

          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <input type="checkbox" checked={form.sets_baptism} onChange={(e) => setForm({ ...form, sets_baptism: e.target.checked })} />
            Grava a data do batismo (e o local, se informado)
          </label>
          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <input type="checkbox" checked={form.clears_exit} onChange={(e) => setForm({ ...form, clears_exit: e.target.checked })} />
            Limpa motivo/data de saida (reativacao)
          </label>
          <label className="flex items-center gap-2 text-sm sm:col-span-2">
            <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            Evento ativo (disponivel para lancamento)
          </label>

          <div className="flex justify-end gap-2 sm:col-span-2">
            <Button type="button" variant="outline" className="h-8 text-sm" onClick={() => setDrawer({ open: false })}>Cancelar</Button>
            <Button type="submit" className="h-8 text-sm" disabled={saving}>{saving ? "Salvando..." : "Salvar"}</Button>
          </div>
        </form>
      </Drawer>
    </div>
  );
}
