"use client";

import { useEffect, useState } from "react";
import { MapPin } from "lucide-react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { dateTimePt, getMeEvents, type ChurchEvent } from "@/lib/api";

export default function AgendaPage() {
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const today = new Date();
    const from = today.toISOString().slice(0, 10);
    const to = new Date(today.getTime() + 90 * 86400000).toISOString().slice(0, 10);
    getMeEvents(from, to)
      .then(setEvents)
      .catch(() => setEvents([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={5} />;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Agenda da igreja</h1>
      {events.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum evento agendado.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {events.map((e) => (
            <Card key={e.id}>
              <div className="flex items-center justify-between gap-2">
                <p className="font-medium">{e.title}</p>
                {e.kind_name && (
                  <span className="rounded-full bg-[var(--brand-soft)] px-2 py-0.5 text-xs text-[var(--brand-strong)]">
                    {e.kind_name}
                  </span>
                )}
              </div>
              <p className="mt-1 text-sm text-[var(--muted)]">{dateTimePt(e.starts_at)}</p>
              {e.location && (
                <p className="mt-1 flex items-center gap-1 text-sm text-[var(--muted)]">
                  <MapPin className="h-3.5 w-3.5" /> {e.location}
                </p>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
