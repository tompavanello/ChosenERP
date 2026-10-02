"use client";

import { useEffect, useState } from "react";
import { Check, Clock, MapPin, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { dateTimePt, getMeRosters, respondRoster, type MyRoster } from "@/lib/api";

const STATUS_LABELS: Record<string, string> = {
  confirmado: "Confirmado",
  recusado: "Não poderei",
  convidado: "Aguardando confirmação",
};

export default function EscalasPage() {
  const [items, setItems] = useState<MyRoster[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const from = new Date().toISOString().slice(0, 10);
    getMeRosters(from)
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);

  async function respond(m: MyRoster, status: "confirmado" | "recusado") {
    setBusy(m.assignment_id);
    setError(null);
    try {
      await respondRoster(m.assignment_id, status);
      setItems((prev) => prev.map((x) => (x.assignment_id === m.assignment_id ? { ...x, status } : x)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Não foi possível responder.");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Minhas escalas</h1>

      {error && <p className="text-sm text-red-600">{error}</p>}

      {loading ? (
        <SkeletonRows rows={4} />
      ) : items.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Você não está escalado em nenhum compromisso próximo.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {items.map((m) => (
            <Card key={m.assignment_id}>
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-medium">{m.title}</p>
                  {m.ministry_name && <p className="text-xs text-[var(--muted)]">{m.ministry_name}</p>}
                </div>
                <Badge>{STATUS_LABELS[m.status] ?? m.status}</Badge>
              </div>

              <p className="mt-2 flex items-center gap-1.5 text-sm text-[var(--muted)]">
                <Clock className="h-3.5 w-3.5" /> {dateTimePt(m.starts_at)}
              </p>
              {m.location && (
                <p className="mt-1 flex items-center gap-1.5 text-sm text-[var(--muted)]">
                  <MapPin className="h-3.5 w-3.5" /> {m.location}
                </p>
              )}
              {m.role && <p className="mt-1 text-sm text-[var(--muted)]">Função: {m.role}</p>}

              {m.status === "convidado" ? (
                <div className="mt-3 flex gap-2">
                  <Button className="flex-1" disabled={busy === m.assignment_id} onClick={() => respond(m, "confirmado")}>
                    <Check className="h-4 w-4" /> Vou participar
                  </Button>
                  <Button variant="outline" className="flex-1" disabled={busy === m.assignment_id} onClick={() => respond(m, "recusado")}>
                    <X className="h-4 w-4" /> Não poderei
                  </Button>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => respond(m, m.status === "confirmado" ? "recusado" : "confirmado")}
                  disabled={busy === m.assignment_id}
                  className="mt-3 text-sm text-[var(--brand)] disabled:opacity-50"
                >
                  {m.status === "confirmado" ? "Não poderei mais participar" : "Mudar para vou participar"}
                </button>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
