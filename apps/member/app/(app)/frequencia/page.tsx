"use client";

import { useEffect, useState } from "react";
import { Activity } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { datePt, getMeFrequency, type FrequencyEntry } from "@/lib/api";

const LABELS: Record<string, string> = {
  frequente: "Frequente",
  pouco_frequente: "Pouco frequente",
  nao_frequente: "Não frequente",
};

export default function FrequenciaPage() {
  const [items, setItems] = useState<FrequencyEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeFrequency()
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Minha frequência</h1>

      {loading ? (
        <SkeletonRows rows={3} />
      ) : items.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Sem registro de frequência no momento.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {items.map((f) => {
            const current = !f.ended_at;
            return (
              <Card key={f.id}>
                <div className="flex items-center justify-between gap-2">
                  <span className="flex items-center gap-2">
                    <Activity className="h-4 w-4 text-[var(--brand)]" />
                    <span className="font-medium">{LABELS[f.frequency] ?? f.frequency}</span>
                  </span>
                  {current && <Badge>Atual</Badge>}
                </div>
                <p className="mt-1 text-sm text-[var(--muted)]">
                  Desde {datePt(f.started_at)}
                  {f.ended_at ? ` até ${datePt(f.ended_at)}` : ""}
                </p>
                {f.notes && <p className="mt-1 text-sm text-[var(--muted)]">{f.notes}</p>}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
