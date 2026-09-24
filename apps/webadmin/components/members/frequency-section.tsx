"use client";

import { useCallback, useEffect, useState } from "react";
import { Activity, Plus } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Field, Input, Select, Textarea } from "@/components/ui/input";
import { EmptyState, SkeletonRows } from "@/components/ui/skeleton";
import { useToast } from "@/components/ui/toast";
import { listMemberFrequency, setMemberFrequency, type FrequencyEntry } from "@/lib/api";
import { FREQUENCY } from "@/lib/constants";
import { datePt } from "@/lib/format";

export function FrequencySection({ memberId, canWrite }: { memberId: string; canWrite: boolean }) {
  const { toast } = useToast();
  const [items, setItems] = useState<FrequencyEntry[] | null>(null);
  const [form, setForm] = useState({ frequency: "frequente", started_at: new Date().toISOString().slice(0, 10), notes: "" });
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    const r = await listMemberFrequency(memberId);
    setItems(r.frequency);
  }, [memberId]);

  useEffect(() => {
    load().catch(() => toast("Erro ao carregar a frequência", "error"));
  }, [load, toast]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await setMemberFrequency(memberId, { frequency: form.frequency, started_at: form.started_at, notes: form.notes });
      setForm({ frequency: form.frequency, started_at: form.started_at, notes: "" });
      await load();
      toast("Frequência atualizada.");
    } catch (err) {
      toast(err instanceof Error ? err.message : "Erro ao salvar", "error");
    } finally {
      setSaving(false);
    }
  }

  const atual = items?.find((f) => !f.ended_at);

  return (
    <div className="space-y-4">
      <Card className="p-4">
        <div className="flex items-center gap-2">
          <Activity className="h-4 w-4 text-sky-600" />
          <h3 className="text-sm font-semibold">Frequência atual</h3>
          {atual ? (
            <Badge tone={(FREQUENCY[atual.frequency]?.tone as "green") ?? "zinc"}>
              {FREQUENCY[atual.frequency]?.label ?? atual.frequency}
            </Badge>
          ) : (
            <span className="text-xs text-zinc-400">não classificada</span>
          )}
        </div>
      </Card>

      {canWrite && (
        <Card className="p-4">
          <form onSubmit={submit} className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <Field label="Nova frequência">
              <Select className="h-8 text-sm" value={form.frequency} onChange={(e) => setForm({ ...form, frequency: e.target.value })}>
                {Object.entries(FREQUENCY).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
              </Select>
            </Field>
            <Field label="A partir de">
              <Input type="date" className="h-8 text-sm" value={form.started_at} onChange={(e) => setForm({ ...form, started_at: e.target.value })} />
            </Field>
            <Field label="Observação" className="sm:col-span-3">
              <Textarea rows={2} className="text-sm" value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
            </Field>
            <div className="sm:col-span-3">
              <Button type="submit" className="h-8 text-sm" disabled={saving}><Plus className="h-4 w-4" /> {saving ? "Salvando..." : "Registrar"}</Button>
            </div>
          </form>
        </Card>
      )}

      <Card className="p-4">
        <h3 className="mb-3 text-sm font-semibold">Histórico de frequência</h3>
        {items === null ? (
          <SkeletonRows rows={3} />
        ) : items.length === 0 ? (
          <EmptyState icon={<Activity className="h-8 w-8" />} title="Sem histórico" description="A frequência do membro aparece aqui." />
        ) : (
          <ol className="relative space-y-3 border-l border-zinc-200 pl-4 dark:border-zinc-800">
            {items.map((f) => {
              const st = FREQUENCY[f.frequency] ?? { label: f.frequency, tone: "zinc" };
              return (
                <li key={f.id} className="relative">
                  <span className={`absolute -left-[21px] top-1.5 h-2 w-2 rounded-full ${f.ended_at ? "bg-zinc-400" : "bg-emerald-500"}`} />
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge tone={(st.tone as "green") ?? "zinc"} className="text-[10px]">{st.label}</Badge>
                    <span className="text-xs text-zinc-400">
                      {datePt(f.started_at)} {f.ended_at ? `→ ${datePt(f.ended_at)}` : "(vigente)"}
                    </span>
                  </div>
                  {f.notes && <p className="mt-0.5 text-sm">{f.notes}</p>}
                </li>
              );
            })}
          </ol>
        )}
      </Card>
    </div>
  );
}
