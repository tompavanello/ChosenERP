"use client";

import { useEffect, useState } from "react";
import { CalendarClock, MapPin, UserCog, Users } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { getMeGroups, type MyGroup } from "@/lib/api";

const WEEKDAYS = ["Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"];
const ROLE_LABELS: Record<string, string> = {
  member: "Membro",
  host: "Anfitrião",
  secretary: "Secretário",
  leader: "Líder",
};
const KIND_LABELS: Record<string, string> = { cell: "Célula", ebd: "EBD", family: "Família" };

export default function GdPage() {
  const [groups, setGroups] = useState<MyGroup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeGroups()
      .then(setGroups)
      .catch(() => setGroups([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Meu GD</h1>

      {loading ? (
        <SkeletonRows rows={4} />
      ) : groups.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">
            Você ainda não participa de um grupo. Fale com a secretaria para ser incluído.
          </p>
        </Card>
      ) : (
        <div className="space-y-4">
          {groups.map((g) => {
            const meeting = [g.weekday != null ? WEEKDAYS[g.weekday] : null, g.meeting_time?.slice(0, 5)]
              .filter(Boolean)
              .join(" · ");
            return (
              <Card key={g.id}>
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <p className="font-semibold">{g.name}</p>
                    <p className="text-xs text-[var(--muted)]">{KIND_LABELS[g.kind] ?? g.kind}</p>
                  </div>
                  <Badge>{ROLE_LABELS[g.my_role] ?? g.my_role}</Badge>
                </div>

                <dl className="mt-3 space-y-1.5 text-sm">
                  {g.leader_name && (
                    <div className="flex items-center gap-2 text-[var(--muted)]">
                      <UserCog className="h-4 w-4" /> Líder: <span className="text-[var(--ink)]">{g.leader_name}</span>
                    </div>
                  )}
                  {meeting && (
                    <div className="flex items-center gap-2 text-[var(--muted)]">
                      <CalendarClock className="h-4 w-4" /> {meeting}
                    </div>
                  )}
                  {g.address && (
                    <div className="flex items-center gap-2 text-[var(--muted)]">
                      <MapPin className="h-4 w-4" /> <span className="text-[var(--ink)]">{g.address}</span>
                    </div>
                  )}
                </dl>

                <div className="mt-4 border-t border-[var(--line)] pt-3">
                  <p className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-[var(--muted)]">
                    <Users className="h-3.5 w-3.5" /> Participantes ({g.members.length})
                  </p>
                  {g.members.length === 0 ? (
                    <p className="text-sm text-[var(--muted)]">Nenhum participante cadastrado.</p>
                  ) : (
                    <ul className="space-y-1 text-sm">
                      {g.members.map((m) => (
                        <li key={m.member_id} className="flex items-center justify-between">
                          <span>{m.member_name}</span>
                          {m.role !== "member" && (
                            <span className="text-xs text-[var(--muted)]">{ROLE_LABELS[m.role] ?? m.role}</span>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
