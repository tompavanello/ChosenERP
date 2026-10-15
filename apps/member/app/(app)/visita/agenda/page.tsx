"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { dateTimePt, getVisitorEvents, type ChurchEvent } from "@/lib/api";

export default function VisitorAgendaPage() {
  const [events, setEvents] = useState<ChurchEvent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getVisitorEvents()
      .then(setEvents)
      .catch(() => setEvents([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={4} />;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Agenda</h1>
      {events.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum evento programado.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {events.map((e) => (
            <Card key={e.id}>
              <p className="font-medium">{e.title}</p>
              <p className="mt-1 text-sm text-[var(--muted)]">
                {dateTimePt(e.starts_at)}
                {e.location ? ` · ${e.location}` : ""}
              </p>
              {e.notes && <p className="mt-1 text-sm text-[var(--muted)]">{e.notes}</p>}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
