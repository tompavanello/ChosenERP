"use client";

import Link from "next/link";
import { Avatar } from "@/components/ui/avatar";
import { Badge, type Tone } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { MEMBERSHIP_STATUS, JOURNEY_STAGES, RELATION_LABELS } from "@/lib/constants";
import type { Person } from "@/lib/api";

export function personStatusTone(person: Person): Tone {
  if (person.type === "visitor") {
    const stage = person.journey_stage ?? "";
    return (JOURNEY_STAGES[stage]?.tone as Tone) ?? "amber";
  }
  if (person.type === "benefactor") return "pink";
  const st = MEMBERSHIP_STATUS[person.membership_status ?? ""] ?? { tone: "zinc" };
  return (st.tone as Tone) ?? "zinc";
}

export function personStatusLabel(person: Person): string {
  if (person.type === "visitor") {
    return JOURNEY_STAGES[person.journey_stage ?? ""]?.label ?? person.journey_stage ?? "Visitante";
  }
  if (person.type === "benefactor") return "Benfeitor";
  const st = MEMBERSHIP_STATUS[person.membership_status ?? ""] ?? { label: person.membership_status ?? "" };
  return st.label ?? person.membership_status ?? "-";
}

export function PersonCard({
  person,
  detailHref,
  actions,
}: {
  person: Person;
  detailHref?: string;
  actions?: React.ReactNode;
}) {
  const tone = personStatusTone(person);
  const label = personStatusLabel(person);
  // `office` e a coluna legada (o catalogo de cargos vive em /api/v1/cargos e o
  // vinculo com mandato e por membro) - exibida como veio, sem dicionario fixo.
  const subtitle = person.type === "member"
    ? person.profession
      ? person.profession
      : person.branch_name
        ? `Filial ${person.branch_name}`
        : "Membro"
    : person.type === "visitor"
      ? person.source
        ? `Chegada: ${person.source}`
        : "Visitante"
      : person.notes
        ? person.notes.slice(0, 40)
        : "Benfeitor";

  return (
    <Card className="p-3 group">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <Avatar name={person.full_name} size="md" />
          <div className="min-w-0">
            {detailHref ? (
              <Link href={detailHref} className="font-semibold truncate block group-hover:text-sky-700 transition-colors">
                {person.full_name || "-"}
              </Link>
            ) : (
              <span className="font-semibold truncate block">{person.full_name || "-"}</span>
            )}
            <p className="text-xs text-zinc-400 truncate max-w-[160px]">{subtitle}</p>
          </div>
        </div>
        <Badge tone={tone} className="text-[10px]">{label}</Badge>
      </div>
      {actions && <div className="flex justify-end gap-1 pt-2 border-t border-zinc-100 mt-2">{actions}</div>}
    </Card>
  );
}

export const RELATION_KINDS = ["spouse", "parent", "child", "discipler", "disciple", "relative"] as const;

export function relationLabel(kind: string): string {
  return RELATION_LABELS[kind] ?? kind;
}
