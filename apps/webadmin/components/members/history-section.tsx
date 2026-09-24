"use client";

import { useCallback, useEffect, useState } from "react";
import { History, Plus } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Field, Select, Input, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { listMemberHistory, addMemberHistory, type MemberHistory } from "@/lib/api";
import { MEMBER_HISTORY_KINDS } from "@/lib/constants";
import { dateTimePt } from "@/lib/format";

// Tipos que a secretaria lanca a mao (os automaticos vem das mudancas de
// situacao - ver internal/members/members.go).
const MANUAL_KINDS = [
  "profissao_fe",
  "batismo_infantil",
  "recebido_jurisdicao",
  "recebido_transferencia",
  "desligamento",
  "abandono",
  "outro",
];

export function HistorySection({ memberId, canWrite }: { memberId: string; canWrite: boolean }) {
  const { toast } = useToast();
  const [items, setItems] = useState<MemberHistory[] | null>(null);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({ kind: "profissao_fe", notes: "", occurred_at: "" });

  const load = useCallback(async () => {
    const r = await listMemberHistory(memberId);
    setItems(r.history);
  }, [memberId]);

  useEffect(() => {
    load().catch(() => toast("Erro ao carregar o historico", "error"));
  }, [load, toast]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await addMemberHistory(memberId, {
        kind: form.kind,
        notes: form.notes,
        occurred_at: form.occurred_at,
      });
      setForm({ kind: "profissao_fe", notes: "", occurred_at: "" });
      await load();
      toast("Evento registrado no historico.");
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
                {MANUAL_KINDS.map((k) => (
                  <option key={k} value={k}>{MEMBER_HISTORY_KINDS[k] ?? k}</option>
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
            <Field label="Observacao" className="sm:col-span-3">
              <Textarea
                rows={2}
                value={form.notes}
                onChange={(e) => setForm({ ...form, notes: e.target.value })}
                placeholder="Ex: batizado pelo Pastor Joao na Sede"
              />
            </Field>
            <div className="sm:col-span-3">
              <Button type="submit" className="h-8 text-sm" disabled={saving}>
                {saving ? "Registrando..." : "Registrar"}
              </Button>
            </div>
          </form>
        </Card>
      )}

      <Card className="p-3">
        <div className="mb-3 flex items-center gap-2">
          <History className="h-4 w-4 text-sky-600" />
          <h3 className="text-sm font-semibold">Historico eclesiastico</h3>
        </div>
        {items === null ? (
          <SkeletonRows rows={3} />
        ) : items.length === 0 ? (
          <EmptyState
            icon={<History className="h-8 w-8" />}
            title="Sem eventos"
            description="Entradas, batismos, transferencias e baixas aparecem aqui."
          />
        ) : (
          <ol className="relative space-y-3 border-l border-zinc-200 pl-4 dark:border-zinc-800">
            {items.map((h) => (
              <li key={h.id} className="relative">
                <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-sky-500" />
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone="sky" className="text-[10px]">
                    {MEMBER_HISTORY_KINDS[h.kind] ?? h.kind}
                  </Badge>
                  <span className="text-xs text-zinc-400">{dateTimePt(h.occurred_at)}</span>
                </div>
                {h.notes && <p className="mt-0.5 text-sm">{h.notes}</p>}
              </li>
            ))}
          </ol>
        )}
      </Card>
    </div>
  );
}
