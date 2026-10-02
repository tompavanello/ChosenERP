"use client";

import { useEffect, useState } from "react";
import { Users } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getMeMinistries, type MemberMinistry } from "@/lib/api";

const ROLE_LABELS: Record<string, string> = {
  volunteer: "Voluntário",
  coordinator: "Coordenador",
  leader: "Líder",
  member: "Membro",
};

export default function MinisteriosPage() {
  const [items, setItems] = useState<MemberMinistry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeMinistries()
      .then(setItems)
      .catch(() => setItems([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Meus ministérios</h1>

      {loading ? (
        <SkeletonRows rows={4} />
      ) : items.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">
            Você ainda não participa de nenhum ministério. Fale com a secretaria para ser incluído.
          </p>
        </Card>
      ) : (
        <div className="space-y-3">
          {items.map((m) => (
            <Card key={m.id}>
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-3">
                  <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand-strong)]">
                    <Users className="h-5 w-5" />
                  </span>
                  <div>
                    <p className="font-medium">{m.name}</p>
                    {m.leader_name && (
                      <p className="text-xs text-[var(--muted)]">Líder: {m.leader_name}</p>
                    )}
                  </div>
                </div>
                <Badge>{ROLE_LABELS[m.role] ?? m.role}</Badge>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
