"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { History, Plus } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge, type Tone } from "@/components/ui/badge";
import { Field, Select, Input, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import {
  listMemberHistory, addMemberHistory, listMemberEventKinds,
  type MemberHistory, type MemberEventKind,
} from "@/lib/api";
import { MEMBER_HISTORY_KINDS, MEMBERSHIP_STATUS, EXIT_REASONS, EVENT_CATEGORY } from "@/lib/constants";
import { dateTimePt } from "@/lib/format";

/** Categorias que o sistema lanca sozinho e nao devem aparecer no formulario. */
const AUTO_CATEGORIES = new Set(["sistema"]);

/** Resumo legivel do que o evento movimenta, mostrado antes de lancar. */
function effectSummary(k: MemberEventKind): string {
  const parts: string[] = [];
  if (k.sets_status) parts.push(`situacao -> ${MEMBERSHIP_STATUS[k.sets_status]?.label ?? k.sets_status}`);
  if (k.sets_exit_reason) parts.push(`motivo -> ${EXIT_REASONS[k.sets_exit_reason] ?? k.sets_exit_reason}`);
  if (k.sets_date_field === "baptism") parts.push("grava a data do batismo");
  if (k.sets_date_field === "joined_at") parts.push("atualiza 'membro desde'");
  if (k.sets_date_field === "marriage_date") parts.push("atualiza data de casamento");
  return parts.join("; ");
}

export function HistorySection({
  memberId,
  canWrite,
  onChanged,
}: {
  memberId: string;
  canWrite: boolean;
  onChanged?: () => void;
}) {
  const { toast } = useToast();
  const [items, setItems] = useState<MemberHistory[] | null>(null);
  const [kinds, setKinds] = useState<MemberEventKind[]>([]);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ kind: "", notes: "", occurred_at: "", baptism_location: "" });

  const load = useCallback(async () => {
    const [r, k] = await Promise.all([listMemberHistory(memberId), listMemberEventKinds()]);
    setItems(r.history);
    const manuais = k.event_kinds.filter((e) => e.is_active && !AUTO_CATEGORIES.has(e.category));
    setKinds(manuais);
    if (manuais.length && !form.kind) setForm((f) => ({ ...f, kind: manuais[0].id }));
  }, [memberId, form.kind]);

  useEffect(() => {
    load().catch(() => toast("Erro ao carregar a vida eclesiastica", "error"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [memberId]);

  const grouped = useMemo(() => {
    const map = new Map<string, MemberEventKind[]>();
    for (const k of kinds) {
      const arr = map.get(k.category) ?? [];
      arr.push(k);
      map.set(k.category, arr);
    }
    return [...map.entries()].sort(
      (a, b) => (EVENT_CATEGORY[a[0]]?.order ?? 99) - (EVENT_CATEGORY[b[0]]?.order ?? 99),
    );
  }, [kinds]);

  const selected = kinds.find((k) => k.id === form.kind);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.kind) return;
    setSaving(true);
    try {
      await addMemberHistory(memberId, {
        event_kind_id: form.kind,
        notes: form.notes,
        occurred_at: form.occurred_at,
        baptism_location: form.baptism_location,
      });
      setForm((f) => ({ ...f, notes: "", occurred_at: "", baptism_location: "" }));
      await load();
      onChanged?.();
      toast("Evento registrado na vida eclesiastica.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao registrar evento", "error");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-4">
      {canWrite && (
        <Card className="p-3">
          <div className="mb-3 flex items-center gap-2">
            <Plus className="h-4 w-4 text-sky-600" />
            <h3 className="text-sm font-semibold">Registrar evento</h3>
          </div>
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Field label="Tipo de evento">
              <Select
                className="h-8 text-sm"
                value={form.kind}
                onChange={(e) => setForm({ ...form, kind: e.target.value })}
              >
                {grouped.length === 0 && <option value="">Nenhum evento disponivel</option>}
                {grouped.map(([cat, list]) => (
                  <optgroup key={cat} label={EVENT_CATEGORY[cat]?.label ?? cat}>
                    {list.map((k) => (
                      <option key={k.id} value={k.id}>{k.name}</option>
                    ))}
                  </optgroup>
                ))}
              </Select>
            </Field>
            <Field label="Data e hora" hint="Em branco usa agora.">
              <Input
                type="datetime-local"
                className="h-8 text-sm"
                value={form.occurred_at}
                onChange={(e) => setForm({ ...form, occurred_at: e.target.value })}
              />
            </Field>
            {selected?.sets_date_field === "baptism" && (
              <Field label="Local do batismo">
                <Input
                  className="h-8 text-sm"
                  value={form.baptism_location}
                  onChange={(e) => setForm({ ...form, baptism_location: e.target.value })}
                  placeholder="Ex: Igreja Sede Matriz"
                />
              </Field>
            )}
            {selected && effectSummary(selected) && (
              <p className="rounded-md bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:bg-sky-950/40 dark:text-sky-300 sm:col-span-3">
                Ao lancar: {effectSummary(selected)}.
              </p>
            )}
            <Field label="Observacao" className="sm:col-span-3">
              <Textarea
                rows={2}
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                placeholder="Ex: batizado pelo Pastor Joao na Sede"
              />
            </Field>
            <div className="sm:col-span-3">
              <Button type="submit" className="h-8 text-sm" disabled={saving || !form.kind}>
                {saving ? "Registrando..." : "Registrar"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      <Card className="p-3">
        <div className="mb-3 flex items-center gap-2">
          <History className="h-4 w-4 text-sky-600" />
          <h3 className="text-sm font-semibold">Linha do tempo</h3>
        </div>
        {items === null ? (
          <SkeletonRows rows={3} />
        ) : items.length === 0 ? (
          <EmptyState
            icon={<History className="h-8 w-8" />}
            description="Batismos, recepcoes, ordenacoes, transferencias e baixas aparecem aqui."
            title="Sem eventos"
          />
        ) : (
          <ol className="relative space-y-3 border-l border-zinc-200 pl-4 dark:border-zinc-800">
            {items.map((h) => (
              <li key={h.id} className="relative">
                <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-sky-500" />
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={(h.event_tone as Tone) ?? "sky"} className="text-[10px]">
                    {h.event_name ?? MEMBER_HISTORY_KINDS[h.kind] ?? h.kind}
                  </Badge>
                  <span className="text-xs text-zinc-400">{dateTimePt(h.occurred_at)}</span>
                </div>
                {h.notes && <p className="mt-0.5 text-sm whitespace-pre-wrap">{h.notes}</p>}
              </li>
            ))}
          </ol>
        )}
      </Card>
    </div>
  );
}
