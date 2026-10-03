"use client";

import { useEffect, useState } from "react";
import { CalendarClock, MapPin, TrendingUp, Users } from "lucide-react";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SkeletonRows } from "@/components/ui/skeleton";
import { datePt, getMeLedGroups, type MyGroup } from "@/lib/api";

const WEEKDAYS = ["Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"];
const KIND_LABELS: Record<string, string> = { cell: "Célula", ebd: "EBD", family: "Família" };
const ROLE_LABELS: Record<string, string> = {
  member: "Membro",
  host: "Anfitrião",
  secretary: "Secretário",
  leader: "Líder",
};

export default function LiderPage() {
  const [groups, setGroups] = useState<MyGroup[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getMeLedGroups()
      .then(setGroups)
      .catch(() => setGroups([]))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div className="space-y-4">
      <h1 className="text-lg font-bold">Área do líder</h1>

      {loading ? (
        <SkeletonRows rows={4} />
      ) : groups.length === 0 ? (
        <Card>
          <p className="text-sm text-[var(--muted)]">Você não lidera nenhum grupo no momento.</p>
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
                  <Badge>Líder</Badge>
                </div>

                <dl className="mt-3 space-y-1.5 text-sm">
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

                {g.attendance && g.attendance.length > 0 && (
                  <div className="mt-4 border-t border-[var(--line)] pt-3">
                    <p className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-[var(--muted)]">
                      <TrendingUp className="h-3.5 w-3.5" /> Últimos encontros
                    </p>
                    <ul className="space-y-1 text-sm">
                      {g.attendance.map((d) => (
                        <li key={d.date} className="flex items-center justify-between">
                          <span className="text-[var(--muted)]">{datePt(d.date)}</span>
                          <span className="tnum text-xs">
                            <span className="text-emerald-600">{d.present} presente(s)</span>
                            {d.absent > 0 && <span className="text-[var(--muted)]"> · {d.absent} falta(s)</span>}
                          </span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
