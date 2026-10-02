"use client";

import { useEffect, useState } from "react";
import { Card } from "@/components/ui/card";
import { SkeletonRows } from "@/components/ui/skeleton";
import { datePt, getMeAnnouncements, type Announcement } from "@/lib/api";

export default function AvisosPage() {
  const [anns, setAnns] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeAnnouncements()
      .then(setAnns)
      .catch(() => setAnns([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <SkeletonRows rows={4} />;

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Avisos</h1>
      {anns.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Nenhum aviso no momento.</p>
        </Card>
      ) : (
        <div className="space-y-3">
          {anns.map((a) => (
            <Card key={a.id}>
              <div className="flex items-center justify-between gap-2">
                <p className="font-medium">{a.title}</p>
                <span className="text-xs text-[var(--muted)]">{datePt(a.published_at)}</span>
              </div>
              <p className="mt-2 whitespace-pre-line text-sm text-[var(--muted)]">{a.body}</p>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
